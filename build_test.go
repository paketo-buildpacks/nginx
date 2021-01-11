package nginx_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/nginx/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {

	var (
		Expect = NewWithT(t).Expect

		layersDir    string
		cnbPath      string
		workspaceDir string

		entryResolver     *fakes.EntryResolver
		dependencyService *fakes.DependencyService
		profileDWriter    *fakes.ProfileDWriter
		calculator        *fakes.Calculator

		clock     chronos.Clock
		timeStamp time.Time

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbPath, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workspaceDir, err = ioutil.TempDir("", "workspace")
		Expect(err).NotTo(HaveOccurred())

		entryResolver = &fakes.EntryResolver{}

		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "nginx",
			Metadata: map[string]interface{}{
				"version": "1.17.*",
				"launch":  true,
			},
		}

		dependencyService = &fakes.DependencyService{}
		dependencyService.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:           "nginx",
			SHA256:       "some-sha",
			Source:       "some-source",
			SourceSHA256: "some-source-sha",
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "1.17.2",
		}

		profileDWriter = &fakes.ProfileDWriter{}
		calculator = &fakes.Calculator{}

		calculator.SumCall.Returns.String = "some-bin-sha"

		// create fake configure binary
		Expect(os.Mkdir(filepath.Join(cnbPath, "bin"), os.ModePerm)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(cnbPath, "bin", "configure"), []byte("binary-contents"), 0755)).To(Succeed())

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		logEmitter := nginx.NewLogEmitter(bytes.NewBuffer(nil))

		build = nginx.Build(entryResolver, dependencyService, profileDWriter, calculator, logEmitter, clock)
	})

	it("does a build", func() {
		result, err := build(packit.BuildContext{
			CNBPath:    cnbPath,
			WorkingDir: workspaceDir,
			Stack:      "some-stack",
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "nginx",
						Metadata: map[string]interface{}{
							"version": "1.17.*",
							"launch":  true,
						},
					},
				},
			},
			Layers: packit.Layers{Path: layersDir},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "nginx",
						Metadata: map[string]interface{}{
							"version": "1.17.*",
							"launch":  true,
						},
					},
				},
			},
			Layers: []packit.Layer{
				{
					Name: "nginx",
					Path: filepath.Join(layersDir, "nginx"),
					SharedEnv: packit.Environment{
						"PATH.append": filepath.Join(layersDir, "nginx", "sbin"),
						"PATH.delim":  ":",
					},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    true,
					Cache:     false,
					Metadata: map[string]interface{}{
						nginx.DepKey:          "some-sha",
						nginx.ConfigureBinKey: "some-bin-sha",
						"built_at":            timeStamp.Format(time.RFC3339Nano),
					},
				},
			},
			Launch: packit.LaunchMetadata{
				Processes: []packit.Process{
					{
						Type:    "web",
						Command: fmt.Sprintf(`nginx -p $PWD -c "%s"`, filepath.Join(workspaceDir, "nginx.conf")),
					},
				},
			},
		}))

		Expect(filepath.Join(layersDir, "nginx")).To(BeADirectory())
		Expect(filepath.Join(layersDir, "nginx", "bin", "configure")).To(BeAnExistingFile())
		Expect(profileDWriter.WriteCall.Receives.LayerDir).To(Equal(filepath.Join(layersDir, "nginx")))
		Expect(profileDWriter.WriteCall.Receives.ScriptName).To(Equal("configure.sh"))
		expectedScript := fmt.Sprintf(`configure "%s" "%s" "%s"`, filepath.Join(workspaceDir, "nginx.conf"), filepath.Join(workspaceDir, "modules"), filepath.Join(layersDir, "nginx", "modules"))
		Expect(profileDWriter.WriteCall.Receives.ScriptContents).To(Equal(expectedScript))
		Expect(filepath.Join(workspaceDir, "logs")).To(BeADirectory())

		Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
			{
				Name: "nginx",
				Metadata: map[string]interface{}{
					"version": "1.17.*",
					"launch":  true,
				},
			},
		}))

		Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
		Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("nginx"))
		Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("1.17.*"))
		Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyService.InstallCall.Receives.Dependency).To(Equal(
			postal.Dependency{
				ID:           "nginx",
				SHA256:       "some-sha",
				Source:       "some-source",
				SourceSHA256: "some-source-sha",
				Stacks:       []string{"some-stack"},
				URI:          "some-uri",
				Version:      "1.17.2",
			},
		))
		Expect(dependencyService.InstallCall.Receives.CnbPath).To(Equal(cnbPath))
		Expect(dependencyService.InstallCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "nginx")))
		Expect(calculator.SumCall.CallCount).To(Equal(1))

	})

	context("when rebuilding a layer", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(layersDir, "nginx.toml"), []byte(fmt.Sprintf(`launch = true
[metadata]
			dependency-sha = "some-sha"
			configure-bin-sha = "some-bin-sha"
			built_at = "%s"
			`, timeStamp.Format(time.RFC3339Nano))), 0644)
			Expect(err).NotTo(HaveOccurred())
		})
		it("does not re-build the nginx layer", func() {
			result, err := build(packit.BuildContext{
				CNBPath:    cnbPath,
				WorkingDir: workspaceDir,
				Stack:      "some-stack",
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "nginx",
							Metadata: map[string]interface{}{
								"version": "1.17.*",
								"launch":  true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "nginx",
							Metadata: map[string]interface{}{
								"version": "1.17.*",
								"launch":  true,
							},
						},
					},
				},
				Layers: []packit.Layer{
					{
						Name:      "nginx",
						Path:      filepath.Join(layersDir, "nginx"),
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{},
						Build:     false,
						Launch:    true,
						Cache:     false,
						Metadata: map[string]interface{}{
							nginx.DepKey:          "some-sha",
							nginx.ConfigureBinKey: "some-bin-sha",
							"built_at":            timeStamp.Format(time.RFC3339Nano),
						},
					},
				},
				Launch: packit.LaunchMetadata{
					Processes: []packit.Process{
						{
							Type:    "web",
							Command: fmt.Sprintf(`nginx -p $PWD -c "%s"`, filepath.Join(workspaceDir, "nginx.conf")),
						},
					},
				},
			}))

			Expect(dependencyService.InstallCall.CallCount).To(Equal(0))
			Expect(profileDWriter.WriteCall.CallCount).To(Equal(0))

		})
	})

	context("failure cases", func() {
		context("unable to create log directory", func() {
			it.Before(func() {
				Expect(os.Chmod(workspaceDir, 0000))
			})

			it.After(func() {
				Expect(os.Chmod(workspaceDir, os.ModePerm))
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
				})

				Expect(err).To(HaveOccurred())
				logsDir := filepath.Join(workspaceDir, "logs")
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to create logs dir : mkdir %s", logsDir))))
			})
		})

		context("configure bin checksum fails", func() {
			it.Before(func() {
				calculator.SumCall.Returns.Error = errors.New("some-error")
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
					Layers:     packit.Layers{Path: layersDir},
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("checksum failed for file")))
			})
		})

		context("when the profile.d cannot be written", func() {
			it.Before(func() {
				profileDWriter.WriteCall.Returns.Error = errors.New("failed to write profile.d")
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
					Layers:     packit.Layers{Path: layersDir},
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("failed to write profile.d"))
			})
		})
	})
}

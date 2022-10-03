package nginx_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/nginx/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		cnbPath    string

		versionParser *fakes.VersionParser
		detect        packit.DetectFunc
	)

	it.Before(func() {
		workingDir = t.TempDir()
		cnbPath = t.TempDir()

		versionParser = &fakes.VersionParser{}
		versionParser.ResolveVersionCall.Returns.ResultVersion = "1.19.*"

		detect = nginx.Detect(nginx.Configuration{NGINXConfLocation: "./nginx.conf", WebServerRoot: "./public"}, versionParser)
	})

	it("returns a plan that provides nginx", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{Name: "nginx"},
			},
		}))
	})

	context("$BP_WEB_SERVER is set for a no-config nginx build", func() {
		it.Before(func() {
			detect = nginx.Detect(nginx.Configuration{WebServer: "nginx"}, versionParser)
		})

		it("requires and provides nginx", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
				CNBPath:    cnbPath,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "nginx"},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "nginx",
						Metadata: nginx.BuildPlanMetadata{
							Version:       "1.19.*",
							VersionSource: "buildpack.toml",
							Launch:        true,
						},
					},
				},
			}))
		})
	})

	context("nginx.conf is present", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`conf`), 0600)).To(Succeed())
		})

		context("when version is set via BP_NGINX_VERSION", func() {
			it.Before(func() {
				detect = nginx.Detect(nginx.Configuration{NGINXVersion: "mainline"}, versionParser)
				versionParser.ResolveVersionCall.Returns.ResultVersion = "1.19.*"
				versionParser.ResolveVersionCall.Returns.Err = nil
			})

			it("requires the given constraint in buildpack.yml", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
					CNBPath:    cnbPath,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{
						{Name: "nginx"},
					},
					Requires: []packit.BuildPlanRequirement{
						{
							Name: "nginx",
							Metadata: nginx.BuildPlanMetadata{
								Version:       "1.19.*",
								VersionSource: "BP_NGINX_VERSION",
								Launch:        true,
							},
						},
					},
				}))

				Expect(versionParser.ResolveVersionCall.Receives.CnbPath).To(Equal(cnbPath))
				Expect(versionParser.ResolveVersionCall.Receives.Version).To(Equal("mainline"))

			})
		})

		context("and BP_LIVE_RELOAD_ENABLED=true in the build environment", func() {
			it.Before(func() {
				detect = nginx.Detect(nginx.Configuration{LiveReloadEnabled: true}, versionParser)
			})

			it("requires watchexec at launch time", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan.Requires).To(Equal([]packit.BuildPlanRequirement{
					{
						Name: "nginx",
						Metadata: nginx.BuildPlanMetadata{
							Version:       "1.19.*",
							VersionSource: "buildpack.toml",
							Launch:        true,
						},
					},
					{
						Name: "watchexec",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
				},
				))
			})
		})

		context("when there is a buildpack.yml", func() {
			it.Before(func() {
				versionParser.ResolveVersionCall.Returns.ResultVersion = "1.2.3"
				versionParser.ResolveVersionCall.Returns.Err = nil
				versionParser.ParseYmlCall.Returns.Exists = true
				versionParser.ParseYmlCall.Returns.YmlVersion = "1.2.3"
			})

			it("requires the given constraint in buildpack.yml", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{
						{Name: "nginx"},
					},
					Requires: []packit.BuildPlanRequirement{
						{
							Name: "nginx",
							Metadata: nginx.BuildPlanMetadata{
								Version:       "1.2.3",
								VersionSource: "buildpack.yml",
								Launch:        true,
							},
						},
					},
				}))
			})
		})

		context("when there is no buildpack.yml && BP_NGINX_VERSION is not set", func() {
			it.Before(func() {
				versionParser.ResolveVersionCall.Returns.ResultVersion = "1.19.*"
			})
			it("requires nginx at any version", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{
						{Name: "nginx"},
					},
					Requires: []packit.BuildPlanRequirement{
						{
							Name: "nginx",
							Metadata: nginx.BuildPlanMetadata{
								Version:       "1.19.*",
								VersionSource: "buildpack.toml",
								Launch:        true,
							},
						},
					},
				}))
			})
		})

	})

	context("nginx.conf is absent", func() {
		// This is for cases where nginx cnb's role is to simply provide the
		// dependency thus facilitating a downstream buildpack to 'require' nginx
		// and provide its own config
		it.Before(func() {
			Expect(filepath.Join(workingDir, "nginx.conf")).NotTo(BeAnExistingFile())
		})
		it("provides nginx", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "nginx"},
				},
				Requires: nil,
			}))
		})
	})

	context("failure cases", func() {
		var confPath string
		it.Before(func() {
			confPath = filepath.Join(workingDir, "nginx.conf")
			Expect(os.WriteFile(confPath,
				[]byte(`conf`),
				0644,
			)).To(Succeed())
		})

		context("unable to stat nginx.conf", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to stat nginx.conf")))
			})
		})

		context("version parsing fails", func() {
			it.Before(func() {
				versionParser.ResolveVersionCall.Returns.Err = errors.New("parsing version failed")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})

				Expect(err).To(MatchError(ContainSubstring("parsing version failed")))
			})
		})
	})
}

package nginx_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/nginx/nginx"
	"github.com/paketo-buildpacks/nginx/nginx/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string

		versionParser *fakes.VersionParser
		detect        packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		versionParser = &fakes.VersionParser{}
		versionParser.ParseVersionCall.Returns.Version = "*"

		detect = nginx.Detect(versionParser)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
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

	context("nginx.conf is present", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"),
				[]byte(`conf`),
				0644,
			)).To(Succeed())
		})

		context("when there is no buildpack.yml", func() {
			it.Before(func() {
				versionParser.ParseVersionCall.Returns.VersionSource = "buildpack.toml"
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
							Name:    "nginx",
							Version: "*",
							Metadata: nginx.BuildPlanMetadata{
								VersionSource: "buildpack.toml",
							},
						},
					},
				}))
			})
		})

		context("when there is a buildpack.yml", func() {
			it.Before(func() {
				versionParser.ParseVersionCall.Returns.Version = "1.2.3"
				versionParser.ParseVersionCall.Returns.VersionSource = "buildpack.yml"
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
							Name:    "nginx",
							Version: "1.2.3",
							Metadata: nginx.BuildPlanMetadata{
								VersionSource: "buildpack.yml",
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
			Expect(ioutil.WriteFile(confPath,
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
				versionParser.ParseVersionCall.Returns.Err = errors.New("parsing version failed")
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

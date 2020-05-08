package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/occam"
	"github.com/cloudfoundry/occam/matchers"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testLogging(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker

		name  string
		image occam.Image
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
	})

	when("building an app image", func() {
		it("correctly outputs logs", func() {
			var err error
			var logs fmt.Stringer
			buildpackVersion, err := GetGitVersion()
			Expect(err).NotTo(HaveOccurred())

			image, logs, err = pack.Build.
				WithBuildpacks(uri).
				WithNoPull().
				Execute(name, filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(matchers.ContainLines(
				fmt.Sprintf("Nginx Server Buildpack %s", buildpackVersion),
				"  Resolving Nginx Server version",
				"    Candidate version sources (in priority order):",
				`      buildpack.yml -> "1.17.*"`,
				"",
				MatchRegexp(`    Selected Nginx Server version \(using buildpack\.yml\): 1\.17\.\d+`),
				"",
				"  Executing build process",
				MatchRegexp(`    Installing Nginx Server \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
				"",
				"  Configuring environment",
				`    PATH -> "$PATH:/layers/paketo-buildpacks_nginx/nginx/sbin"`,
				MatchRegexp(`    Writing profile.d/configure.sh`),
				MatchRegexp(`      Calls executable that parses templates in nginx conf`),
			))

		})
	})
}

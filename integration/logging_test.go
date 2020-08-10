package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/occam/matchers"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testLogging(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker

		name   string
		source string
		image  occam.Image
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
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	when("building an app image", func() {
		it("correctly outputs logs", func() {
			var err error
			var logs fmt.Stringer
			buildpackVersion, err := GetGitVersion()
			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			image, logs, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithNoPull().
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(matchers.ContainLines(
				fmt.Sprintf("%s %s", buildpackInfo.Buildpack.Name, buildpackVersion),
				"  Resolving Nginx Server version",
				"    Candidate version sources (in priority order):",
				`      buildpack.yml -> "1.19.*"`,
				"",
				MatchRegexp(`    Selected Nginx Server version \(using buildpack\.yml\): 1\.19\.\d+`),
				"",
				"  Executing build process",
				MatchRegexp(`    Installing Nginx Server \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
				"",
				"  Configuring environment",
				fmt.Sprintf(`    PATH -> "$PATH:/layers/%s/nginx/sbin"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				MatchRegexp(`    Writing profile.d/configure.sh`),
				MatchRegexp(`      Calls executable that parses templates in nginx conf`),
			))
		})
	})
}

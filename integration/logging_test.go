package integration_test

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

func testLogging(t *testing.T, context spec.G, it spec.S) {
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

		source, err = occam.Source(filepath.Join("testdata", "simple_app"))
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when building an app image", func() {
		it("correctly outputs logs", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithBuildpacks(settings.Buildpacks.NGINX.Online).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(matchers.ContainLines(
				fmt.Sprintf("%s 1.2.3", settings.Buildpack.Name),
				"  Resolving Nginx Server version",
				"    Candidate version sources (in priority order):",
				`      buildpack.yml -> "1.27.*"`,
			))
			Expect(logs).To(matchers.ContainLines(
				MatchRegexp(`    Selected Nginx Server version \(using buildpack\.yml\): 1\.27\.\d+`),
			))
			Expect(logs).To(matchers.ContainLines(
				"    WARNING: Setting the server version through buildpack.yml will be deprecated soon in Nginx Server Buildpack v2.0.0.",
				"    Please specify the version through the $BP_NGINX_VERSION environment variable instead. See docs for more information.",
			))
			Expect(logs).To(matchers.ContainLines(
				"  Executing build process",
				MatchRegexp(`    Installing Nginx Server \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
			))
			Expect(logs).To(matchers.ContainLines(
				"  Configuring build environment",
				fmt.Sprintf(`    PATH -> "$PATH:/layers/%s/nginx/sbin"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
			Expect(logs).To(matchers.ContainLines(
				"  Configuring launch environment",
				`    EXECD_CONF -> "/workspace/nginx.conf"`,
				fmt.Sprintf(`    PATH       -> "$PATH:/layers/%s/nginx/sbin"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})
	})

	context("when version is set via BP_NGINX_VERSION", func() {
		it("correctly outputs logs", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithEnv(map[string]string{"BP_NGINX_VERSION": "stable"}).
				WithBuildpacks(settings.Buildpacks.NGINX.Online).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(matchers.ContainLines(
				fmt.Sprintf("%s 1.2.3", settings.Buildpack.Name),
				"  Resolving Nginx Server version",
				"    Candidate version sources (in priority order):",
				`      BP_NGINX_VERSION -> "1.26.*"`,
				`      buildpack.yml    -> "1.27.*"`,
			))
			Expect(logs).To(matchers.ContainLines(
				MatchRegexp(`    Selected Nginx Server version \(using BP_NGINX_VERSION\): 1\.26\.\d+`),
			))
			Expect(logs).To(matchers.ContainLines(
				"  Executing build process",
				MatchRegexp(`    Installing Nginx Server \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
			))
			Expect(logs).To(matchers.ContainLines(
				"  Configuring build environment",
				fmt.Sprintf(`    PATH -> "$PATH:/layers/%s/nginx/sbin"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
			Expect(logs).To(matchers.ContainLines(
				"  Configuring launch environment",
				`    EXECD_CONF -> "/workspace/nginx.conf"`,
				fmt.Sprintf(`    PATH       -> "$PATH:/layers/%s/nginx/sbin"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})
	})
}

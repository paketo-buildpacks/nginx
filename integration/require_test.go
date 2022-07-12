package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testRequire(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker

		name    string
		source  string
		sbomDir string

		image     occam.Image
		container occam.Container
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		source, err = os.MkdirTemp("", "require")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.WriteFile(filepath.Join(source, "plan.toml"), []byte(`
[[requires]]
	name = "nginx"

	[requires.metadata]
		launch = true
`), 0600)).To(Succeed())
	})

	it.After(func() {
		if container.ID != "" {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		}

		if image.ID != "" {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		}

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

		Expect(os.RemoveAll(source)).To(Succeed())
		Expect(os.RemoveAll(sbomDir)).To(Succeed())
	})

	it("installs nginx into the container", func() {
		var err error
		image, _, err = pack.Build.
			WithBuildpacks(
				settings.Buildpacks.NGINX.Online,
				settings.Buildpacks.BuildPlan.Online,
			).
			WithPullPolicy("never").
			WithSBOMOutputDir(sbomDir).
			Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		container, err = docker.Container.Run.
			WithCommand("nginx -v").
			Execute(image.ID)
		Expect(err).ToNot(HaveOccurred())

		logs, err := docker.Container.Logs.Execute(container.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(logs).To(ContainLines(
			MatchRegexp(`nginx version: nginx\/\d+\.\d+\.\d+`),
		))
	})
}

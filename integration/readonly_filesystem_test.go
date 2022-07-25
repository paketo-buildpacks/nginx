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

func testReadonlyFilesystem(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

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

		source, err = occam.Source(filepath.Join("testdata", "simple_app"))
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(filepath.Join(source, "custom.conf"), []byte(`server {
  listen 8080;
  root public;
  index index.html index.htm Default.htm;
}`), 0600)
		Expect(err).NotTo(HaveOccurred())
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

	it("does not try to write out any conf files at runtime", func() {
		var err error
		image, _, err = pack.Build.
			WithBuildpacks(settings.Buildpacks.NGINX.Online).
			WithPullPolicy("never").
			WithSBOMOutputDir(sbomDir).
			Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		container, err = docker.Container.Run.
			WithPublish("8080").
			WithReadOnly().
			WithMounts("type=tmpfs,destination=/tmp").
			Execute(image.ID)
		Expect(err).ToNot(HaveOccurred())

		Eventually(container).Should(Serve(ContainSubstring("Hello World!")).WithEndpoint("/index.html"), func() string {
			logs, _ := docker.Container.Logs.Execute(container.ID)
			return logs.String()
		})
	})
}

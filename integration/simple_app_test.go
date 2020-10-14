package integration

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testSimpleApp(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		name      string
		source    string
		image     occam.Image
		container occam.Container
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	when("pushing simple app", func() {
		it("serves up staticfile", func() {
			var err error

			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			image, _, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s/index.html", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
		})
	})
	when("an Nginx app uses the stream module", func() {
		it("starts successfully", func() {
			var err error

			source, err = occam.Source(filepath.Join("testdata", "with_stream_module"))
			Expect(err).NotTo(HaveOccurred())

			image, _, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s/index.html", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			logs, err := docker.Container.Logs.Execute(container.ID)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainSubstring("Stream. protocol = TCP"))
			Expect(logs).ToNot(ContainSubstring("dlopen()"))
			Expect(logs).ToNot(ContainSubstring("cannot open shared object file"))
		})
	})
}

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testSimpleApp(t *testing.T, context spec.G, it spec.S) {
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
		if container.ID != "" {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		}
		if image.ID != "" {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		}
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when pushing a simple app", func() {
		it.Before(func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it("serves up staticfile", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())
			Eventually(container).Should(Serve(ContainSubstring("Hello World!")).WithEndpoint("/index.html"))
		})
	})

	context("when an nginx app uses the stream module", func() {
		it.Before(func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "with_stream_module"))
			Expect(err).NotTo(HaveOccurred())
		})

		it("starts successfully", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())
			Eventually(container).Should(Serve(ContainSubstring("Exciting Content")).WithEndpoint("/index.html"))

			logs, err := docker.Container.Logs.Execute(container.ID)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainSubstring("Stream. protocol = TCP"))
			Expect(logs).ToNot(ContainSubstring("dlopen()"))
			Expect(logs).ToNot(ContainSubstring("cannot open shared object file"))
		})
	})

	context("when BP_LIVE_RELOAD_ENABLED=true", func() {
		var noReloadContainer occam.Container

		it.Before(func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(noReloadContainer.ID)).To(Succeed())
		})

		it("adds a reloadable process type as the default process", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.Build.
				WithBuildpacks(
					watchexecBuildpack,
					nginxBuildpack,
				).
				WithPullPolicy("never").
				WithEnv(map[string]string{
					"BP_LIVE_RELOAD_ENABLED": "true",
				}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())
			Eventually(container).Should(Serve(ContainSubstring("Hello World!")).WithEndpoint("/index.html"))

			noReloadContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				WithEntrypoint("no-reload").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(noReloadContainer).Should(BeAvailable())
			Eventually(noReloadContainer).Should(Serve(ContainSubstring("Hello World!")).WithEndpoint("/index.html"))

			Expect(logs).To(ContainLines("  Assigning launch processes:"))
			Expect(logs).To(ContainLines(`    web (default): watchexec --restart --watch /workspace --shell none -- nginx -p /workspace -c /workspace/nginx.conf`))
			Expect(logs).To(ContainLines(`    no-reload:     nginx -p /workspace -c /workspace/nginx.conf`))
		})
	})
	context("when build configuration cannot be parsed", func() {
		it.Before(func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it("fails with a helpful error", func() {
			var err error
			var logs fmt.Stringer
			_, logs, err = pack.Build.
				WithBuildpacks(
					watchexecBuildpack,
					nginxBuildpack,
				).
				WithPullPolicy("never").
				WithEnv(map[string]string{
					"BP_LIVE_RELOAD_ENABLED": "not-a-bool",
				}).
				Execute(name, source)
			Expect(err).To(HaveOccurred())

			Expect(logs).To(ContainSubstring("invalid syntax"))
		})
	})
}

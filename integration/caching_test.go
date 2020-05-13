package integration

import (
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/paketo-buildpacks/occam/matchers"
	. "github.com/onsi/gomega"
)

func testCaching(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		name         string
		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		imageIDs = map[string]struct{}{}
		containerIDs = map[string]struct{}{}
	})

	it.After(func() {
		for id := range containerIDs {
			Expect(docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id := range imageIDs {
			Expect(docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
	})

	it("uses a cached layer and doesn't run twice", func() {
		source := filepath.Join("testdata", "simple_app")

		build := pack.Build.WithBuildpacks(uri)

		firstImage, _, err := build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		imageIDs[firstImage.ID] = struct{}{}

		Expect(firstImage.Buildpacks).To(HaveLen(1))
		Expect(firstImage.Buildpacks[0].Key).To(Equal("paketo-buildpacks/nginx"))
		Expect(firstImage.Buildpacks[0].Layers).To(HaveKey("nginx"))

		container, err := docker.Container.Run.Execute(firstImage.ID)
		Expect(err).NotTo(HaveOccurred())

		containerIDs[container.ID] = struct{}{}

		Eventually(container).Should(BeAvailable())

		secondImage, _, err := build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		imageIDs[secondImage.ID] = struct{}{}

		Expect(secondImage.Buildpacks).To(HaveLen(1))
		Expect(secondImage.Buildpacks[0].Key).To(Equal("paketo-buildpacks/nginx"))
		Expect(secondImage.Buildpacks[0].Layers).To(HaveKey("nginx"))

		container, err = docker.Container.Run.Execute(secondImage.ID)
		Expect(err).NotTo(HaveOccurred())

		containerIDs[container.ID] = struct{}{}

		Eventually(container).Should(BeAvailable())

		Expect(secondImage.Buildpacks[0].Layers["nginx"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[0].Layers["nginx"].Metadata["built_at"]))
		Expect(secondImage.Buildpacks[0].Layers["nginx"].Metadata["dependency-sha"]).To(Equal(firstImage.Buildpacks[0].Layers["nginx"].Metadata["dependency-sha"]))
		Expect(secondImage.Buildpacks[0].Layers["nginx"].Metadata["configure-bin-sha"]).To(Equal(firstImage.Buildpacks[0].Layers["nginx"].Metadata["configure-bin-sha"]))
		Expect(secondImage.ID).To(Equal(firstImage.ID))
	})
}

package nginx_test

import (
	"bytes"
	"testing"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testLogEmitter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer  *bytes.Buffer
		emitter nginx.LogEmitter
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)
		emitter = nginx.NewLogEmitter(buffer)
	})

	context("Title", func() {
		it("logs the buildpack title", func() {
			emitter.Title(packit.BuildpackInfo{
				Name:    "some-name",
				Version: "some-version",
			})
			Expect(buffer.String()).To(Equal("some-name some-version\n"))
		})
	})

	context("Candidates", func() {
		it("logs the candidate entries", func() {
			emitter.Candidates([]packit.BuildpackPlanEntry{
				{
					Metadata: map[string]interface{}{
						"version":        "some-version",
						"version-source": "some-source",
					},
				},
				{
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
			})
			Expect(buffer.String()).To(Equal(`    Candidate version sources (in priority order):
      some-source -> "some-version"
      <unknown>   -> "other-version"

`))
		})
	})

	context("SelectedEntry", func() {
		it("logs the selected entry", func() {
			emitter.SelectedDependency(packit.BuildpackPlanEntry{
				Metadata: map[string]interface{}{"version-source": "some-source"},
			}, "some-version")
			Expect(buffer.String()).To(Equal("    Selected Nginx Server version (using some-source): some-version\n"))
		})
	})

	context("Environment", func() {
		it("logs the environment variables", func() {
			emitter.Environment(packit.Environment{
				"SOME_VAR.override": "some-value",
			})
			Expect(buffer.String()).To(Equal("    SOME_VAR -> \"some-value\"\n"))
		})
	})
}

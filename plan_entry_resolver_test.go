package nginx_test

import (
	"testing"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPlanEntryResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect        = NewWithT(t).Expect
		entryResolver nginx.PlanEntryResolver
	)

	it.Before(func() {
		entryResolver = nginx.NewPlanEntryResolver()
	})

	context("when resolving multiple entries", func() {
		it("selectes the highest priority", func() {
			entry := entryResolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "nginx",
					Metadata: map[string]interface{}{
						"version":        "1.2.3",
						"version-source": "buildpack.yml",
					},
				},

				{
					Name: "nginx",
				},
			})

			Expect(entry).To(Equal(packit.BuildpackPlanEntry{

				Name: "nginx",
				Metadata: map[string]interface{}{
					"version":        "1.2.3",
					"version-source": "buildpack.yml",
				},
			}))
		})
	})

}

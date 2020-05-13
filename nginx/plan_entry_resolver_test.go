package nginx_test

import (
	"testing"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/nginx/nginx"
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
					Name:    "nginx",
					Version: "1.2.3",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
					},
				},

				{
					Name: "nginx",
				},
			})

			Expect(entry).To(Equal(packit.BuildpackPlanEntry{

				Name:    "nginx",
				Version: "1.2.3",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
				},
			}))
		})
	})

}

package integration

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/occam"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testNoConfApp(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		pack   occam.Pack
		name   string
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	when("pushing app with no conf", func() {
		// This is for when downstream buildpacks require nginx
		it("build fails but provides unused nginx", func() {
			var err error
			_, _, err = pack.Build.
				WithBuildpacks(uri).
				WithNoPull().
				Execute(name, filepath.Join("testdata", "no_conf_app"))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("[detector] pass: paketo-buildpacks/nginx")))
			Expect(err).To(MatchError(ContainSubstring("provides unused nginx")))
		})
	})
}

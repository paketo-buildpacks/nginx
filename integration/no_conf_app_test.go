package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testNoConfApp(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		pack   occam.Pack
		name   string
		source string
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		source, err = occam.Source(filepath.Join("testdata", "no_conf_app"))
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when pushing app with no conf", func() {
		it("fails to build but provides unused nginx", func() {
			_, _, err := pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("[detector] pass: %s", buildpackInfo.Buildpack.ID))))
			Expect(err).To(MatchError(ContainSubstring("provides unused nginx")))
		})
	})
}

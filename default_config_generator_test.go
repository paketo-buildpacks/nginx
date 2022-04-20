package nginx_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/nginx"
	"github.com/sclevine/spec"
)

func testDefaultConfigGenerator(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		sourceDir  string
		generator  nginx.DefaultConfigGenerator
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		sourceDir, err = os.MkdirTemp("", "source-dir")
		Expect(err).NotTo(HaveOccurred())

		generator = nginx.NewDefaultConfigGenerator()
	})

	context("Generate", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(sourceDir, "source.conf"), []byte("some file contents"), os.ModePerm)).To(Succeed())
		})

		it("writes a default nginx.conf to the working directory", func() {
			err := generator.Generate(filepath.Join(sourceDir, "source.conf"), filepath.Join(workingDir, "nginx.conf"))
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).To(BeARegularFile())
			contents, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal(`some file contents`))
		})

		context("failure cases", func() {
			context("source file cannot be copied", func() {
				it("returns an error", func() {
					err := generator.Generate("not-a-path", filepath.Join(workingDir, "nginx.conf"))
					Expect(err).To(MatchError(ContainSubstring("failed to generate default nginx.conf: stat")))
				})
			})
		})
	})
}

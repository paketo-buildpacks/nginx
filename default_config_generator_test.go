package nginx_test

import (
	"fmt"
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
			Expect(os.WriteFile(filepath.Join(sourceDir, "template.conf"), []byte("root $(( .Root ));"), os.ModePerm)).To(Succeed())
		})

		it("writes a default nginx.conf to the working directory", func() {
			err := generator.Generate(filepath.Join(sourceDir, "template.conf"), filepath.Join(workingDir, "nginx.conf"), "")
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).To(BeARegularFile())
			contents, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal(`root {{ env "APP_ROOT" }}/public;`))
		})

		it("writes a nginx.conf with specified relative root directory", func() {
			err := generator.Generate(filepath.Join(sourceDir, "template.conf"), filepath.Join(workingDir, "nginx.conf"), "custom")
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).To(BeARegularFile())
			contents, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal(`root {{ env "APP_ROOT" }}/custom;`))
		})

		it("writes a nginx.conf with specified absolute path to root directory", func() {
			err := generator.Generate(filepath.Join(sourceDir, "template.conf"), filepath.Join(workingDir, "nginx.conf"), "/some/absolute/path")
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).To(BeARegularFile())
			contents, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal(`root /some/absolute/path;`))
		})

		context("failure cases", func() {
			context("source template cannot be found", func() {
				it("returns an error", func() {
					err := generator.Generate("not-a-path", filepath.Join(workingDir, "nginx.conf"), "custom")
					Expect(err).To(MatchError(ContainSubstring("failed to locate nginx.conf template: stat")))
				})
			})
			context("destination file already exists and it's read-only", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(sourceDir, "template.conf"), []byte("root $(( .Root ));"), os.ModePerm)).To(Succeed())
					Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte("read-only file"), 0444)).To(Succeed())
				})
				it("returns an error", func() {
					err := generator.Generate(filepath.Join(sourceDir, "template.conf"), filepath.Join(workingDir, "nginx.conf"), "custom")
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to create %[1]s: open %[1]s: permission denied", filepath.Join(workingDir, "nginx.conf")))))
				})
			})
		})
	})
}

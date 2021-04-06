package nginx_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/nginx"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		cnbPath    string
		parser     nginx.Parser
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbPath, err = ioutil.TempDir("", "cnbPath")
		Expect(err).NotTo(HaveOccurred())

		Expect(ioutil.WriteFile(
			filepath.Join(cnbPath, "buildpack.toml"),
			[]byte(`
[metadata]
  [metadata.default-versions]
    nginx = "the-default-version"
  [metadata.version-lines]
    mainline = "1.17.*"
    stable = "1.16.*"
`), 0644)).To(Succeed())

		parser = nginx.NewParser()
	})

	context("Calling ParseYml", func() {
		context("when buildpack.yml is valid and specifies an nginx version", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`---
nginx:
  version: 1.2.3`),
					os.ModePerm,
				))
			})

			it("parses out a version from buildpack.yml and detects version source", func() {
				version, ok, err := parser.ParseYml(workingDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(Equal(true))
				Expect(version).To(Equal("1.2.3"))
			})
		})

		context("buildpack.yml cannot be opened", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`some-content`),
					0000,
				))
			})

			it("returns an error", func() {
				_, _, err := parser.ParseYml(workingDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("buildpack.yml cannot be parsed", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`%%%`),
					0644,
				))
			})

			it("returns an error", func() {
				_, _, err := parser.ParseYml(workingDir)
				Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
			})
		})

	})

	context("Calling ResolveVersion", func() {
		context("with mainline version", func() {
			it("return the buildpack.toml mainline version", func() {
				version, err := parser.ResolveVersion(cnbPath, "mainline")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("1.17.*"))
			})
		})

		context("with stable version", func() {
			it("return the buildpack.toml stable version", func() {
				version, err := parser.ResolveVersion(cnbPath, "stable")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("1.16.*"))
			})
		})

		context("with empty version", func() {
			it("return the buildpack.toml default version", func() {
				version, err := parser.ResolveVersion(cnbPath, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("the-default-version"))
			})
		})

		context("with semver version", func() {
			it("return the same semver version", func() {
				version, err := parser.ResolveVersion(cnbPath, "1.1.1")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("1.1.1"))
			})
		})
	})

	context("failure cases", func() {
		context("buildpack.toml cannot be opened", func() {
			it.Before(func() {
				Expect(os.Chmod(filepath.Join(cnbPath, "buildpack.toml"), 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.ResolveVersion(cnbPath, "")
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("buildpack.toml cannot be parsed", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(cnbPath, "buildpack.toml"),
					[]byte(`%%%`),
					0644,
				)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := parser.ResolveVersion(cnbPath, "")
				Expect(err).To(MatchError(ContainSubstring("bare keys cannot contain")))
			})
		})
	})
}

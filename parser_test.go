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

	context("when buildpack.yml exists", func() {
		context("when buildpack.yml is valid and specifies an nginx semver", func() {
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
				version, versionSource, err := parser.ParseVersion(workingDir, cnbPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("1.2.3"))
				Expect(versionSource).To(Equal("buildpack.yml"))
			})
		})

		context("when buildpack.yml specifies mainline", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`---
nginx:
  version: mainline`),
					os.ModePerm,
				))
			})

			it("parses out the mainline constraint", func() {
				version, _, err := parser.ParseVersion(workingDir, cnbPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("1.17.*"))
			})
		})

		context("when buildpack.yml specifies stable", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`---
nginx:
  version: stable`),
					os.ModePerm,
				))
			})

			it("parses out the stable constraint", func() {
				version, _, err := parser.ParseVersion(workingDir, cnbPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("1.16.*"))
			})
		})

		context("when buildpack.yml does NOT specify any nginx constraint", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`---
some-other-dep:
  version: 1.2.3`),
					os.ModePerm,
				))
			})
			it("parses out a general * constraint", func() {
				version, versionSource, err := parser.ParseVersion(workingDir, cnbPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("the-default-version"))
				Expect(versionSource).To(Equal("buildpack.toml"))
			})

		})
	})

	context("when buildpack.yml does NOT exist", func() {
		it("parses out a general * constraint", func() {
			version, versionSource, err := parser.ParseVersion(workingDir, cnbPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("the-default-version"))
			Expect(versionSource).To(Equal("buildpack.toml"))
		})
	})

	context("failure cases", func() {
		context("buildpack.yml cannot be opened", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`some-content`),
					0000,
				))
			})

			it("returns an error", func() {
				_, _, err := parser.ParseVersion(workingDir, cnbPath)
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
				_, _, err := parser.ParseVersion(workingDir, cnbPath)
				Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
			})
		})

		context("buildpack.toml cannot be opened", func() {
			it.Before(func() {
				Expect(os.Chmod(filepath.Join(cnbPath, "buildpack.toml"), 0000)).To(Succeed())

				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`---
nginx:
  version: mainline`),
					os.ModePerm,
				))
			})

			it("returns an error", func() {
				_, _, err := parser.ParseVersion(workingDir, cnbPath)
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

				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "buildpack.yml"),
					[]byte(`---
nginx:
  version: mainline`),
					os.ModePerm,
				))
			})

			it("returns an error", func() {
				_, _, err := parser.ParseVersion(workingDir, cnbPath)
				Expect(err).To(MatchError(ContainSubstring("bare keys cannot contain")))
			})
		})
	})
}

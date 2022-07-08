package internal_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/nginx/cmd/configure/internal"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitConfigure(t *testing.T) {
	suite := spec.New("cmd/configure/internal", spec.Report(report.Terminal{}))
	suite("Run", testRun)
	suite.Run(t)
}

func testRun(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		mainConf         string
		localModulePath  string
		globalModulePath string
		workingDir       string
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		mainConf = filepath.Join(workingDir, "nginx.conf")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when the template contains a 'port' action", func() {
		it.Before(func() {
			Expect(os.WriteFile(mainConf, []byte("Hi the port is {{port}}."), 0600)).To(Succeed())
			os.Setenv("PORT", "8080")
		})

		it("inserts the port value into that location in the text", func() {
			err := internal.Run(mainConf, localModulePath, globalModulePath)
			Expect(err).ToNot(HaveOccurred())

			output, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
			Expect(err).ToNot(HaveOccurred())

			Expect(string(output)).To(Equal("Hi the port is 8080."))
		})
	})

	context("when the template contains a 'tempDir' action", func() {
		it.Before(func() {
			Expect(os.WriteFile(mainConf, []byte("Hi the tempDir is {{ tempDir }}."), 0600)).To(Succeed())
		})

		it("inserts the location of the user's temp directory into that location in the text", func() {
			err := internal.Run(mainConf, localModulePath, globalModulePath)
			Expect(err).ToNot(HaveOccurred())

			output, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
			Expect(err).ToNot(HaveOccurred())

			Expect(string(output)).To(Equal(fmt.Sprintf("Hi the tempDir is %s.", os.TempDir())))
		})
	})

	context("when the template contains an 'env' action", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`The env var FOO is {{env "FOO"}}`), 0600)).To(Succeed())
			os.Setenv("FOO", "BAR")
		})

		it("inserts the env variable into that location in the text", func() {
			err := internal.Run(mainConf, localModulePath, globalModulePath)
			Expect(err).ToNot(HaveOccurred())

			output, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
			Expect(err).ToNot(HaveOccurred())

			Expect(string(output)).To(Equal("The env var FOO is BAR"))
		})
	})

	context("templating a load_module directive using the 'module' func", func() {
		it.Before(func() {
			localModulePath = filepath.Join(workingDir, "local_modules")
			Expect(os.Mkdir(localModulePath, 0744)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(localModulePath, "local.so"), []byte("dummy data"), 0600)).To(Succeed())

			globalModulePath = filepath.Join(workingDir, "global_modules")
			Expect(os.Mkdir(globalModulePath, 0744)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(globalModulePath, "global.so"), []byte("dummy data"), 0600)).To(Succeed())
		})

		context("when the module is in local modules directory", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{module "local"}}`), 0600)).To(Succeed())
			})

			it("loads the module from the local directory", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).ToNot(HaveOccurred())

				output, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(output)).To(Equal(fmt.Sprintf("load_module %s/local.so;", localModulePath)))
			})
		})

		context("when the module is in global modules directory", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{module "global"}}`), 0600)).To(Succeed())
			})

			it("loads the module from the global directory", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).ToNot(HaveOccurred())

				output, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(output)).To(Equal(fmt.Sprintf("load_module %s/global.so;", globalModulePath)))
			})
		})
	})

	context("when the template uses include files", func() {
		context("include file is a complete path", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`
	http {
	include mime.types;
	include custom.conf;

	tcp_nopush on;
	keepalive_timeout 30;
	port_in_redirect off; # Ensure that redirects don't include the internal container PORT - 8080
	}`), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, "custom.conf"), []byte(`Hi the port is {{ port }}.`), 0600)).To(Succeed())
				os.Setenv("PORT", "8080")
			})

			it("parses 'include' file and interpolates values", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).ToNot(HaveOccurred())

				output, err := os.ReadFile(filepath.Join(workingDir, "custom.conf"))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(output)).To(Equal("Hi the port is 8080."))
			})
		})

		context("include is a file mask", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "subdir"), os.ModePerm)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`
		http {
		include mime.types;
		include ./subdir/*.conf;

		tcp_nopush on;
		keepalive_timeout 30;
		port_in_redirect off; # Ensure that redirects don't include the internal container PORT - 8080
		}`), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, "dontFix.conf"), []byte(`Hi the port is {{ port }}.`), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, "subdir", "custom1.conf"), []byte(`Hi the port is {{ port }}.`), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, "subdir", "custom2.conf"), []byte(`Hi the port is {{ port }}.`), 0600)).To(Succeed())
				os.Setenv("PORT", "8080")
			})

			it("parses 'include' files and interpolates values into all files that match the mask", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).ToNot(HaveOccurred())

				output, err := os.ReadFile(filepath.Join(workingDir, "dontFix.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(Equal(`Hi the port is {{ port }}.`))

				output, err = os.ReadFile(filepath.Join(workingDir, "subdir", "custom1.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(Equal(`Hi the port is 8080.`))

				output, err = os.ReadFile(filepath.Join(workingDir, "subdir", "custom2.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(Equal(`Hi the port is 8080.`))
			})
		})
	})

	context("failure cases", func() {
		context("when the template file does not exist", func() {
			it.Before(func() {
				mainConf = "/no/such/template.conf"
			})

			it("prints an error and exits non-zero", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).To(MatchError(ContainSubstring("could not read config file (/no/such/template.conf): open /no/such/template.conf: no such file or directory")))
			})
		})

		context("when the template file cannot be written", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{module "global"}}`), 0444)).To(Succeed())
			})

			it("prints an error and exits non-zero", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).To(MatchError(MatchRegexp("failed to create nginx.conf: .*: permission denied")))
			})
		})

		context("when the template is malformed", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{ port "argument" }}`), 0600)).To(Succeed())
			})

			it("prints an error and exits non-zero", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).To(MatchError(MatchRegexp("failed to execute template: .*: wrong number of args for port: want 0 got 1")))
			})
		})

		context("when the include file mask is malformed", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`include \/\/.conf;`), 0600)).To(Succeed())
			})

			it("prints an error and exits non-zero", func() {
				err := internal.Run(mainConf, localModulePath, globalModulePath)
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to get 'include' files for %s", workingDir))))
				Expect(err).To(MatchError(ContainSubstring(`/\/\/.conf: syntax error in pattern`)))
			})
		})
	})
}

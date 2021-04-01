package main_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega/gexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitConfigure(t *testing.T) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
	)

	path, err := gexec.Build("github.com/paketo-buildpacks/nginx/cmd/configure")
	Expect(err).ToNot(HaveOccurred())

	defer gexec.CleanupBuildArtifacts()

	spec.Run(t, "Configure", func(t *testing.T, context spec.G, it spec.S) {
		var (
			localModulePath  string
			globalModulePath string
			workingDir       string

			command *exec.Cmd
			buffer  *bytes.Buffer
		)

		it.Before(func() {
			var err error
			workingDir, err = ioutil.TempDir("", "working-dir")
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		context("when the template contains a 'port' action", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte("Hi the port is {{port}}."), 0644)).To(Succeed())

				command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
				command.Env = []string{"PORT=8080"}

				buffer = bytes.NewBuffer(nil)
			})

			it("inserts the port value into that location in the text", func() {
				session, err := gexec.Start(command, buffer, buffer)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0), buffer.String)

				output, err := ioutil.ReadFile(filepath.Join(workingDir, "nginx.conf"))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(output)).To(Equal("Hi the port is 8080."))
			})
		})

		context("when the template contains an 'env' action", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`The env var FOO is {{env "FOO"}}`), 0644)).To(Succeed())

				command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
				command.Env = []string{"FOO=BAR"}

				buffer = bytes.NewBuffer(nil)
			})

			it("inserts the env variable into that location in the text", func() {
				session, err := gexec.Start(command, buffer, buffer)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0), buffer.String)

				output, err := ioutil.ReadFile(filepath.Join(workingDir, "nginx.conf"))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(output)).To(Equal("The env var FOO is BAR"))
			})
		})

		context("templating a load_module directive using the 'module' func", func() {
			it.Before(func() {
				localModulePath = filepath.Join(workingDir, "local_modules")
				Expect(os.Mkdir(localModulePath, 0744)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(localModulePath, "local.so"), []byte("dummy data"), 0644)).To(Succeed())

				globalModulePath = filepath.Join(workingDir, "global_modules")
				Expect(os.Mkdir(globalModulePath, 0744)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(globalModulePath, "global.so"), []byte("dummy data"), 0644)).To(Succeed())
			})

			context("when the module is in local modules directory", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{module "local"}}`), 0644)).To(Succeed())

					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					buffer = bytes.NewBuffer(nil)
				})

				it("loads the module from the local directory", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(0), buffer.String)

					output, err := ioutil.ReadFile(filepath.Join(workingDir, "nginx.conf"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(output)).To(Equal(fmt.Sprintf("load_module %s/local.so;", localModulePath)))
				})
			})

			context("when the module is in global modules directory", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{module "global"}}`), 0644)).To(Succeed())

					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					buffer = bytes.NewBuffer(nil)
				})

				it("loads the module from the global directory", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(0), buffer.String)

					output, err := ioutil.ReadFile(filepath.Join(workingDir, "nginx.conf"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(output)).To(Equal(fmt.Sprintf("load_module %s/global.so;", globalModulePath)))
				})
			})
		})

		context("when the template uses include files", func() {
			context("include file is a complete path", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`
http {
  include mime.types;
	include custom.conf;

  tcp_nopush on;
  keepalive_timeout 30;
  port_in_redirect off; # Ensure that redirects don't include the internal container PORT - 8080
}`), 0644)).To(Succeed())
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "custom.conf"), []byte(`Hi the port is {{ port }}.`), 0644)).To(Succeed())

					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					command.Env = []string{"PORT=8080"}
					buffer = bytes.NewBuffer(nil)
				})

				it("parses 'include' file and interpolates values", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(0), buffer.String)

					output, err := ioutil.ReadFile(filepath.Join(workingDir, "custom.conf"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(output)).To(Equal("Hi the port is 8080."))
				})
			})
			context("include is a file mask", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, "subdir"), os.ModePerm)).To(Succeed())
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`
http {
  include mime.types;
	include ./subdir/*.conf;

  tcp_nopush on;
  keepalive_timeout 30;
  port_in_redirect off; # Ensure that redirects don't include the internal container PORT - 8080
}`), 0644)).To(Succeed())
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "dontFix.conf"), []byte(`Hi the port is {{ port }}.`), 0644)).To(Succeed())
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "subdir", "custom1.conf"), []byte(`Hi the port is {{ port }}.`), 0644)).To(Succeed())
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "subdir", "custom2.conf"), []byte(`Hi the port is {{ port }}.`), 0644)).To(Succeed())

					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					command.Env = []string{"PORT=8080"}
					buffer = bytes.NewBuffer(nil)
				})

				it("parses 'include' files and interpolates values into all files that match the mask", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(0), buffer.String)

					output, err := ioutil.ReadFile(filepath.Join(workingDir, "dontFix.conf"))
					Expect(err).ToNot(HaveOccurred())
					Expect(string(output)).To(Equal(`Hi the port is {{ port }}.`))

					output, err = ioutil.ReadFile(filepath.Join(workingDir, "subdir", "custom1.conf"))
					Expect(err).ToNot(HaveOccurred())
					Expect(string(output)).To(Equal(`Hi the port is 8080.`))

					output, err = ioutil.ReadFile(filepath.Join(workingDir, "subdir", "custom2.conf"))
					Expect(err).ToNot(HaveOccurred())
					Expect(string(output)).To(Equal(`Hi the port is 8080.`))
				})
			})
		})

		context("failure cases", func() {
			context("when the template file does not exist", func() {
				it.Before(func() {
					command = exec.Command(path, "/no/such/template.conf", localModulePath, globalModulePath)
					buffer = bytes.NewBuffer(nil)
				})

				it("prints an error and exits non-zero", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(1), buffer.String)

					Expect(buffer.String()).To(Equal("Could not read config file (/no/such/template.conf): open /no/such/template.conf: no such file or directory\n"))
				})
			})

			context("when the template file cannot be written", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{module "global"}}`), 0444)).To(Succeed())

					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					buffer = bytes.NewBuffer(nil)
				})

				it("prints an error and exits non-zero", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(1), buffer.String)

					Expect(buffer.String()).To(MatchRegexp("failed to create nginx.conf: .*: permission denied\n"))
				})
			})

			context("when the template is malformed", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`{{ port "argument" }}`), 0644)).To(Succeed())

					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					buffer = bytes.NewBuffer(nil)
				})

				it("prints an error and exits non-zero", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(1), buffer.String)

					Expect(buffer.String()).To(MatchRegexp("failed to execute template: .*: wrong number of args for port: want 0 got 1\n"))
				})
			})
			context("when the include file mask is malformed", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte(`include \/\/.conf;`), 0644)).To(Succeed())

					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					buffer = bytes.NewBuffer(nil)
				})

				it("prints an error and exits non-zero", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())

					Eventually(session).Should(gexec.Exit(1), buffer.String)
					Expect(buffer.String()).To(ContainSubstring(`failed to get 'include' files for /tmp/working-dir`))
					Expect(buffer.String()).To(ContainSubstring(`/\/\/.conf: syntax error in pattern`))
				})
			})
			context("when an include file does not exist", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte("include donotexist.conf;"), 0644)).To(Succeed())
					command = exec.Command(path, filepath.Join(workingDir, "nginx.conf"), localModulePath, globalModulePath)
					buffer = bytes.NewBuffer(nil)
				})

				it("exits non-zero", func() {
					session, err := gexec.Start(command, buffer, buffer)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session).Should(gexec.Exit(1), buffer.String)
					Expect(buffer.String()).To(ContainSubstring(`failed to get 'include' files for /tmp/working-dir`))
					Expect(buffer.String()).To(ContainSubstring("no matching files exist"))
				})
			})
		})
	}, spec.Report(report.Terminal{}))
}

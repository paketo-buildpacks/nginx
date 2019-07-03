/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

var (
	pathToCli string
)

func TestUnitConfigure(t *testing.T) {
	spec.Run(t, "Configure", testConfigure, spec.Report(report.Terminal{}))
}

func runCli(tmpDir, body string, env []string, localModulePath, globalModulePath string) string {
	Expect(ioutil.WriteFile(filepath.Join(tmpDir, "nginx.conf"), []byte(body), 0644)).To(Succeed())

	command := exec.Command(pathToCli, filepath.Join(tmpDir, "nginx.conf"), localModulePath, globalModulePath)
	command.Env = env
	session, err := gexec.Start(command, os.Stdout, os.Stderr)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))

	output, err := ioutil.ReadFile(filepath.Join(tmpDir, "nginx.conf"))
	Expect(err).ToNot(HaveOccurred())

	return string(output)
}

func testConfigure(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory
	var (
		localModulePath, globalModulePath string
	)

	it.Before(func() {
		var err error

		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
		pathToCli, err = gexec.Build("github.com/cloudfoundry/nginx-cnb/cmd/configure")
		Expect(err).ToNot(HaveOccurred())
	})

	when("it runs", func() {
		it.Before(func() {
			test.WriteFile(t, filepath.Join(factory.Home, "nginx.conf"), "")
		})

		it("templates current port using the 'port' func", func() {
			body := runCli(factory.Home, "Hi the port is {{port}}.", []string{"PORT=8080"}, "", "")
			Expect(body).To(Equal("Hi the port is 8080."))
		})

		it("templates environment variables using the 'env' func", func() {
			body := runCli(factory.Home, `The env var FOO is {{env "FOO"}}`, []string{"FOO=BAR"}, "", "")
			Expect(body).To(Equal("The env var FOO is BAR"))
		})

		when("templating a load_module directive using the 'module' func", func() {
			it.Before(func() {
				localModulePath = filepath.Join(factory.Home, "local_modules")
				globalModulePath = filepath.Join(factory.Home, "global_modules")

				Expect(os.Mkdir(localModulePath, 0744)).To(Succeed())
				Expect(os.Mkdir(globalModulePath, 0744)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(localModulePath, "local.so"), []byte("dummy data"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(globalModulePath, "global.so"), []byte("dummy data"), 0644)).To(Succeed())
			})

			when("when the module is in local modules directory", func() {
				it("loads the module from the local directory", func() {
					body := runCli(factory.Home, `{{module "local"}}`, nil, localModulePath, globalModulePath)
					Expect(body).To(Equal(fmt.Sprintf("load_module %s/local.so;", localModulePath)))
				})
			})

			when("when the module is in global modules directory", func() {
				it("loads the module from the global directory", func() {
					body := runCli(factory.Home, `{{module "global"}}`, nil, localModulePath, globalModulePath)
					Expect(body).To(Equal(fmt.Sprintf("load_module %s/global.so;", globalModulePath)))
				})
			})
		})
	})
}

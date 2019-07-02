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

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	uri string
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	root, err := dagger.FindBPRoot()
	Expect(err).ToNot(HaveOccurred())
	uri, err = dagger.PackageBuildpack(root)
	Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(uri)
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("push simple app", func() {
		it("serves up staticfile", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app"), uri)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			_, _, err = app.HTTPGet("/index.html")
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Destroy()).To(Succeed())
		})
	})

	when("an Nginx app uses the stream module", func() {
		it("starts successfully", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "with_stream_module"), uri)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			_, _, err = app.HTTPGet("/index.html")
			Expect(err).ToNot(HaveOccurred())

			logs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).ToNot(ContainSubstring("dlopen()"))
			Expect(logs).ToNot(ContainSubstring("cannot open shared object file"))
			Expect(logs).ToNot(ContainSubstring("No such file or directory"))

			Expect(app.Destroy()).To(Succeed())
		})
	})
}

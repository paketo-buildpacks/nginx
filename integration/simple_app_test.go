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
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/occam"

	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/occam/matchers"
	. "github.com/onsi/gomega"
)

func testSimpleApp(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		name      string
		image     occam.Image
		container occam.Container
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
	})

	when("pushing simple app", func() {
		//when("rebuilding app", func() {
		it("serves up staticfile", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(uri).
				WithNoPull().
				Execute(name, filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s/index.html", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
		})
	})
	when("an Nginx app uses the stream module", func() {
		it("starts successfully", func() {
			var err error

			image, _, err = pack.Build.
				WithBuildpacks(uri).
				WithNoPull().
				Execute(name, filepath.Join("testdata", "with_stream_module"))
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s/index.html", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			logs, err := docker.Container.Logs.Execute(container.ID)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainSubstring("Stream. protocol = TCP"))
			Expect(logs).ToNot(ContainSubstring("dlopen()"))
			Expect(logs).ToNot(ContainSubstring("cannot open shared object file"))
		})
	})
}

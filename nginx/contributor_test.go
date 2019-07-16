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

package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/helper"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/sclevine/spec/report"

	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func TestUnitNGINXContributor(t *testing.T) {
	spec.Run(t, "NGINX Contributor", testNGINXContributor, spec.Report(report.Terminal{}))
}

func testNGINXContributor(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("NewContributor", func() {
		var stubNGINXFixture = filepath.Join("testdata", "stub-nginx.tar.gz")

		it("returns true if a build plan exists", func() {
			f := test.NewBuildFactory(t)
			f.AddBuildPlan(Dependency, buildplan.Dependency{})
			f.AddDependency(Dependency, stubNGINXFixture)

			_, willContribute, err := NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})

		it("returns false if a build plan does not exist", func() {
			f := test.NewBuildFactory(t)

			_, willContribute, err := NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})

		it("should contribute nginx to launch when launch is true", func() {
			f := test.NewBuildFactory(t)

			Expect(helper.WriteFile(filepath.Join(f.Build.Buildpack.Root, "bin", "configure"), os.ModePerm, "")).To(Succeed())
			Expect(helper.WriteFile(filepath.Join(f.Build.Application.Root, "nginx.conf"), os.ModePerm, "")).To(Succeed())

			f.AddBuildPlan(Dependency, buildplan.Dependency{
				Metadata: buildplan.Metadata{"launch": true},
			})
			f.AddDependency(Dependency, stubNGINXFixture)

			nodeContributor, _, err := NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(nodeContributor.Contribute()).To(Succeed())

			layer := f.Build.Layers.Layer(Dependency)

			Expect(layer).To(test.HaveLayerMetadata(false, false, true))
			Expect(filepath.Join(layer.Root, "stub.txt")).To(BeARegularFile())

			nginxConfPath := filepath.Join(f.Build.Application.Root, "nginx.conf")
			appModsPath := filepath.Join(f.Build.Application.Root, "modules")
			pkgModsPath := filepath.Join(layer.Root, "modules")
			varifyCmd := fmt.Sprintf(`configure "%s" "%s" "%s"`, nginxConfPath, appModsPath, pkgModsPath)
			Expect(layer).To(test.HaveProfile("configure", varifyCmd))

			nginxCmd := fmt.Sprintf(`nginx -p $PWD -c "%s"`, nginxConfPath)
			Expect(f.Build.Layers).To(test.HaveApplicationMetadata(
				layers.Metadata{Processes: []layers.Process{{"web", nginxCmd}}},
			))

		})
	})
}

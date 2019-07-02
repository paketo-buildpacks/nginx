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
	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"os"
	"path/filepath"
)

// Contributor is responsible for deciding what this buildpack will contribute during build
type Contributor struct {
	app                application.Application
	verifyPath         string
	launchContribution bool
	launchLayer        layers.Layers
	nginxLayer         layers.DependencyLayer
}

// NewContributor will create a new Contributor object
func NewContributor(context build.Build) (c Contributor, willContribute bool, err error) {
	plan, wantDependency := context.BuildPlan[Dependency]
	if !wantDependency {
		return Contributor{}, false, nil
	}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return Contributor{}, false, err
	}

	dep, err := deps.Best(Dependency, plan.Version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	contributor := Contributor{
		app:         context.Application,
		verifyPath:  filepath.Join(context.Buildpack.Root, "bin", "verify"),
		launchLayer: context.Layers,
		nginxLayer:  context.Layers.DependencyLayer(dep),
	}

	if _, ok := plan.Metadata["launch"]; ok {
		contributor.launchContribution = true
	}

	return contributor, true, nil
}

// Contribute will install NGINX, configure required env variables & set a start command
func (c Contributor) Contribute() error {
	return c.nginxLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.Body("Expanding to %s", layer.Root)
		if err := helper.ExtractTarGz(artifact, layer.Root, 2); err != nil {
			return err
		}

		if err := helper.CopyFile(c.verifyPath, filepath.Join(layer.Root, "bin", "verify")); err != nil {
			return err
		}

		if err := layer.AppendPathSharedEnv("PATH", filepath.Join(layer.Root, "sbin")); err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Join(c.app.Root, "logs"), 0755); err != nil {
			return err
		}

		nginxConfPath := filepath.Join(c.app.Root, "nginx.conf")
		if err := CheckPortExistsInConf(nginxConfPath, layer.Logger); err != nil {
			return err
		}

		appModsPath := filepath.Join(c.app.Root, "modules")
		pkgModsPath := filepath.Join(layer.Root, "modules")
		varifyCmd := fmt.Sprintf(`verify "%s" "%s" "%s"`, nginxConfPath, appModsPath, pkgModsPath)
		nginxCmd := fmt.Sprintf(`nginx -p $PWD -c "%s"`, nginxConfPath)

		return c.launchLayer.WriteApplicationMetadata(layers.Metadata{
			Processes: []layers.Process{{"web", fmt.Sprintf(`%s && %s`, varifyCmd, nginxCmd)}},
		})
	}, c.flags()...)
}

func (c Contributor) flags() []layers.Flag {
	var flags []layers.Flag

	if c.launchContribution {
		flags = append(flags, layers.Launch)
	}

	return flags
}

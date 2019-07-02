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

package php

import (
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

// Contributor represents a PHP contribution by the buildpack
type Contributor struct {
	launchContribution bool
	buildContribution  bool
	phpLayer           layers.DependencyLayer
	appRoot            string
}

// NewContributor creates a new Contributor instance. willContribute is true if build plan contains "php-binary" dependency, otherwise false.
func NewContributor(context build.Build) (c Contributor, willContribute bool, err error) {
	plan, wantDependency := context.BuildPlan[Dependency]
	if !wantDependency {
		context.Logger.SubsequentLine("Dependency not wanted, skipping")
		return Contributor{}, false, nil
	}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return Contributor{}, false, err
	}

	if plan.Version == "" {
		context.Logger.SubsequentLine("Dependency version not specified, but is required")
		return Contributor{}, false, nil
	}
	dep, err := deps.Best(Dependency, plan.Version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	contributor := Contributor{
		appRoot:  context.Application.Root,
		phpLayer: context.Layers.DependencyLayer(dep),
	}

	if _, ok := plan.Metadata["launch"]; ok {
		contributor.launchContribution = true
	}

	if _, ok := plan.Metadata["build"]; ok {
		contributor.buildContribution = true
	}

	return contributor, true, nil
}

// Contribute contributes an expanded PHP to a cache layer.
func (c Contributor) Contribute() error {
	return c.phpLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.SubsequentLine("Expanding to %s", layer.Root)
		if err := helper.ExtractTarGz(artifact, layer.Root, 1); err != nil {
			return err
		}

		if err := layer.AppendPathSharedEnv("PATH", filepath.Join(layer.Root, "sbin")); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("MIBDIRS", filepath.Join(layer.Root, "mibs")); err != nil {
			return err
		}

		return nil
	}, c.flags()...)
}

func (c Contributor) flags() []layers.Flag {
	var flags []layers.Flag

	if c.buildContribution {
		flags = append(flags, layers.Build, layers.Cache)
	}

	if c.launchContribution {
		flags = append(flags, layers.Launch)
	}

	return flags
}

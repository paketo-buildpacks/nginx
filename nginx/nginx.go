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
	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"
	yaml "gopkg.in/yaml.v2"
)

const (
	// Dependency is the key used in the build plan by this buildpack
	Dependency = "nginx"
	Mainline = "mainline"
	Stable = "stable"
)

// BuildpackYAML defines configuration options allowed to end users
type BuildpackYAML struct {
	Config Config `yaml:"nginx"`
}

// Config is used by BuildpackYAML and defines NGINX specific config options available to users
type Config struct {
	Version string `yaml:"version"`
}

// LoadBuildpackYAML reads `buildpack.yml` and NGINX specific config options in it
func LoadBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")
	if exists, err := helper.FileExists(configFile); err != nil {
		return BuildpackYAML{}, err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return BuildpackYAML{}, err
		}
		defer file.Close()

		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return BuildpackYAML{}, err
		}

		err = yaml.Unmarshal(contents, &buildpackYAML)
		if err != nil {
			return BuildpackYAML{}, err
		}
	}
	return buildpackYAML, nil
}

// LoadMainlineVersion out of buildpack.toml
func LoadMainlineVersion(metadata buildpack.Metadata) string {
	versionLines, ok := metadata["version-lines"]
	if ok {
		versionLines := versionLines.(map[string]string)
		mainlineVersion, ok := versionLines[Mainline]
		if ok {
			return mainlineVersion
		}
	}
	return ""
}

// LoadStableVersion out of buildpack.toml
func LoadStableVersion(metadata buildpack.Metadata) string {
	versionLines, ok := metadata["version-lines"]
	if ok {
		versionLines := versionLines.(map[string]string)
		mainlineVersion, ok := versionLines[Stable]
		if ok {
			return mainlineVersion
		}
	}
	return ""
}

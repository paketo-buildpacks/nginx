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
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
		versionLines := versionLines.(map[string]interface{})
		mainlineVersion, ok := versionLines[Mainline]
		if ok {
			return mainlineVersion.(string)
		}
	}
	return ""
}

// LoadStableVersion out of buildpack.toml
func LoadStableVersion(metadata buildpack.Metadata) string {
	versionLines, ok := metadata["version-lines"]
	if ok {
		versionLines := versionLines.(map[string]interface{})
		mainlineVersion, ok := versionLines[Stable]
		if ok {
			return mainlineVersion.(string)
		}
	}
	return ""
}

// CheckPortExistsInConf will validate that a `listen {{port}}` directive has been added by the user, if not it prints a warning to the user
func CheckPortExistsInConf(nginxConfPath string, log logger.Logger) error {
	conf, err := ioutil.ReadFile(nginxConfPath)
	if err != nil {
		return err
	}

	if ! strings.Contains(string(conf), "listen {{port}}") && ! strings.Contains(string(conf), "listen 8080") {
		log.BodyWarning("No `listen {{port}}` directive in nginx.conf, your app may not start.")
	}

	return nil
}
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
	"os"
	"path/filepath"

	"github.com/cloudfoundry/nginx-cnb/nginx"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/php-dist-cnb/php"

	"github.com/cloudfoundry/libcfbuildpack/detect"
)

func main() {
	detectionContext, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to run detection: %s", err)
		os.Exit(101)
	}

	if err := detectionContext.BuildPlan.Init(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize Build Plan: %s\n", err)
		os.Exit(101)
	}

	code, err := runDetect(detectionContext)
	if err != nil {
		detectionContext.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	_, phpIsPossible := context.BuildPlan[php.Dependency]
	if phpIsPossible {
		context.Logger.SubsequentLine("PHP is in the buildplan, so preparing to serve PHP.")
		return context.Pass(buildplan.BuildPlan{})
	}

	nginxConfExists, err := helper.FileExists(filepath.Join(context.Application.Root, "nginx.conf"))
	if err != nil {
		return context.Fail(), err
	}

	if !nginxConfExists {
		return context.Fail(), fmt.Errorf("unable to find nginx.conf")
	}

	buildpackYAML, err := nginx.LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	if buildpackYAML.Config.Version == nginx.Mainline {
		mainlineVersion := nginx.LoadMainlineVersion(context.Buildpack.Metadata)
		if mainlineVersion != "" {
			buildpackYAML.Config.Version = mainlineVersion
		}
	} else if buildpackYAML.Config.Version == nginx.Stable {
		stableVersion := nginx.LoadStableVersion(context.Buildpack.Metadata)
		if stableVersion != "" {
			buildpackYAML.Config.Version = stableVersion
		}
	}

	return context.Pass(buildplan.BuildPlan{
		nginx.Dependency: buildplan.Dependency{
			Version:  buildpackYAML.Config.Version,
			Metadata: buildplan.Metadata{"launch": true},
		},
	})
}

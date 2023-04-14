package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface DependencyService --output fakes/dependency_service.go
type DependencyService interface {
	Resolve(path, name, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

//go:generate faux --interface Calculator --output fakes/calculator.go
type Calculator interface {
	Sum(paths ...string) (string, error)
}

//go:generate faux --interface ConfigGenerator --output fakes/config_generator.go
type ConfigGenerator interface {
	Generate(config Configuration) error
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	GenerateFromDependency(dependency postal.Dependency, dir string) (sbom.SBOM, error)
}

func Build(config Configuration,
	dependencyService DependencyService,
	configGenerator ConfigGenerator,
	calculator Calculator,
	sbomGenerator SBOMGenerator,
	logger scribe.Emitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		planner := draft.NewPlanner()

		logger.Process("Resolving Nginx Server version")
		entry, sortedEntries := planner.Resolve("nginx", context.Plan.Entries, []interface{}{
			"BP_NGINX_VERSION",
			"buildpack.yml",
			"buildpack.toml",
		})
		logger.Candidates(sortedEntries)

		entryVersion, _ := entry.Metadata["version"].(string)
		dependency, err := dependencyService.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry.Name, entryVersion, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency, clock.Now())

		versionSource, _ := entry.Metadata["version-source"].(string)
		if versionSource == "buildpack.yml" {
			nextMajorVersion := semver.MustParse(context.BuildpackInfo.Version).IncMajor()
			logger.Subprocess("WARNING: Setting the server version through buildpack.yml will be deprecated soon in Nginx Server Buildpack v%s.", nextMajorVersion.String())
			logger.Subprocess("Please specify the version through the $BP_NGINX_VERSION environment variable instead. See docs for more information.")
			logger.Break()
		}

		if !filepath.IsAbs(config.NGINXConfLocation) {
			config.NGINXConfLocation = filepath.Join(context.WorkingDir, config.NGINXConfLocation)
		}

		if config.WebServer == "nginx" {
			err = configGenerator.Generate(config)
			if err != nil {
				return packit.BuildResult{}, fmt.Errorf("failed to generate nginx.conf : %w", err)
			}
		}

		var hasNGINXConf bool
		if _, err := os.Stat(config.NGINXConfLocation); err == nil {
			hasNGINXConf = true
		}

		if hasNGINXConf {
			confs, err := getIncludedConfs(config.NGINXConfLocation)
			if err != nil {
				return packit.BuildResult{}, fmt.Errorf("failed to find configuration files: %w", err)
			}

			for _, path := range append([]string{config.NGINXConfLocation}, confs...) {
				info, err := os.Stat(path)
				if err != nil {
					return packit.BuildResult{}, fmt.Errorf("failed to stat configuration files: %w", err)
				}

				err = os.Chmod(path, info.Mode()|0060)
				if err != nil {
					return packit.BuildResult{}, fmt.Errorf("failed to chmod configuration files: %w", err)
				}
			}
		}

		layer, err := context.Layers.Get(NGINX)
		if err != nil {
			return packit.BuildResult{}, err
		}

		bom := dependencyService.GenerateBillOfMaterials(dependency)
		launch, build := planner.MergeLayerTypes("nginx", context.Plan.Entries)

		var buildMetadata packit.BuildMetadata
		if build {
			buildMetadata.BOM = bom
		}

		var launchMetadata packit.LaunchMetadata
		if launch && hasNGINXConf {
			command := "nginx"
			args := []string{
				"-p", context.WorkingDir,
				"-c", config.NGINXConfLocation,
				"-g", "pid /tmp/nginx.pid;",
			}
			launchMetadata.Processes = []packit.Process{
				{
					Type:    "web",
					Command: command,
					Args:    args,
					Default: true,
					Direct:  true,
				},
			}
			launchMetadata.BOM = bom

			if config.LiveReloadEnabled {
				launchMetadata.Processes = []packit.Process{
					{
						Type:    "web",
						Command: "watchexec",
						Args: append([]string{
							"--restart",
							"--watch", context.WorkingDir,
							"--shell", "none",
							"--",
							command,
						}, args...),
						Default: true,
						Direct:  true,
					},
					{
						Type:    "no-reload",
						Command: command,
						Args:    args,
						Direct:  true,
					},
				}
			}
		}

		configureBinPath := filepath.Join(context.CNBPath, "bin", "configure")
		currConfigureBinChecksum, err := calculator.Sum(configureBinPath)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("checksum failed for file %s: %w", configureBinPath, err)
		}

		if !shouldInstall(layer.Metadata, currConfigureBinChecksum, dependency.Checksum) {
			logger.Process("Reusing cached layer %s", layer.Path)
			logger.Break()

			layer.Launch, layer.Build = launch, build

			return packit.BuildResult{
				Layers: []packit.Layer{layer},
				Build:  buildMetadata,
				Launch: launchMetadata,
			}, nil
		}

		logger.Process("Executing build process")

		layer, err = layer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		layer.Launch, layer.Build = launch, build

		logger.Subprocess("Installing Nginx Server %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencyService.Deliver(dependency, context.CNBPath, layer.Path, context.Platform.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		layer.Metadata = map[string]interface{}{
			DepKey:          dependency.Checksum,
			ConfigureBinKey: currConfigureBinChecksum,
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		layer.SharedEnv.Append("PATH", filepath.Join(layer.Path, "sbin"), string(os.PathListSeparator))
		layer.LaunchEnv.Default("EXECD_CONF", config.NGINXConfLocation)
		layer.ExecD = []string{configureBinPath}

		if config.WebServer == "nginx" {
			layer.LaunchEnv.Default("APP_ROOT", context.WorkingDir)
			layer.LaunchEnv.Default("PORT", "8080")
		}

		logger.EnvironmentVariables(layer)

		logger.LaunchProcesses(launchMetadata.Processes)

		logger.GeneratingSBOM(layer.Path)
		var sbomContent sbom.SBOM
		duration, err = clock.Measure(func() error {
			sbomContent, err = sbomGenerator.GenerateFromDependency(dependency, layer.Path)
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)
		layer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		return packit.BuildResult{
			Layers: []packit.Layer{layer},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}

func shouldInstall(layerMetadata map[string]interface{}, configBinChecksum, dependencyChecksum string) bool {
	prevDepChecksum, depOk := layerMetadata[DepKey].(string)
	prevBinChecksum, binOk := layerMetadata[ConfigureBinKey].(string)
	if !depOk || !binOk {
		return true
	}

	if !cargo.Checksum(dependencyChecksum).Match(cargo.Checksum(prevDepChecksum)) {
		return true
	}

	if !cargo.Checksum(configBinChecksum).Match(cargo.Checksum(prevBinChecksum)) {
		return true
	}

	return false
}

var IncludeConfRegexp = regexp.MustCompile(`include\s+(\S*.conf);`)

func getIncludedConfs(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file (%s): %w", path, err)
	}

	var files []string
	for _, match := range IncludeConfRegexp.FindAllStringSubmatch(string(content), -1) {
		if len(match) == 2 {
			glob := match[1]
			if !filepath.IsAbs(glob) {
				glob = filepath.Join(filepath.Dir(path), glob)
			}

			matches, err := filepath.Glob(glob)
			if err != nil {
				return nil, fmt.Errorf("failed to get 'include' files for %s: %w", glob, err)
			}

			files = append(files, matches...)
		}
	}

	return files, nil
}

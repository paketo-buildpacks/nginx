package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(string, []packit.BuildpackPlanEntry, []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

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

//go:generate faux --interface Bindings --output fakes/binding_resolver.go
type Bindings interface {
	Resolve(typ, provider, platformDir string) ([]servicebindings.Binding, error)
}

//go:generate faux --interface ConfigGenerator --output fakes/config_generator.go
type ConfigGenerator interface {
	Generate(env BuildEnvironment) error
}

type BuildEnvironment struct {
	BasicAuthFile             string
	ConfLocation              string `env:"BP_NGINX_CONF_LOCATION"`
	NginxVersion              string `env:"BP_NGINX_VERSION"`
	Reload                    bool   `env:"BP_LIVE_RELOAD_ENABLED"`
	WebServer                 string `env:"BP_WEB_SERVER"`
	WebServerForceHTTPS       bool   `env:"BP_WEB_SERVER_FORCE_HTTPS"`
	WebServerPushStateEnabled bool   `env:"BP_WEB_SERVER_ENABLE_PUSH_STATE"`
	WebServerRoot             string `env:"BP_WEB_SERVER_ROOT"`
	WebServerLocationPath     string `env:"BP_WEB_SERVER_LOCATION_PATH"`
}

func Build(buildEnv BuildEnvironment,
	entryResolver EntryResolver,
	dependencyService DependencyService,
	bindings Bindings,
	config ConfigGenerator,
	calculator Calculator,
	logger scribe.Emitter,
	clock chronos.Clock) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Process("Resolving Nginx Server version")

		priorities := []interface{}{
			"BP_NGINX_VERSION",
			"buildpack.yml",
			"buildpack.toml",
		}
		entry, sortedEntries := entryResolver.Resolve("nginx", context.Plan.Entries, priorities)
		entryVersion, _ := entry.Metadata["version"].(string)

		logger.Candidates(sortedEntries)

		dependency, err := dependencyService.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry.Name, entryVersion, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency, clock.Now())

		versionSource := entry.Metadata["version-source"]
		if versionSource != nil {
			if versionSource.(string) == "buildpack.yml" {
				nextMajorVersion := semver.MustParse(context.BuildpackInfo.Version).IncMajor()
				logger.Subprocess("WARNING: Setting the server version through buildpack.yml will be deprecated soon in Nginx Server Buildpack v%s.", nextMajorVersion.String())
				logger.Subprocess("Please specify the version through the $BP_NGINX_VERSION environment variable instead. See docs for more information.")
				logger.Break()
			}
		}

		err = os.MkdirAll(filepath.Join(context.WorkingDir, "logs"), os.ModePerm)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to create logs dir : %w", err)
		}

		buildEnv.ConfLocation = cleanNginxConfLocation(buildEnv.ConfLocation, context.WorkingDir)

		if buildEnv.WebServer == "nginx" {
			bindingSet, err := bindings.Resolve("htpasswd", "", context.Platform.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}

			if len(bindingSet) > 1 {
				return packit.BuildResult{}, fmt.Errorf("binding resolver found more than one binding of type 'htpasswd'")
			}

			if len(bindingSet) == 1 {
				if _, ok := bindingSet[0].Entries[".htpasswd"]; !ok {
					return packit.BuildResult{}, fmt.Errorf("binding of type 'htpasswd' does not contain required entry '.htpasswd'")
				}
				buildEnv.BasicAuthFile = filepath.Join(bindingSet[0].Path, ".htpasswd")
			}

			err = config.Generate(buildEnv)
			if err != nil {
				return packit.BuildResult{}, fmt.Errorf("failed to generate nginx.conf : %w", err)
			}
		}

		layer, err := context.Layers.Get(NGINX)
		if err != nil {
			return packit.BuildResult{}, err
		}

		bom := dependencyService.GenerateBillOfMaterials(dependency)
		launch, build := entryResolver.MergeLayerTypes("nginx", context.Plan.Entries)

		var buildMetadata packit.BuildMetadata
		if build {
			buildMetadata.BOM = bom
		}

		var launchMetadata packit.LaunchMetadata

		if launch {
			command := "nginx"
			args := []string{
				"-p",
				context.WorkingDir,
				"-c",
				buildEnv.ConfLocation,
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

			if buildEnv.Reload {
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
		currConfigureBinSHA256, err := calculator.Sum(configureBinPath)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("checksum failed for file %s: %w", configureBinPath, err)
		}

		if !shouldInstall(layer.Metadata, currConfigureBinSHA256, dependency.SHA256) {
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
			DepKey:          dependency.SHA256,
			ConfigureBinKey: currConfigureBinSHA256,
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		layer.SharedEnv.Append("PATH", filepath.Join(layer.Path, "sbin"), string(os.PathListSeparator))

		layer.LaunchEnv.Override("EXECD_CONF", buildEnv.ConfLocation)
		execdDir := filepath.Join(layer.Path, "exec.d")
		err = os.MkdirAll(execdDir, os.ModePerm)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = fs.Copy(configureBinPath, filepath.Join(execdDir, "0-configure"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		if buildEnv.WebServer == "nginx" {
			layer.LaunchEnv.Override("APP_ROOT", context.WorkingDir)
		}

		logger.EnvironmentVariables(layer)

		logger.LaunchProcesses(launchMetadata.Processes)
		return packit.BuildResult{
			Layers: []packit.Layer{layer},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}

func shouldInstall(layerMetadata map[string]interface{}, configBinSHA256, dependencySHA256 string) bool {
	prevDepSHA256, depOk := layerMetadata[DepKey].(string)
	prevBinSHA256, binOk := layerMetadata[ConfigureBinKey].(string)
	if !depOk || !binOk {
		return true
	}

	if dependencySHA256 != prevDepSHA256 {
		return true
	}

	if configBinSHA256 != prevBinSHA256 {
		return true
	}

	return false
}

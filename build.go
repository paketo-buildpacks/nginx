package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(string, []packit.BuildpackPlanEntry, []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface DependencyService --output fakes/dependency_service.go
type DependencyService interface {
	Resolve(path, name, version, stack string) (postal.Dependency, error)
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

//go:generate faux --interface ProfileDWriter --output fakes/profiled_writer.go
type ProfileDWriter interface {
	Write(layerDir, scriptName, scriptContents string) error
}

//go:generate faux --interface Calculator --output fakes/calculator.go
type Calculator interface {
	Sum(paths ...string) (string, error)
}

func Build(entryResolver EntryResolver, dependencyService DependencyService, profileDWriter ProfileDWriter, calculator Calculator, logger scribe.Emitter, clock chronos.Clock) packit.BuildFunc {
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

		layer, err := context.Layers.Get(NGINX)
		if err != nil {
			return packit.BuildResult{}, err
		}

		nginxConfPath := filepath.Join(context.WorkingDir, ConfFile)
		configureBinPath := filepath.Join(context.CNBPath, "bin", "configure")
		currConfigureBinSHA256, err := calculator.Sum(configureBinPath)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("checksum failed for file %s: %w", configureBinPath, err)
		}

		bom := dependencyService.GenerateBillOfMaterials(dependency)
		launch, build := entryResolver.MergeLayerTypes("nginx", context.Plan.Entries)

		var buildMetadata packit.BuildMetadata
		if build {
			buildMetadata.BOM = bom
		}

		var launchMetadata packit.LaunchMetadata
		if launch {
			launchMetadata.Processes = []packit.Process{
				{
					Type:    "web",
					Command: fmt.Sprintf(`nginx -p $PWD -c "%s"`, nginxConfPath),
					Default: true,
				},
			}
			launchMetadata.BOM = bom
		}

		if !shouldInstall(layer.Metadata, currConfigureBinSHA256, dependency.SHA256) {
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

		err = os.MkdirAll(filepath.Join(layer.Path, "bin"), os.ModePerm)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = fs.Copy(configureBinPath, filepath.Join(layer.Path, "bin", "configure"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Subprocess("Installing Nginx Server %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencyService.Install(dependency, context.CNBPath, layer.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		layer.SharedEnv.Append("PATH", filepath.Join(layer.Path, "sbin"), ":")
		logger.EnvironmentVariables(layer)

		err = profileDWriter.Write(
			layer.Path,
			"configure.sh",
			fmt.Sprintf(`configure "%s" "%s" "%s"`,
				nginxConfPath,
				filepath.Join(context.WorkingDir, "modules"),
				filepath.Join(layer.Path, "modules"),
			),
		)
		if err != nil {
			return packit.BuildResult{}, err
		}

		layer.Metadata = map[string]interface{}{
			DepKey:          dependency.SHA256,
			ConfigureBinKey: currConfigureBinSHA256,
			"built_at":      clock.Now().Format(time.RFC3339Nano),
		}

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

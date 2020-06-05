package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/postal"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve([]packit.BuildpackPlanEntry) packit.BuildpackPlanEntry
}

//go:generate faux --interface DependencyService --output fakes/dependency_service.go
type DependencyService interface {
	Resolve(path, name, version, stack string) (postal.Dependency, error)
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
}

//go:generate faux --interface ProfileDWriter --output fakes/profiled_writer.go
type ProfileDWriter interface {
	Write(layerDir, scriptName, scriptContents string) error
}

//go:generate faux --interface Calculator --output fakes/calculator.go
type Calculator interface {
	Sum(path string) (string, error)
}

func Build(entryResolver EntryResolver, dependencyService DependencyService, profileDWriter ProfileDWriter, calculator Calculator, logger LogEmitter, clock chronos.Clock) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title(context.BuildpackInfo)

		logger.Process("Resolving Nginx Server version")
		logger.Candidates(context.Plan.Entries)

		entry := entryResolver.Resolve(context.Plan.Entries)

		dependency, err := dependencyService.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry.Name, entry.Version, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency.Version)

		err = os.MkdirAll(filepath.Join(context.WorkingDir, "logs"), os.ModePerm)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to create logs dir : %w", err)
		}

		nginxLayer, err := context.Layers.Get(NGINX, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		nginxConfPath := filepath.Join(context.WorkingDir, ConfFile)
		defaultStartProcesses := []packit.Process{
			{
				Type:    "web",
				Command: fmt.Sprintf(`nginx -p $PWD -c "%s"`, nginxConfPath),
			},
		}

		configureBinPath := filepath.Join(context.CNBPath, "bin", "configure")
		currConfigureBinSHA256, err := calculator.Sum(configureBinPath)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("checksum failed for file %s: %w", configureBinPath, err)
		}

		if !shouldInstall(nginxLayer.Metadata, currConfigureBinSHA256, dependency.SHA256) {
			return packit.BuildResult{
				Plan: context.Plan,
				Layers: []packit.Layer{
					nginxLayer,
				},
				Processes: defaultStartProcesses,
			}, nil
		}

		logger.Break()
		logger.Process("Executing build process")

		err = nginxLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = os.MkdirAll(filepath.Join(nginxLayer.Path, "bin"), os.ModePerm)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = fs.Copy(configureBinPath, filepath.Join(nginxLayer.Path, "bin", "configure"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Subprocess("Installing Nginx Server %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencyService.Install(dependency, context.CNBPath, nginxLayer.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		logger.Process("Configuring environment")
		nginxLayer.SharedEnv.Append("PATH", filepath.Join(nginxLayer.Path, "sbin"), ":")
		logger.Environment(nginxLayer.SharedEnv)

		profileDWriter.Write(
			nginxLayer.Path,
			"configure.sh",
			fmt.Sprintf(`configure "%s" "%s" "%s"`,
				nginxConfPath,
				filepath.Join(context.WorkingDir, "modules"),
				filepath.Join(nginxLayer.Path, "modules"),
			),
		)

		nginxLayer.Metadata = map[string]interface{}{
			DepKey:          dependency.SHA256,
			ConfigureBinKey: currConfigureBinSHA256,
			"built_at":      clock.Now().Format(time.RFC3339Nano),
		}

		return packit.BuildResult{
			Plan: context.Plan,
			Layers: []packit.Layer{
				nginxLayer,
			},
			Processes: defaultStartProcesses,
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

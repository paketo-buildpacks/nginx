package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/paketo-buildpacks/packit/v2"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ResolveVersion(cnbPath, version string) (resultVersion string, err error)
	ParseYml(workDir string) (ymlVersion string, exists bool, err error)
}

type BuildPlanMetadata struct {
	Version       string `toml:"version,omitempty"`
	VersionSource string `toml:"version-source,omitempty"`
	Launch        bool   `toml:"launch"`
}

func Detect(versionParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: NGINX},
				},
			},
		}

		_, err := os.Stat(getNginxConfLocation(context.WorkingDir))
		if err != nil {
			if os.IsNotExist(err) {
				return plan, nil
			}

			return packit.DetectResult{}, fmt.Errorf("failed to stat nginx.conf: %w", err)
		}

		version, envVarExists := os.LookupEnv("BP_NGINX_VERSION")
		var requirements []packit.BuildPlanRequirement

		if envVarExists {
			version, err = versionParser.ResolveVersion(context.CNBPath, version)
			if err != nil {
				return packit.DetectResult{}, err
			}
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: NGINX,
				Metadata: BuildPlanMetadata{
					Version:       version,
					VersionSource: "BP_NGINX_VERSION",
					Launch:        true,
				},
			})
		}

		ymlVersion, ymlExists, err := versionParser.ParseYml(context.WorkingDir)

		if ymlExists {
			version, err = versionParser.ResolveVersion(context.CNBPath, ymlVersion)
			if err != nil {
				return packit.DetectResult{}, err
			}
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: NGINX,
				Metadata: BuildPlanMetadata{
					Version:       version,
					VersionSource: "buildpack.yml",
					Launch:        true,
				},
			})
		}

		if !envVarExists && !ymlExists {
			version, err = versionParser.ResolveVersion(context.CNBPath, "")
			if err != nil {
				return packit.DetectResult{}, err
			}
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: NGINX,
				Metadata: BuildPlanMetadata{
					Version:       version,
					VersionSource: "buildpack.toml",
					Launch:        true,
				},
			})
		}

		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("parsing version failed: %w", err)
		}

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.DetectResult{}, err
		}

		if shouldReload {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "watchexec",
				Metadata: map[string]interface{}{
					"launch": true,
				},
			})
		}

		plan.Plan.Requires = requirements

		return plan, nil
	}
}

func checkLiveReloadEnabled() (bool, error) {
	if reload, ok := os.LookupEnv("BP_LIVE_RELOAD_ENABLED"); ok {
		shouldEnableReload, err := strconv.ParseBool(reload)
		if err != nil {
			return false, fmt.Errorf("failed to parse BP_LIVE_RELOAD_ENABLED value %s: %w", reload, err)
		}
		return shouldEnableReload, nil
	}
	return false, nil
}

func getNginxConfLocation(workingDir string) string {
	if customPath, ok := os.LookupEnv("BP_NGINX_CONF_LOCATION"); ok {
		if filepath.IsAbs(customPath) {
			return customPath
		}
		return filepath.Join(workingDir, customPath)
	}
	return filepath.Join(workingDir, ConfFile)
}

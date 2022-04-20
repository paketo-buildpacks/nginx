package nginx

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
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

func Detect(buildEnv BuildEnvironment, versionParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: NGINX},
				},
			},
		}

		confExists, err := fs.Exists(getNginxConfLocation(context.WorkingDir))
		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("failed to stat nginx.conf: %w", err)
		}
		if !confExists && buildEnv.WebServer != "nginx" {
			return plan, nil
		}

		var requirements []packit.BuildPlanRequirement
		var version string
		if buildEnv.NginxVersion != "" {
			version, err = versionParser.ResolveVersion(context.CNBPath, buildEnv.NginxVersion)
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

		if buildEnv.NginxVersion == "" && !ymlExists {
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

		if buildEnv.Reload {
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

func getNginxConfLocation(workingDir string) string {
	if customPath, ok := os.LookupEnv("BP_NGINX_CONF_LOCATION"); ok {
		if filepath.IsAbs(customPath) {
			return customPath
		}
		return filepath.Join(workingDir, customPath)
	}
	return filepath.Join(workingDir, ConfFile)
}

package nginx

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
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

		_, err := os.Stat(filepath.Join(context.WorkingDir, ConfFile))
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

		plan.Plan.Requires = requirements

		return plan, nil
	}
}

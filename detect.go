package nginx

import (
	"fmt"
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

func Detect(config Configuration, versionParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: NGINX},
				},
			},
		}

		if !filepath.IsAbs(config.NGINXConfLocation) {
			config.NGINXConfLocation = filepath.Join(context.WorkingDir, config.NGINXConfLocation)
		}

		confExists, err := fs.Exists(config.NGINXConfLocation)
		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("failed to stat nginx.conf: %w", err)
		}
		if !confExists && config.WebServer != "nginx" {
			return plan, nil
		}

		var requirements []packit.BuildPlanRequirement
		var version string
		if config.NGINXVersion != "" {
			version, err = versionParser.ResolveVersion(context.CNBPath, config.NGINXVersion)
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

		if config.NGINXVersion == "" && !ymlExists {
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

		if config.LiveReloadEnabled {
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

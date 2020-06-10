package nginx

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(workingDir, cnbPath string) (version, versionSource string, err error)
}

type BuildPlanMetadata struct {
	VersionSource string `toml:"version-source,omitempty"`
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

		version, versionSource, err := versionParser.ParseVersion(context.WorkingDir, context.CNBPath)
		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("parsing version failed: %w", err)
		}

		plan.Plan.Requires = []packit.BuildPlanRequirement{
			{
				Name:    NGINX,
				Version: version,
				Metadata: BuildPlanMetadata{
					VersionSource: versionSource,
				},
			},
		}
		return plan, nil
	}
}

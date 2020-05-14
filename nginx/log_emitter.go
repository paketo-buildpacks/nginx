package nginx

import (
	"io"
	"strconv"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

type LogEmitter struct {
	scribe.Logger
}

func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Logger: scribe.NewLogger(output),
	}
}

func (e LogEmitter) Title(info packit.BuildpackInfo) {
	e.Logger.Title("%s %s", info.Name, info.Version)
}

func (e LogEmitter) Candidates(entries []packit.BuildpackPlanEntry) {
	e.Logger.Subprocess("Candidate version sources (in priority order):")

	var (
		sources [][2]string
		maxLen  int
	)

	for _, entry := range entries {
		versionSource, ok := entry.Metadata["version-source"].(string)
		if !ok {
			versionSource = "<unknown>"
		}

		if len(versionSource) > maxLen {
			maxLen = len(versionSource)
		}

		sources = append(sources, [2]string{versionSource, entry.Version})
	}

	for _, source := range sources {
		e.Logger.Action(("%-" + strconv.Itoa(maxLen) + "s -> %q"), source[0], source[1])
	}

	e.Logger.Break()
}

func (e LogEmitter) SelectedDependency(entry packit.BuildpackPlanEntry, version string) {
	source, ok := entry.Metadata["version-source"].(string)
	if !ok {
		source = "<unknown>"
	}

	e.Logger.Subprocess("Selected Nginx Server version (using %s): %s", source, version)
}

func (e LogEmitter) Environment(environment packit.Environment) {
	e.Logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(environment))
}

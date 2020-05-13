package nginx

import (
	"sort"

	"github.com/paketo-buildpacks/packit"
)

type PlanEntryResolver struct{}

func NewPlanEntryResolver() PlanEntryResolver {
	return PlanEntryResolver{}
}

func (r PlanEntryResolver) Resolve(entries []packit.BuildpackPlanEntry) packit.BuildpackPlanEntry {
	var (
		priorities = map[string]int{
			"buildpack.yml": 3,
			"":              -1,
		}
	)

	sort.Slice(entries, func(i, j int) bool {
		leftSource := entries[i].Metadata["version-source"]
		left, _ := leftSource.(string)

		rightSource := entries[j].Metadata["version-source"]
		right, _ := rightSource.(string)

		return priorities[left] > priorities[right]
	})

	chosenEntry := entries[0]

	if chosenEntry.Metadata == nil {
		chosenEntry.Metadata = map[string]interface{}{}
	}

	for _, entry := range entries {
		if entry.Metadata["build"] == true {
			chosenEntry.Metadata["build"] = true
		}
	}

	return chosenEntry
}

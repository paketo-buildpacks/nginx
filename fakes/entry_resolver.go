package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit"
)

type EntryResolver struct {
	ResolveCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			String                  string
			BuildpackPlanEntrySlice []packit.BuildpackPlanEntry
			InterfaceSlice          []interface {
			}
		}
		Returns struct {
			BuildpackPlanEntry      packit.BuildpackPlanEntry
			BuildpackPlanEntrySlice []packit.BuildpackPlanEntry
		}
		Stub func(string, []packit.BuildpackPlanEntry, []interface {
		}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	}
}

func (f *EntryResolver) Resolve(param1 string, param2 []packit.BuildpackPlanEntry, param3 []interface {
}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry) {
	f.ResolveCall.Lock()
	defer f.ResolveCall.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.String = param1
	f.ResolveCall.Receives.BuildpackPlanEntrySlice = param2
	f.ResolveCall.Receives.InterfaceSlice = param3
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2, param3)
	}
	return f.ResolveCall.Returns.BuildpackPlanEntry, f.ResolveCall.Returns.BuildpackPlanEntrySlice
}

package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

type DependencyService struct {
	GenerateBillOfMaterialsCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Dependencies []postal.Dependency
		}
		Returns struct {
			BOMEntrySlice []packit.BOMEntry
		}
		Stub func(...postal.Dependency) []packit.BOMEntry
	}
	InstallCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Dependency postal.Dependency
			CnbPath    string
			LayerPath  string
		}
		Returns struct {
			Error error
		}
		Stub func(postal.Dependency, string, string) error
	}
	ResolveCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Path    string
			Name    string
			Version string
			Stack   string
		}
		Returns struct {
			Dependency postal.Dependency
			Error      error
		}
		Stub func(string, string, string, string) (postal.Dependency, error)
	}
}

func (f *DependencyService) GenerateBillOfMaterials(param1 ...postal.Dependency) []packit.BOMEntry {
	f.GenerateBillOfMaterialsCall.mutex.Lock()
	defer f.GenerateBillOfMaterialsCall.mutex.Unlock()
	f.GenerateBillOfMaterialsCall.CallCount++
	f.GenerateBillOfMaterialsCall.Receives.Dependencies = param1
	if f.GenerateBillOfMaterialsCall.Stub != nil {
		return f.GenerateBillOfMaterialsCall.Stub(param1...)
	}
	return f.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice
}
func (f *DependencyService) Install(param1 postal.Dependency, param2 string, param3 string) error {
	f.InstallCall.mutex.Lock()
	defer f.InstallCall.mutex.Unlock()
	f.InstallCall.CallCount++
	f.InstallCall.Receives.Dependency = param1
	f.InstallCall.Receives.CnbPath = param2
	f.InstallCall.Receives.LayerPath = param3
	if f.InstallCall.Stub != nil {
		return f.InstallCall.Stub(param1, param2, param3)
	}
	return f.InstallCall.Returns.Error
}
func (f *DependencyService) Resolve(param1 string, param2 string, param3 string, param4 string) (postal.Dependency, error) {
	f.ResolveCall.mutex.Lock()
	defer f.ResolveCall.mutex.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.Path = param1
	f.ResolveCall.Receives.Name = param2
	f.ResolveCall.Receives.Version = param3
	f.ResolveCall.Receives.Stack = param4
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2, param3, param4)
	}
	return f.ResolveCall.Returns.Dependency, f.ResolveCall.Returns.Error
}

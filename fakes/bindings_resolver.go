package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

type BindingsResolver struct {
	ResolveOneCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Typ         string
			Provider    string
			PlatformDir string
		}
		Returns struct {
			Binding servicebindings.Binding
			Error   error
		}
		Stub func(string, string, string) (servicebindings.Binding, error)
	}
}

func (f *BindingsResolver) ResolveOne(param1 string, param2 string, param3 string) (servicebindings.Binding, error) {
	f.ResolveOneCall.mutex.Lock()
	defer f.ResolveOneCall.mutex.Unlock()
	f.ResolveOneCall.CallCount++
	f.ResolveOneCall.Receives.Typ = param1
	f.ResolveOneCall.Receives.Provider = param2
	f.ResolveOneCall.Receives.PlatformDir = param3
	if f.ResolveOneCall.Stub != nil {
		return f.ResolveOneCall.Stub(param1, param2, param3)
	}
	return f.ResolveOneCall.Returns.Binding, f.ResolveOneCall.Returns.Error
}

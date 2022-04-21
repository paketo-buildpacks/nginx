package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/nginx"
)

type ConfigGenerator struct {
	GenerateCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Env nginx.BuildEnvironment
		}
		Returns struct {
			Error error
		}
		Stub func(nginx.BuildEnvironment) error
	}
}

func (f *ConfigGenerator) Generate(param1 nginx.BuildEnvironment) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.Env = param1
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1)
	}
	return f.GenerateCall.Returns.Error
}

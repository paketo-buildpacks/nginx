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
			Destination string
			Env         nginx.BuildEnvironment
		}
		Returns struct {
			Error error
		}
		Stub func(string, nginx.BuildEnvironment) error
	}
}

func (f *ConfigGenerator) Generate(param1 string, param2 nginx.BuildEnvironment) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.Destination = param1
	f.GenerateCall.Receives.Env = param2
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1, param2)
	}
	return f.GenerateCall.Returns.Error
}

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
			Config nginx.Configuration
		}
		Returns struct {
			Error error
		}
		Stub func(nginx.Configuration) error
	}
}

func (f *ConfigGenerator) Generate(param1 nginx.Configuration) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.Config = param1
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1)
	}
	return f.GenerateCall.Returns.Error
}

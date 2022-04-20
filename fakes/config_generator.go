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
			TemplateSource string
			Destination    string
			Env            nginx.BuildEnvironment
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, nginx.BuildEnvironment) error
	}
}

func (f *ConfigGenerator) Generate(param1 string, param2 string, param3 nginx.BuildEnvironment) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.TemplateSource = param1
	f.GenerateCall.Receives.Destination = param2
	f.GenerateCall.Receives.Env = param3
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1, param2, param3)
	}
	return f.GenerateCall.Returns.Error
}

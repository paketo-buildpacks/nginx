package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/nginx"
)

type BuildEnvironmentParser struct {
	ParseCall struct {
		mutex     sync.Mutex
		CallCount int
		Returns   struct {
			BuildEnvironment nginx.BuildEnvironment
			Error            error
		}
		Stub func() (nginx.BuildEnvironment, error)
	}
}

func (f *BuildEnvironmentParser) Parse() (nginx.BuildEnvironment, error) {
	f.ParseCall.mutex.Lock()
	defer f.ParseCall.mutex.Unlock()
	f.ParseCall.CallCount++
	if f.ParseCall.Stub != nil {
		return f.ParseCall.Stub()
	}
	return f.ParseCall.Returns.BuildEnvironment, f.ParseCall.Returns.Error
}

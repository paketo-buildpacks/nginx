package fakes

import "sync"

type ConfigGenerator struct {
	GenerateCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			TemplateSource string
			Destination    string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *ConfigGenerator) Generate(param1 string, param2 string) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.TemplateSource = param1
	f.GenerateCall.Receives.Destination = param2
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1, param2)
	}
	return f.GenerateCall.Returns.Error
}

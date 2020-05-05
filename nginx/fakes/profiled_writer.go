package fakes

import "sync"

type ProfileDWriter struct {
	WriteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			LayerDir       string
			ScriptName     string
			ScriptContents string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
}

func (f *ProfileDWriter) Write(param1 string, param2 string, param3 string) error {
	f.WriteCall.Lock()
	defer f.WriteCall.Unlock()
	f.WriteCall.CallCount++
	f.WriteCall.Receives.LayerDir = param1
	f.WriteCall.Receives.ScriptName = param2
	f.WriteCall.Receives.ScriptContents = param3
	if f.WriteCall.Stub != nil {
		return f.WriteCall.Stub(param1, param2, param3)
	}
	return f.WriteCall.Returns.Error
}

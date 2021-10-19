package fakes

import "sync"

type VersionParser struct {
	ParseYmlCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkDir string
		}
		Returns struct {
			YmlVersion string
			Exists     bool
			Err        error
		}
		Stub func(string) (string, bool, error)
	}
	ResolveVersionCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			CnbPath string
			Version string
		}
		Returns struct {
			ResultVersion string
			Err           error
		}
		Stub func(string, string) (string, error)
	}
}

func (f *VersionParser) ParseYml(param1 string) (string, bool, error) {
	f.ParseYmlCall.mutex.Lock()
	defer f.ParseYmlCall.mutex.Unlock()
	f.ParseYmlCall.CallCount++
	f.ParseYmlCall.Receives.WorkDir = param1
	if f.ParseYmlCall.Stub != nil {
		return f.ParseYmlCall.Stub(param1)
	}
	return f.ParseYmlCall.Returns.YmlVersion, f.ParseYmlCall.Returns.Exists, f.ParseYmlCall.Returns.Err
}
func (f *VersionParser) ResolveVersion(param1 string, param2 string) (string, error) {
	f.ResolveVersionCall.mutex.Lock()
	defer f.ResolveVersionCall.mutex.Unlock()
	f.ResolveVersionCall.CallCount++
	f.ResolveVersionCall.Receives.CnbPath = param1
	f.ResolveVersionCall.Receives.Version = param2
	if f.ResolveVersionCall.Stub != nil {
		return f.ResolveVersionCall.Stub(param1, param2)
	}
	return f.ResolveVersionCall.Returns.ResultVersion, f.ResolveVersionCall.Returns.Err
}

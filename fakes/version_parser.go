package fakes

import "sync"

type VersionParser struct {
	ParseVersionCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			CnbPath    string
		}
		Returns struct {
			Version       string
			VersionSource string
			Err           error
		}
		Stub func(string, string) (string, string, error)
	}
}

func (f *VersionParser) ParseVersion(param1 string, param2 string) (string, string, error) {
	f.ParseVersionCall.Lock()
	defer f.ParseVersionCall.Unlock()
	f.ParseVersionCall.CallCount++
	f.ParseVersionCall.Receives.WorkingDir = param1
	f.ParseVersionCall.Receives.CnbPath = param2
	if f.ParseVersionCall.Stub != nil {
		return f.ParseVersionCall.Stub(param1, param2)
	}
	return f.ParseVersionCall.Returns.Version, f.ParseVersionCall.Returns.VersionSource, f.ParseVersionCall.Returns.Err
}

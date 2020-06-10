package fakes

import "sync"

type Calculator struct {
	SumCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			String string
			Error  error
		}
		Stub func(string) (string, error)
	}
}

func (f *Calculator) Sum(param1 string) (string, error) {
	f.SumCall.Lock()
	defer f.SumCall.Unlock()
	f.SumCall.CallCount++
	f.SumCall.Receives.Path = param1
	if f.SumCall.Stub != nil {
		return f.SumCall.Stub(param1)
	}
	return f.SumCall.Returns.String, f.SumCall.Returns.Error
}

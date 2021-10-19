package fakes

import "sync"

type Calculator struct {
	SumCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Paths []string
		}
		Returns struct {
			String string
			Error  error
		}
		Stub func(...string) (string, error)
	}
}

func (f *Calculator) Sum(param1 ...string) (string, error) {
	f.SumCall.mutex.Lock()
	defer f.SumCall.mutex.Unlock()
	f.SumCall.CallCount++
	f.SumCall.Receives.Paths = param1
	if f.SumCall.Stub != nil {
		return f.SumCall.Stub(param1...)
	}
	return f.SumCall.Returns.String, f.SumCall.Returns.Error
}

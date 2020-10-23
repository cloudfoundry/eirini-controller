// Code generated by counterfeiter. DO NOT EDIT.
package k8sfakes

import (
	"sync"

	"code.cloudfoundry.org/eirini/k8s"
	v1 "k8s.io/api/batch/v1"
)

type FakeJobDeletingClient struct {
	DeleteStub        func(string, string) error
	deleteMutex       sync.RWMutex
	deleteArgsForCall []struct {
		arg1 string
		arg2 string
	}
	deleteReturns struct {
		result1 error
	}
	deleteReturnsOnCall map[int]struct {
		result1 error
	}
	GetByGUIDStub        func(string, bool) ([]v1.Job, error)
	getByGUIDMutex       sync.RWMutex
	getByGUIDArgsForCall []struct {
		arg1 string
		arg2 bool
	}
	getByGUIDReturns struct {
		result1 []v1.Job
		result2 error
	}
	getByGUIDReturnsOnCall map[int]struct {
		result1 []v1.Job
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeJobDeletingClient) Delete(arg1 string, arg2 string) error {
	fake.deleteMutex.Lock()
	ret, specificReturn := fake.deleteReturnsOnCall[len(fake.deleteArgsForCall)]
	fake.deleteArgsForCall = append(fake.deleteArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("Delete", []interface{}{arg1, arg2})
	fake.deleteMutex.Unlock()
	if fake.DeleteStub != nil {
		return fake.DeleteStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.deleteReturns
	return fakeReturns.result1
}

func (fake *FakeJobDeletingClient) DeleteCallCount() int {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	return len(fake.deleteArgsForCall)
}

func (fake *FakeJobDeletingClient) DeleteCalls(stub func(string, string) error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = stub
}

func (fake *FakeJobDeletingClient) DeleteArgsForCall(i int) (string, string) {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	argsForCall := fake.deleteArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeJobDeletingClient) DeleteReturns(result1 error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = nil
	fake.deleteReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeJobDeletingClient) DeleteReturnsOnCall(i int, result1 error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = nil
	if fake.deleteReturnsOnCall == nil {
		fake.deleteReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeJobDeletingClient) GetByGUID(arg1 string, arg2 bool) ([]v1.Job, error) {
	fake.getByGUIDMutex.Lock()
	ret, specificReturn := fake.getByGUIDReturnsOnCall[len(fake.getByGUIDArgsForCall)]
	fake.getByGUIDArgsForCall = append(fake.getByGUIDArgsForCall, struct {
		arg1 string
		arg2 bool
	}{arg1, arg2})
	fake.recordInvocation("GetByGUID", []interface{}{arg1, arg2})
	fake.getByGUIDMutex.Unlock()
	if fake.GetByGUIDStub != nil {
		return fake.GetByGUIDStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getByGUIDReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeJobDeletingClient) GetByGUIDCallCount() int {
	fake.getByGUIDMutex.RLock()
	defer fake.getByGUIDMutex.RUnlock()
	return len(fake.getByGUIDArgsForCall)
}

func (fake *FakeJobDeletingClient) GetByGUIDCalls(stub func(string, bool) ([]v1.Job, error)) {
	fake.getByGUIDMutex.Lock()
	defer fake.getByGUIDMutex.Unlock()
	fake.GetByGUIDStub = stub
}

func (fake *FakeJobDeletingClient) GetByGUIDArgsForCall(i int) (string, bool) {
	fake.getByGUIDMutex.RLock()
	defer fake.getByGUIDMutex.RUnlock()
	argsForCall := fake.getByGUIDArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeJobDeletingClient) GetByGUIDReturns(result1 []v1.Job, result2 error) {
	fake.getByGUIDMutex.Lock()
	defer fake.getByGUIDMutex.Unlock()
	fake.GetByGUIDStub = nil
	fake.getByGUIDReturns = struct {
		result1 []v1.Job
		result2 error
	}{result1, result2}
}

func (fake *FakeJobDeletingClient) GetByGUIDReturnsOnCall(i int, result1 []v1.Job, result2 error) {
	fake.getByGUIDMutex.Lock()
	defer fake.getByGUIDMutex.Unlock()
	fake.GetByGUIDStub = nil
	if fake.getByGUIDReturnsOnCall == nil {
		fake.getByGUIDReturnsOnCall = make(map[int]struct {
			result1 []v1.Job
			result2 error
		})
	}
	fake.getByGUIDReturnsOnCall[i] = struct {
		result1 []v1.Job
		result2 error
	}{result1, result2}
}

func (fake *FakeJobDeletingClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	fake.getByGUIDMutex.RLock()
	defer fake.getByGUIDMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeJobDeletingClient) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ k8s.JobDeletingClient = new(FakeJobDeletingClient)

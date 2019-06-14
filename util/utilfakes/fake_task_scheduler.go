// Code generated by counterfeiter. DO NOT EDIT.
package utilfakes

import (
	"sync"

	"code.cloudfoundry.org/eirini/util"
)

type FakeTaskScheduler struct {
	ScheduleStub        func(util.Task)
	scheduleMutex       sync.RWMutex
	scheduleArgsForCall []struct {
		arg1 util.Task
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeTaskScheduler) Schedule(arg1 util.Task) {
	fake.scheduleMutex.Lock()
	fake.scheduleArgsForCall = append(fake.scheduleArgsForCall, struct {
		arg1 util.Task
	}{arg1})
	fake.recordInvocation("Schedule", []interface{}{arg1})
	fake.scheduleMutex.Unlock()
	if fake.ScheduleStub != nil {
		fake.ScheduleStub(arg1)
	}
}

func (fake *FakeTaskScheduler) ScheduleCallCount() int {
	fake.scheduleMutex.RLock()
	defer fake.scheduleMutex.RUnlock()
	return len(fake.scheduleArgsForCall)
}

func (fake *FakeTaskScheduler) ScheduleCalls(stub func(util.Task)) {
	fake.scheduleMutex.Lock()
	defer fake.scheduleMutex.Unlock()
	fake.ScheduleStub = stub
}

func (fake *FakeTaskScheduler) ScheduleArgsForCall(i int) util.Task {
	fake.scheduleMutex.RLock()
	defer fake.scheduleMutex.RUnlock()
	argsForCall := fake.scheduleArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeTaskScheduler) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.scheduleMutex.RLock()
	defer fake.scheduleMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeTaskScheduler) recordInvocation(key string, args []interface{}) {
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

var _ util.TaskScheduler = new(FakeTaskScheduler)

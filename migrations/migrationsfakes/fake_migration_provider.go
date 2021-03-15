// Code generated by counterfeiter. DO NOT EDIT.
package migrationsfakes

import (
	"sync"

	"code.cloudfoundry.org/eirini/migrations"
)

type FakeMigrationProvider struct {
	GetLatestMigrationIndexStub        func() int
	getLatestMigrationIndexMutex       sync.RWMutex
	getLatestMigrationIndexArgsForCall []struct{}
	getLatestMigrationIndexReturns     struct {
		result1 int
	}
	getLatestMigrationIndexReturnsOnCall map[int]struct {
		result1 int
	}
	ProvideStub        func() []migrations.MigrationStep
	provideMutex       sync.RWMutex
	provideArgsForCall []struct{}
	provideReturns     struct {
		result1 []migrations.MigrationStep
	}
	provideReturnsOnCall map[int]struct {
		result1 []migrations.MigrationStep
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeMigrationProvider) GetLatestMigrationIndex() int {
	fake.getLatestMigrationIndexMutex.Lock()
	ret, specificReturn := fake.getLatestMigrationIndexReturnsOnCall[len(fake.getLatestMigrationIndexArgsForCall)]
	fake.getLatestMigrationIndexArgsForCall = append(fake.getLatestMigrationIndexArgsForCall, struct{}{})
	stub := fake.GetLatestMigrationIndexStub
	fakeReturns := fake.getLatestMigrationIndexReturns
	fake.recordInvocation("GetLatestMigrationIndex", []interface{}{})
	fake.getLatestMigrationIndexMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeMigrationProvider) GetLatestMigrationIndexCallCount() int {
	fake.getLatestMigrationIndexMutex.RLock()
	defer fake.getLatestMigrationIndexMutex.RUnlock()
	return len(fake.getLatestMigrationIndexArgsForCall)
}

func (fake *FakeMigrationProvider) GetLatestMigrationIndexCalls(stub func() int) {
	fake.getLatestMigrationIndexMutex.Lock()
	defer fake.getLatestMigrationIndexMutex.Unlock()
	fake.GetLatestMigrationIndexStub = stub
}

func (fake *FakeMigrationProvider) GetLatestMigrationIndexReturns(result1 int) {
	fake.getLatestMigrationIndexMutex.Lock()
	defer fake.getLatestMigrationIndexMutex.Unlock()
	fake.GetLatestMigrationIndexStub = nil
	fake.getLatestMigrationIndexReturns = struct {
		result1 int
	}{result1}
}

func (fake *FakeMigrationProvider) GetLatestMigrationIndexReturnsOnCall(i int, result1 int) {
	fake.getLatestMigrationIndexMutex.Lock()
	defer fake.getLatestMigrationIndexMutex.Unlock()
	fake.GetLatestMigrationIndexStub = nil
	if fake.getLatestMigrationIndexReturnsOnCall == nil {
		fake.getLatestMigrationIndexReturnsOnCall = make(map[int]struct {
			result1 int
		})
	}
	fake.getLatestMigrationIndexReturnsOnCall[i] = struct {
		result1 int
	}{result1}
}

func (fake *FakeMigrationProvider) Provide() []migrations.MigrationStep {
	fake.provideMutex.Lock()
	ret, specificReturn := fake.provideReturnsOnCall[len(fake.provideArgsForCall)]
	fake.provideArgsForCall = append(fake.provideArgsForCall, struct{}{})
	stub := fake.ProvideStub
	fakeReturns := fake.provideReturns
	fake.recordInvocation("Provide", []interface{}{})
	fake.provideMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeMigrationProvider) ProvideCallCount() int {
	fake.provideMutex.RLock()
	defer fake.provideMutex.RUnlock()
	return len(fake.provideArgsForCall)
}

func (fake *FakeMigrationProvider) ProvideCalls(stub func() []migrations.MigrationStep) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = stub
}

func (fake *FakeMigrationProvider) ProvideReturns(result1 []migrations.MigrationStep) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = nil
	fake.provideReturns = struct {
		result1 []migrations.MigrationStep
	}{result1}
}

func (fake *FakeMigrationProvider) ProvideReturnsOnCall(i int, result1 []migrations.MigrationStep) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = nil
	if fake.provideReturnsOnCall == nil {
		fake.provideReturnsOnCall = make(map[int]struct {
			result1 []migrations.MigrationStep
		})
	}
	fake.provideReturnsOnCall[i] = struct {
		result1 []migrations.MigrationStep
	}{result1}
}

func (fake *FakeMigrationProvider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getLatestMigrationIndexMutex.RLock()
	defer fake.getLatestMigrationIndexMutex.RUnlock()
	fake.provideMutex.RLock()
	defer fake.provideMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeMigrationProvider) recordInvocation(key string, args []interface{}) {
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

var _ migrations.MigrationProvider = new(FakeMigrationProvider)

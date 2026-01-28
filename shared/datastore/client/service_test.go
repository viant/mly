package client

import (
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/viant/mly/shared/circut"
	"github.com/viant/mly/shared/config/datastore"
)

type MockAero struct {
	counter uint64

	L *sync.RWMutex
}

func (m *MockAero) Get(policy *aero.BasePolicy, key *aero.Key, binNames ...string) (record *aero.Record, err error) {
	return nil, nil
}

func (m *MockAero) Put(writePolicy *aero.WritePolicy, key *aero.Key, value aero.BinMap) (err error) {
	log.Printf("wait lock %d", m.counter)
	m.L.RLock()
	defer m.L.RUnlock()
	log.Printf("put %d", m.counter)
	atomic.AddUint64(&m.counter, 1)
	return nil
}

func TestPut(t *testing.T) {
	mockLock := new(sync.RWMutex)
	mockAero := &MockAero{
		L: mockLock,
	}

	config := &datastore.Connection{
		ID: "test",
	}
	config.Init()

	service, _ := NewWithOptionsV2(config, nil)
	service.Client = mockAero

	breaker := circut.New(time.Second, service)
	service.Breaker = breaker

	key, _ := aero.NewKey("test", "test", "test")
	policy := aero.NewWritePolicy(0, 0)

	var finishGroup, startGroup sync.WaitGroup

	totalPuts := 20
	finishGroup.Add(totalPuts)
	startGroup.Add(totalPuts)

	// prevent put from completing
	mockLock.Lock()

	for i := 0; i < totalPuts; i++ {
		go func() {
			defer finishGroup.Done()
			log.Printf("start %d", i)
			startGroup.Done()
			err := service.Put(policy, key, aero.BinMap{"test": "test"})
			log.Printf("done %d, err: %v", i, err)
		}()
	}

	startGroup.Wait()
	mockLock.Unlock()
	finishGroup.Wait()

	if mockAero.counter != 1 {
		t.Fatalf("expected 1 put, got %d", mockAero.counter)
	}

	// run without any blocking
	finishGroup.Add(totalPuts)
	for i := 0; i < totalPuts; i++ {
		go func() {
			defer finishGroup.Done()
			err := service.Put(policy, key, aero.BinMap{"test": "test"})
			log.Printf("done %d, err: %v", i, err)
		}()
	}
	finishGroup.Wait()

	log.Printf("put total counter: %d (max %d)", mockAero.counter, totalPuts)
}

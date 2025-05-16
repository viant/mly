package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/aerospike/aerospike-client-go"
	"github.com/stretchr/testify/assert"

	"github.com/viant/mly/shared/circut"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/datastore/client"
)

// stubAeroRecord returns a preset Aerospike record.
type stubAeroRecord struct {
	record *aerospike.Record
}

func (s stubAeroRecord) Get(policy *aerospike.BasePolicy, key *aerospike.Key, binNames ...string) (*aerospike.Record, error) {
	return s.record, nil
}

func (s stubAeroRecord) Put(writePolicy *aerospike.WritePolicy, key *aerospike.Key, value aerospike.BinMap) error {
	panic("unexpected Put call")
}

type stubProber struct {
	ProbeCount int
}

func (s *stubProber) Probe() {
	s.ProbeCount++
}

// TestFromClientMapsRecordAndDoesNotMapHashBin verifies that fromClient maps bins into the struct
// and skips the HashBin key for field mapping, while returning the dictHash.
func TestFromClientMapsRecordAndDoesNotMapHashBin(t *testing.T) {
	svc := &Service{}
	rec := &aerospike.Record{Bins: map[string]interface{}{
		"Field":        "value",
		common.HashBin: 123,
	}}
	clientSvc := &client.Service{
		Client:  stubAeroRecord{record: rec},
		Breaker: circut.New(time.Second*10, &stubProber{}),
	}

	key := &Key{Namespace: "ns", Set: "set", Value: "key"}
	type Foo struct {
		Field   string
		HashVal int
	}
	foo := &Foo{Field: "", HashVal: -1}
	dictHash, err := svc.fromClient(context.Background(), clientSvc, key, foo)
	assert.Nil(t, err)
	// fromClient should return the HashBin value
	assert.EqualValues(t, 123, dictHash)
	// Field should be mapped
	assert.EqualValues(t, "value", foo.Field)
	// HashVal should remain unchanged because generic.Set skips HashBin mapping
	assert.EqualValues(t, -1, foo.HashVal)
}

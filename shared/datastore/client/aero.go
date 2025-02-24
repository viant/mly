package client

import (
	aero "github.com/aerospike/aerospike-client-go"
)

// Aero is a mock of aerospike.Client.
type Aero interface {
	// Mocks aerospike.Client.Get
	Get(policy *aero.BasePolicy, key *aero.Key, binNames ...string) (record *aero.Record, err error)

	// Mocks aerospike.Client.Put
	Put(writePolicy *aero.WritePolicy, key *aero.Key, value aero.BinMap) (err error)
}

// keyString avoids using Key.String() to avoid inner usages of fmt.Sprintf
/*
goos: darwin
goarch: arm64
pkg: github.com/viant/mly/shared/datastore/client
cpu: Apple M3 Max
BenchmarkKeyString-16           32234271                31.74 ns/op
BenchmarkKeyStringer-16           986280              1229 ns/op
*/
func keyString(key *aero.Key) string {
	if key.Value() == nil {
		return key.Namespace() + "." + key.SetName()
	}

	// key.Value().String() may contain inner usages of fmt.Sprintf
	return key.Namespace() + "." + key.SetName() + "." + key.Value().String()
}

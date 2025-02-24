package client

import (
	"testing"

	aero "github.com/aerospike/aerospike-client-go"
)

func BenchmarkKeyString(b *testing.B) {
	key, _ := aero.NewKey("test", "test", "test")
	strs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		strs[i] = keyString(key)
	}
}

func BenchmarkKeyStringer(b *testing.B) {
	key, _ := aero.NewKey("test", "test", "test")
	strs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		strs[i] = key.String()
	}
}

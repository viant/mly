package datastore

import (
	"context"

	"github.com/viant/mly/shared/config"
)

type Storer interface {
	// Put writes the value to a set of backplanes.
	// dictHash is provided to invalidate caches in case the same input should result in a different value; kind of like an etag/version.
	Put(ctx context.Context, key *Key, value Value, dictHash int) error

	// GetInto will attempt to pull a value from backplanes.
	// If a further cache has a value but a closer one does not, this should populate the closer cache.
	// Should return the dictHash (etag/version).
	GetInto(ctx context.Context, key *Key, storable Value) (dictHash int, err error)

	// TODO refactor out implementation Aerospike specific values.
	Key(key string) *Key

	// TODO remove / rethink implementation.
	Mode() StoreMode
	SetMode(mode StoreMode)

	Enabled() bool

	Config() *config.Datastore
}

// StoreMode represents service mode.
// TODO: change implementation and split the code appropriately for handling server and client side.
type StoreMode int

const (
	ModeServer = StoreMode(0)
	ModeClient = StoreMode(1)
)

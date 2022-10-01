package datastore

import (
	"context"
	"github.com/viant/mly/shared/config"
)

type Storer interface {
	Put(ctx context.Context, key *Key, value Value, dictHash int) error
	GetInto(ctx context.Context, key *Key, storable Value) (dictHash int, err error)
	Key(key string) *Key
	Enabled() bool
	Mode() StoreMode
	SetMode(mode StoreMode)
	Config() *config.Datastore
}

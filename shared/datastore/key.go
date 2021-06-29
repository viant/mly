package datastore

import (
	"fmt"
	aero "github.com/aerospike/aerospike-client-go"
	"github.com/viant/mly/shared/config"
	"strconv"
	"time"
)

//Key represents a datastore key
type Key struct {
	Namespace string
	Set       string
	Value     interface{}
	*aero.GenerationPolicy
	TimeToLive time.Duration
	L2         *Key
}

func (k *Key) AsString() string {
	switch value := k.Value.(type) {
	case string:
		return value
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.Itoa(int(value))
	default:
		return fmt.Sprintf("%v", value)
	}
}

func (k *Key) Key() (*aero.Key, error) {
	return aero.NewKey(k.Namespace, k.Set, k.Value)
}

func (k *Key) WritePolicy(generation uint32) *aero.WritePolicy {
	policy := aero.NewWritePolicy(0, 0)
	if k.TimeToLive == 0 {
		k.TimeToLive = time.Hour
	}
	policy.Expiration = uint32(k.TimeToLive / time.Second)
	policy.Generation = generation
	if k.GenerationPolicy != nil {
		policy.GenerationPolicy = *k.GenerationPolicy
	}
	return policy
}

func NewKey(cfg *config.Datastore, key string) *Key {
	storeKey := &Key{
		Namespace:  cfg.Namespace,
		Set:        cfg.Dataset,
		Value:      key,
		TimeToLive: cfg.TimeToLive(),
	}
	if cfg.L2 != nil {
		storeKey.L2 = &Key{
			Namespace:  cfg.L2.Namespace,
			Set:        cfg.L2.Dataset,
			Value:      key,
			TimeToLive: cfg.L2.TimeToLive(),
		}
	}
	return storeKey
}

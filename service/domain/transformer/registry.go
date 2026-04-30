package transformer

import (
	"fmt"

	"github.com/viant/mly/service/domain"
)

// Register registers an output transformer.
func Register(key string, transformer domain.Transformer) {
	Singleton().Register(key, transformer)
}

// Registry represents a registry.
// Registry is not concurrency safe.
type Registry struct {
	registry map[string]domain.Transformer
}

func (r *Registry) Register(key string, transformer domain.Transformer) {
	r.registry[key] = transformer
}

// Lookup returns a transformer by key or error.
func (r *Registry) Lookup(key string) (domain.Transformer, error) {
	transformer, ok := r.registry[key]
	if !ok {
		return nil, fmt.Errorf("failed to lookup transformer: %v", key)
	}
	return transformer, nil
}

var registry = &Registry{
	registry: make(map[string]domain.Transformer),
}

// Singleton return transformer registry
func Singleton() *Registry {
	return registry
}

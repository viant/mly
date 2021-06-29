package transformer

import (
	"fmt"
	"github.com/viant/mly/service/domain"
)

type Registry struct {
	registry map[string]domain.Transformer
}

func (r *Registry) Register(key string, transformer domain.Transformer) {
	r.registry[key] = transformer
}

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

//Singleton return transformer registry
func Singleton() *Registry {
	return registry
}

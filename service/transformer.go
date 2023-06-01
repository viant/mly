package service

import (
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/domain/transformer"
)

func getTransformer(name string, signature *domain.Signature) (domain.Transformer, error) {
	result, err := transformer.Singleton().Lookup(name)
	if err == nil && result != nil {
		return result, nil
	}

	if name != "" {
		return nil, err
	}

	// otherwise return default transformer
	return domain.Transform, nil
}

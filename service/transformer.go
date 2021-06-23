package service

import (
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/domain/transformer"
)

func getTransformer(name string, signature *domain.Signature) domain.Transformer {
	result, err := transformer.Singleton().Lookup(name)
	if err == nil && result != nil {
		return result
	} //otherwise return default transformer
	return domain.Transform
}

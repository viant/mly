package test

import (
	"context"
	"fmt"

	"github.com/francoispqt/gojay"
	"github.com/viant/bintly"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/toolbox"
)

type VO struct {
	Value float32
}

func (p *VO) EncodeBinary(enc *bintly.Writer) error {
	enc.Float32(p.Value)
	return nil
}

func (p *VO) DecodeBinary(dec *bintly.Reader) error {
	dec.Float32(&p.Value)
	return nil
}

func (s *VO) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Float32Key("value", s.Value)
}

func (s *VO) IsNil() bool {
	return s == nil
}

func (p *VO) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		if err := pair("value", p.Value); err != nil {
			return err
		}
		return nil
	}
}

func (p *VO) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		switch key {
		case "value":
			p.Value = float32(toolbox.AsFloat(value))
		}
		return nil
	})
}

func VOTransformer(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &VO{}
	switch actual := output.(type) {
	case []interface{}:
		return FloatTransformer(ctx, signature, input, actual[0])
	case [][]float32:
		result.Value = actual[0][0]
	case *shared.Output:
		switch val := actual.Values[0].(type) {
		case [][]float32:
			result.Value = val[actual.InputIndex][0]
		}
	default:
		return nil, fmt.Errorf("unsupported type: %T", actual)
	}
	return result, nil
}

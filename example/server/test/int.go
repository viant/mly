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

type IntOutput struct {
	Value int64
	Sl    string
}

func (p *IntOutput) EncodeBinary(enc *bintly.Writer) error {
	enc.Int64(p.Value)
	enc.String(p.Sl)
	return nil
}

func (p *IntOutput) DecodeBinary(dec *bintly.Reader) error {
	dec.Int64(&p.Value)
	dec.String(&p.Sl)
	return nil
}

func (s *IntOutput) MarshalJSONObject(enc *gojay.Encoder) {
	enc.IntKey("value", int(s.Value))
	enc.StringKey("sl", s.Sl)
}

func (s *IntOutput) IsNil() bool {
	return s == nil
}

func (p *IntOutput) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		if err := pair("value", p.Value); err != nil {
			return err
		}
		if err := pair("sl", p.Sl); err != nil {
			return err
		}
		return nil
	}
}

func (p *IntOutput) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		switch key {
		case "value":
			p.Value = int64(toolbox.AsInt(value))
		case "sl":
			p.Sl = toolbox.AsString(value)
		}
		return nil
	})
}

func IntTransformer(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &IntOutput{}
	result.Sl = toolbox.AsString(input.Value("sa"))
	switch actual := output.(type) {
	case []interface{}:
		return IntTransformer(ctx, signature, input, actual[0])
	case [][]int64:
		result.Value = actual[0][0]
	case *shared.Output:
		switch val := actual.Values[0].(type) {
		case [][]int64:
			result.Value = val[actual.InputIndex][0]
		}
	default:
		return nil, fmt.Errorf("unsupported type: %T", actual)
	}
	return result, nil
}

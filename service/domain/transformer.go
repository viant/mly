package domain

import (
	"context"
	"fmt"

	"github.com/viant/gtly"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
)

// Transformer represents output transformer
type Transformer func(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error)

// Transform default Transformer
func Transform(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	fields := []*storable.Field{}
	for _, output := range signature.Outputs {
		fields = append(fields, &storable.Field{Name: output.Name, DataType: output.DataType})
	}
	var pairs = []*kvPair{}
	result := storable.New(fields)
	var outputValue interface{}
	switch val := output.(type) {
	case [][]float32:
		outputValue = val[0][0]
		pairs = append(pairs, &kvPair{
			k: signature.Outputs[0].Name,
			v: outputValue,
		})
	case [][]float64:
		outputValue = val[0][0]
		pairs = append(pairs, &kvPair{
			k: signature.Outputs[0].Name,
			v: outputValue,
		})
	case [][]string:
		outputValue = val[0][0]
		pairs = append(pairs, &kvPair{
			k: signature.Outputs[0].Name,
			v: outputValue,
		})
	case [][]int64:
		outputValue = val[0][0]
		pairs = append(pairs, &kvPair{
			k: signature.Outputs[0].Name,
			v: outputValue,
		})
	case []interface{}:
		for i := range val {
			tensor := val[i]
			switch t := tensor.(type) {
			case [][]float32:
				outputValue = t[0][0]
			case [][]float64:
				outputValue = t[0][0]
			case [][]string:
				outputValue = t[0][0]
			case [][]int64:
				outputValue = t[0][0]
			default:
				return nil, fmt.Errorf("unsupported output sub-type: %v %T", signature.Outputs[0].Name, t)
			}
			pairs = append(pairs, &kvPair{
				k: signature.Outputs[i].Name,
				v: outputValue,
			})

		}
	default:
		return nil, fmt.Errorf("unsupported output type: %v, %T", signature.Outputs[0].Name, output)
	}

	err := result.Set(func(pair common.Pair) error {
		for _, kvPair := range pairs {
			if err := pair(kvPair.k, kvPair.v); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

type kvPair struct {
	k string
	v interface{}
}

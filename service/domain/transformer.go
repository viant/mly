package domain

import (
	"context"
	"fmt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/gtly"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
)

//Transformer represents output transformer
type Transformer func(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error)

//Transform transform default model output
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
	case []*tf.Tensor:
		for i := range val {
			tensor := val[i].Value()
			switch t := tensor.(type) {
			case []float32:
				outputValue = t[0]
			case []float64:
				outputValue = t[0]
			case []string:
				outputValue = t[0]
			case []int64:
				outputValue = t[0]
			default:
				return nil, fmt.Errorf("unsupported type: %T", t)
			}
			pairs = append(pairs, &kvPair{
				k: signature.Outputs[i].Name,
				v: outputValue,
			})

		}
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

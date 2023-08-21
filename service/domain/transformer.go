package domain

import (
	"context"
	"fmt"

	"github.com/viant/gtly"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
)

// Transformer is an adapter module used when the output of the TensorFlow model wants to be modified server side.
// signature is the Signature of the relevant model, determined by request context.
// input is the request body object as a *gtly.Object
// output is the TensorFlow SavedModel prediction output.
type Transformer func(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error)

func appendPair(outputs []Output, v []interface{}, ii int, pairs []*kvPair) ([]*kvPair, error) {
	var outputValue interface{}

	for i, tensor := range v {
		switch t := tensor.(type) {
		case [][]string:
			outputValue = t[ii][0]
		case [][]float32:
			outputValue = t[ii][0]
		case [][]float64:
			outputValue = t[ii][0]
		case [][]int64:
			outputValue = t[ii][0]
		case [][]int32:
			outputValue = t[ii][0]
		case []string:
			outputValue = t[ii]
		case []float32:
			outputValue = t[ii]
		case []float64:
			outputValue = t[ii]
		case []int64:
			outputValue = t[ii]
		case []int32:
			outputValue = t[ii]
		default:
			return nil, fmt.Errorf("unsupported default transform sub-type: %v %T", outputs[i].Name, t)
		}

		pairs = append(pairs, &kvPair{
			k: outputs[i].Name,
			v: outputValue,
		})
	}

	return pairs, nil
}

// The default Transformer will create an object that when marshalled will return as close to the the TensorFlow
// SavedModel signature as possible; with the caveat that any output will be keyed by the output tensor's name.
func Transform(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	fields := []*storable.Field{}
	outputs := signature.Outputs
	for _, output := range outputs {
		fields = append(fields, &storable.Field{Name: output.Name, DataType: output.DataType})
	}

	var pairs = []*kvPair{}
	result := storable.New(fields)

	var err error

	switch val := output.(type) {
	case *shared.Output:
		pairs, err = appendPair(outputs, val.Values, val.InputIndex, pairs)
		if err != nil {
			return nil, err
		}
	case []interface{}:
		pairs, err = appendPair(outputs, val, 0, pairs)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported default transform type: %v, %T", outputs[0].Name, output)
	}

	err = result.Set(func(pair common.Pair) error {
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

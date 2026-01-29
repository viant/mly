package router

import (
	"context"
	"fmt"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/request/shape"
)

// preparedReplacement holds a pre-parsed replacement value for fixed evaluator outputs
type preparedReplacement struct {
	name  string
	typ   string
	value interface{}
}

type fixedEvaluator struct {
	prepared []preparedReplacement
}

// OutputNames returns the output names in the order they will be returned by Predict
func (f *fixedEvaluator) OutputNames() []string {
	names := make([]string, len(f.prepared))
	for i, p := range f.prepared {
		names[i] = p.name
	}
	return names
}

func newFixedEvaluator(repls []config.PredictionReplacement) (*fixedEvaluator, error) {
	prepared := make([]preparedReplacement, 0, len(repls))

	err := func() error {
		for _, r := range repls {

			var pr preparedReplacement
			switch r.Type {
			case "string":
				v, ok := r.Value.(string)
				if !ok {
					v = fmt.Sprintf("%v", r.Value)
				}
				pr = preparedReplacement{typ: "string", value: v}
			case "int":
				switch n := r.Value.(type) {
				case int:
					pr = preparedReplacement{typ: "int", value: n}
				case int32:
					pr = preparedReplacement{typ: "int", value: int(n)}
				case int64:
					pr = preparedReplacement{typ: "int", value: int(n)}
				case float32:
					pr = preparedReplacement{typ: "int", value: int(n)}
				case float64:
					pr = preparedReplacement{typ: "int", value: int(n)}
				default:
					return fmt.Errorf("router replacement %q: value %T not coercible to int", r.Name, r.Value)
				}
			case "int32":
				switch n := r.Value.(type) {
				case int:
					pr = preparedReplacement{typ: "int32", value: int32(n)}
				case int32:
					pr = preparedReplacement{typ: "int32", value: n}
				case int64:
					pr = preparedReplacement{typ: "int32", value: int32(n)}
				case float32:
					pr = preparedReplacement{typ: "int32", value: int32(n)}
				case float64:
					pr = preparedReplacement{typ: "int32", value: int32(n)}
				default:
					return fmt.Errorf("router replacement %q: value %T not coercible to int32", r.Name, r.Value)
				}
			case "int64":
				switch n := r.Value.(type) {
				case int:
					pr = preparedReplacement{typ: "int64", value: int64(n)}
				case int32:
					pr = preparedReplacement{typ: "int64", value: int64(n)}
				case int64:
					pr = preparedReplacement{typ: "int64", value: n}
				case float32:
					pr = preparedReplacement{typ: "int64", value: int64(n)}
				case float64:
					pr = preparedReplacement{typ: "int64", value: int64(n)}
				default:
					return fmt.Errorf("router replacement %q: value %T not coercible to int64", r.Name, r.Value)
				}
			case "float32":
				switch n := r.Value.(type) {
				case int:
					pr = preparedReplacement{typ: "float32", value: float32(n)}
				case int32:
					pr = preparedReplacement{typ: "float32", value: float32(n)}
				case int64:
					pr = preparedReplacement{typ: "float32", value: float32(n)}
				case float32:
					pr = preparedReplacement{typ: "float32", value: n}
				case float64:
					pr = preparedReplacement{typ: "float32", value: float32(n)}
				default:
					return fmt.Errorf("router replacement %q: value %T not coercible to float32", r.Name, r.Value)
				}
			case "float64":
				switch n := r.Value.(type) {
				case int:
					pr = preparedReplacement{typ: "float64", value: float64(n)}
				case int32:
					pr = preparedReplacement{typ: "float64", value: float64(n)}
				case int64:
					pr = preparedReplacement{typ: "float64", value: float64(n)}
				case float32:
					pr = preparedReplacement{typ: "float64", value: float64(n)}
				case float64:
					pr = preparedReplacement{typ: "float64", value: n}
				default:
					return fmt.Errorf("router replacement %q: value %T not coercible to float64", r.Name, r.Value)
				}
			default:
				return fmt.Errorf("unsupported router replacement type %q for %q", r.Type, r.Name)
			}

			pr.name = r.Name
			prepared = append(prepared, pr)
		}

		return nil
	}()
	if err != nil {
		return nil, err
	}

	return &fixedEvaluator{prepared: prepared}, nil
}

func (f *fixedEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	batchSize, err := shape.DetermineBatchSize(params)
	if err != nil {
		return nil, err
	}

	makeString := func(v string) [][]string {
		out := make([][]string, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []string{v}
		}
		return out
	}

	makeInt32 := func(v int32) [][]int32 {
		out := make([][]int32, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []int32{v}
		}
		return out
	}

	makeInt64 := func(v int64) [][]int64 {
		out := make([][]int64, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []int64{v}
		}
		return out
	}

	makeInt := func(v int) [][]int {
		out := make([][]int, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []int{v}
		}
		return out
	}

	makeFloat32 := func(v float32) [][]float32 {
		out := make([][]float32, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []float32{v}
		}
		return out
	}

	makeFloat64 := func(v float64) [][]float64 {
		out := make([][]float64, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []float64{v}
		}
		return out
	}

	results := make([]interface{}, len(f.prepared))
	for i, repl := range f.prepared {
		switch repl.typ {
		case "string":
			results[i] = makeString(repl.value.(string))
		case "int":
			results[i] = makeInt(repl.value.(int))
		case "int32":
			results[i] = makeInt32(repl.value.(int32))
		case "int64":
			results[i] = makeInt64(repl.value.(int64))
		case "float32":
			results[i] = makeFloat32(repl.value.(float32))
		case "float64":
			results[i] = makeFloat64(repl.value.(float64))
		default:
			return nil, fmt.Errorf("unsupported replacement type %q", repl.typ)
		}
	}

	return results, nil
}

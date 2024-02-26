package evaluator

import "context"

type Evaluator interface {
	// Evaluate should take the unstructured input params tensor and
	// produce an expected output tensor.
	Evaluate(ctx context.Context, params []interface{}) ([]interface{}, error)

	Close() error
}

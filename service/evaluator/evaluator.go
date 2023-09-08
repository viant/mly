package evaluator

import "context"

type Evaluator interface {
	// Evaluate should take the unstructured input params and produce an expected
	// output.
	Evaluate(ctx context.Context, params []interface{}) ([]interface{}, error)

	Close() error
}

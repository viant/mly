package tfmodel

import "context"

type Runner interface {
	Evaluate(context.Context, []interface{}) ([]interface{}, error)
}

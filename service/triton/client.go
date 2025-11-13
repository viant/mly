package triton

import (
	"context"
)

type TritonClient interface {
	ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) ([]interface{}, error)

	ModelReady(ctx context.Context, modelName string) (bool, error)

	ModelLoad(ctx context.Context, modelName string) error

	Close() error
}

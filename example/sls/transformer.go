package sls

import (
	"context"
	"fmt"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
	"sync"
)

var slFieldAccessor *gtly.Accessor
var xFieldAccessor *gtly.Accessor

var once = sync.Once{}

func Transform(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &Record{}
	once.Do(func() {
		slFieldAccessor = input.Proto().Accessor("sl")
		xFieldAccessor = input.Proto().Accessor("x")
	})
	result.Sl = slFieldAccessor.String(input)
	result.X = xFieldAccessor.String(input)
	switch actual := output.(type) {
	case []interface{}:
		switch value := actual[0].(type) {
		case []int64:
			result.Value = value[0]
		default:
			return nil, fmt.Errorf("unsupproted tensor value type: %T", value)
		}
	default:
		return nil, fmt.Errorf("unsupproted output type: %T", actual)
	}
	return result, nil
}

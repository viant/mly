package slf

import (
	"context"
	"fmt"

	"github.com/viant/gtly"
	"github.com/viant/mly/example/transformer/slf/model"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

// conforms to service/domain.Transformer
// only supports single-item requests
func Transform(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	actual, err := extract(output, 0)
	if err != nil {
		return nil, err
	}

	segment := "other"
	if actual < 1 {
		segment = "one"
	} else if actual < 2 {
		segment = "two"
	} else if actual < 5 {
		segment = "five"
	}

	s := new(model.Segmented)
	s.Class = segment
	return s, nil
}

func extract(o interface{}, i int) (float32, error) {
	switch typed := o.(type) {
	case *shared.Output:
		return extract(typed.Values[0], typed.InputIndex)
	case []interface{}:
		return extract(typed[0], 0)
	case [][]float32:
		if len(typed) > i && len(typed[0]) > 0 {
			return typed[i][0], nil
		} else {
			return -1, fmt.Errorf("improper result i:%d actual:%v", i, typed)
		}
	default:
		return -1, fmt.Errorf("expected [][]float32, but had: %T", o)
	}
}

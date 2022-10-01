package sls

import (
	"context"
	"fmt"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
	"github.com/viant/toolbox"
)

func Transform(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &Record{}
	result.Sl = toolbox.AsString(input.Value("sl"))
	result.X = toolbox.AsString(input.Value("x"))
	switch actual := output.(type) {
	case []int64:
		result.Value = actual[0]
	default:
		return nil, fmt.Errorf("unsupproted type: %T", actual)
	}
	return result, nil
}

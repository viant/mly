package vec

import (
	"context"
	"fmt"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/toolbox"
)

func Transform(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &Record{}
	result.Sa = toolbox.AsString(input.Value("sa"))
	switch actual := output.(type) {
	case *shared.Output:
		switch val := actual.Values[0].(type) {
		case []int64:
			result.Value = val[actual.InputIndex]
		}
	default:
		return nil, fmt.Errorf("unsupproted type: %T", actual)
	}
	return result, nil
}

package stat

import (
	"testing"

	"github.com/viant/mly/shared/stat"
)

func TestEval(t *testing.T) {
	stat.TestMapping(t, NewEval)
}

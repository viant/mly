package stat

import (
	"testing"

	"github.com/viant/mly/shared/stat"
)

func TestService(t *testing.T) {
	stat.TestMapping(t, NewProvider)
}

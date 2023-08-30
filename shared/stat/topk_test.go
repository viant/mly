package stat

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopK(t *testing.T) {
	k := NewTopK(10, 0)

	k.Aggregate(ErrorWrap(fmt.Errorf("test")))

	found := false
	for _, ie := range k.Top {
		if ie.Count > 0 {
			found = true
			break
		}
	}

	assert.True(t, found)
}

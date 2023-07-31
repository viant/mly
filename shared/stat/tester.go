package stat

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/gmetric/counter"
)

// TestMapping checks to ensure that a standard 1-to-1 mapping between keys and values exists.
func TestMapping(t *testing.T, builder func() counter.Provider) {
	p := builder()
	keys := p.Keys()
	for _, key := range keys {
		if key == ErrorKey {
			continue
		}

		i := p.Map(key)
		assert.NotEqual(t, -1, i)
	}

	assert.Equal(t, 0, p.Map(fmt.Errorf("")))
}

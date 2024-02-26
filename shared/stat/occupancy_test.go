package stat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/viant/gmetric"
	"github.com/viant/gmetric/counter"
)

type Ocv struct{}

func (_ Ocv) Keys() []string {
	return []string{"pending"}
}

func (_ Ocv) NewCounter() counter.CustomCounter {
	return new(Occupancy)
}

func (_ Ocv) Map(_ interface{}) int {
	return 0
}

func TestMetric(t *testing.T) {
	gms := gmetric.New()
	c := gms.MultiOperationCounter("test", "test", "test", time.Microsecond, time.Second, 2, Ocv{})

	c.IncrementValue(Enter)
	c.IncrementValue(Enter)
	c.IncrementValue(Enter)

	oc, ok := c.Operation.Operation.MultiCounter.Counters[0].Custom.(*Occupancy)
	assert.True(t, ok)
	assert.Equal(t, oc.Max, uint64(3))

	c.DecrementValue(Exit)
	c.DecrementValue(Exit)
	c.DecrementValue(Exit)

	assert.Equal(t, oc.Max, uint64(3))
}

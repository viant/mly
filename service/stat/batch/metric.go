package batch

import (
	"sync"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/gmetric/counter"
)

type BatchMetrics struct {
	mtx     sync.Mutex
	metrics map[int]*gmetric.Operation
	builder Builder
}

var NoOp = func(time.Time, ...interface{}) int64 {
	return 0
}

// Builder is a function that is used to define the properties of the metric.
type Builder func(bs int) *gmetric.Operation

// NewBatchMetrics will do nothing if builder is nil.
func NewBatchMetrics(builder Builder) *BatchMetrics {
	return &BatchMetrics{
		metrics: make(map[int]*gmetric.Operation),
		builder: builder,
	}
}

func (bm *BatchMetrics) Begin(batchSize int) counter.OnDone {
	if bm.builder == nil {
		return NoOp
	}

	metric, ok := bm.metrics[batchSize]
	if !ok {
		bm.mtx.Lock()
		defer bm.mtx.Unlock()

		metric, ok = bm.metrics[batchSize]
		if !ok {
			metric = bm.builder(batchSize)
			bm.metrics[batchSize] = metric
		}
	}

	if metric == nil {
		return NoOp
	}

	return metric.Begin(time.Now())
}

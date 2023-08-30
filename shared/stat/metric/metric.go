package metric

import (
	"time"

	"github.com/viant/gmetric"
)

func EnterThenExit(metric *gmetric.Operation, start time.Time, enterValue interface{}, exitValue interface{}) func() {
	metric.IncrementValue(enterValue)

	index := metric.Index(start)
	recentCounter := metric.Recent[index]
	recentCounter.IncrementValue(enterValue)

	return func() {
		metric.DecrementValue(exitValue)
		recentCounter := metric.Recent[index]
		recentCounter.DecrementValue(exitValue)
	}
}

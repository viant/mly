package promc

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

type BaseErrorCounters struct {
	DeadlineExceededCounter prometheus.Counter
	CanceledCounter         prometheus.Counter

	OtherErrorCounter prometheus.Counter
}

func (c BaseErrorCounters) Observe(err error) {
	if c.DeadlineExceededCounter != nil && errors.Is(err, context.DeadlineExceeded) {
		c.DeadlineExceededCounter.Inc()
	} else if c.CanceledCounter != nil && errors.Is(err, context.Canceled) {
		c.CanceledCounter.Inc()
	} else if c.OtherErrorCounter != nil {
		c.OtherErrorCounter.Inc()
	}
}

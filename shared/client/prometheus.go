package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/mly/shared/stat/buckets"
	"github.com/viant/mly/shared/stat/promc"
)

const (
	promDescRunDuration        = "Duration of client Run calls."
	promDescHTTPDuration       = "Duration of client HTTP calls, including retries."
	promDescHTTPClientDuration = "Duration of client HTTP client calls."
	promDescBatchSize          = "Size of client batches."
)

var (
	// EarlyCtxError
	// loadFromCache error - this can only be a type error from Response.DataItemType(), (*Service).readFromCache()

	runErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mly",
			Subsystem: "client",
			Name:      "prediction_error_counter",
			Help:      "Number of client and kind of prediction errors.",
		},
		[]string{"model", "error"},
	)

	httpErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mly",
			Subsystem: "client",
			Name:      "http_error_counter",
			Help:      "Number of client HTTP errors.",
		},
		[]string{"model", "error"},
	)

	httpClientErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mly",
			Subsystem: "client",
			Name:      "http_client_error_counter",
			Help:      "Number of client HTTP client errors.",
		},
		[]string{"model", "error"},
	)
)

type prometheusMetrics struct {
	runDurationHistogram        prometheus.Observer
	batchSizeHistogram          prometheus.Observer
	httpDurationHistogram       prometheus.Observer
	httpClientDurationHistogram prometheus.Observer

	// The Summary metrics may be nil if noPrometheusSummaries is true.

	runDurationSummary        prometheus.Observer
	batchSizeSummary          prometheus.Observer
	httpDurationSummary       prometheus.Observer
	httpClientDurationSummary prometheus.Observer

	runErrorEarlyCtxCounter prometheus.Counter
	runBaseErrorCounters    promc.BaseErrorCounters

	httpDownCounter       prometheus.Counter
	httpBaseErrorCounters promc.BaseErrorCounters

	httpClientBaseErrorCounters promc.BaseErrorCounters
}

func (m prometheusMetrics) observeRunDuration(duration float64) {
	if m.runDurationHistogram != nil {
		m.runDurationHistogram.Observe(duration)
	}
	if m.runDurationSummary != nil {
		m.runDurationSummary.Observe(duration)
	}
}

func (m prometheusMetrics) observeBatchSize(batchSize float64) {
	if m.batchSizeHistogram != nil {
		m.batchSizeHistogram.Observe(batchSize)
	}
	if m.batchSizeSummary != nil {
		m.batchSizeSummary.Observe(batchSize)
	}
}

func (m prometheusMetrics) observeHttpDuration(duration float64) {
	if m.httpDurationHistogram != nil {
		m.httpDurationHistogram.Observe(duration)
	}
	if m.httpDurationSummary != nil {
		m.httpDurationSummary.Observe(duration)
	}
}

func (m prometheusMetrics) observeHttpClientDuration(duration float64) {
	if m.httpClientDurationHistogram != nil {
		m.httpClientDurationHistogram.Observe(duration)
	}
	if m.httpClientDurationSummary != nil {
		m.httpClientDurationSummary.Observe(duration)
	}
}

// Used strictly to test for error type.
var are prometheus.AlreadyRegisteredError

func isPrometheusAlreadyRegisteredError(err error) bool {
	if err == nil {
		return false
	}

	return errors.As(err, &are)
}

func (m *prometheusMetrics) registerPrometheusMetrics(registerer prometheus.Registerer, model string, noPrometheusSummaries bool) error {
	// convenience function
	register := func(metric prometheus.Collector) error {
		err := registerer.Register(metric)
		if err != nil && !isPrometheusAlreadyRegisteredError(err) {

			dc := make(chan *prometheus.Desc)
			go func() {
				metric.Describe(dc)
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			var metricName string
			select {
			case desc := <-dc:
				metricName = desc.String()
			case <-ctx.Done():
				metricName = "unknown"
			}

			return fmt.Errorf("failed to register %s: %T, %w", metricName, err, err)
		}

		return nil
	}

	var err error
	if !noPrometheusSummaries {
		runDurationSummaryMicros := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  "mly",
				Subsystem:  "client",
				Name:       "run_duration_summary_us",
				Help:       promDescRunDuration,
				Objectives: buckets.CommonSummaryObjectives,
			},
			[]string{"model"},
		)

		err = register(runDurationSummaryMicros)
		if err != nil {
			return err
		}
		m.runDurationSummary = runDurationSummaryMicros.WithLabelValues(model)

		batchSizeSummary := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  "mly",
				Subsystem:  "client",
				Name:       "batch_size_summary",
				Help:       promDescBatchSize,
				Objectives: buckets.CommonSummaryObjectives,
			},
			[]string{"model"},
		)

		err = register(batchSizeSummary)
		if err != nil {
			return err
		}
		m.batchSizeSummary = batchSizeSummary.WithLabelValues(model)

		httpDurationSummaryMicros := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  "mly",
				Subsystem:  "client",
				Name:       "http_duration_summary_us",
				Help:       promDescHTTPDuration,
				Objectives: buckets.CommonSummaryObjectives,
			},
			[]string{"model"},
		)

		err = register(httpDurationSummaryMicros)
		if err != nil {
			return err
		}
		m.httpDurationSummary = httpDurationSummaryMicros.WithLabelValues(model)

		httpClientDurationSummaryMicros := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  "mly",
				Subsystem:  "client",
				Name:       "http_client_duration_summary_us",
				Help:       promDescHTTPClientDuration,
				Objectives: buckets.CommonSummaryObjectives,
			},
			[]string{"model"},
		)

		err = register(httpClientDurationSummaryMicros)
		if err != nil {
			return err
		}
		m.httpClientDurationSummary = httpClientDurationSummaryMicros.WithLabelValues(model)
	}

	runDurationHistogramMicros := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mly",
			Subsystem: "client",
			Name:      "run_duration_histogram_us",
			Help:      promDescRunDuration,
			Buckets:   buckets.MicrosecondBuckets,
		},
		[]string{"model"},
	)

	err = register(runDurationHistogramMicros)
	if err != nil {
		return err
	}
	m.runDurationHistogram = runDurationHistogramMicros.WithLabelValues(model)

	batchSizeHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mly",
			Subsystem: "client",
			Name:      "batch_size_histogram",
			Help:      promDescBatchSize,
			Buckets:   []float64{1, 2, 3, 4, 5, 7, 10, 12, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
		},
		[]string{"model"},
	)
	err = register(batchSizeHistogram)
	if err != nil {
		return err
	}
	m.batchSizeHistogram = batchSizeHistogram.WithLabelValues(model)

	httpDurationHistogramMicros := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mly",
			Subsystem: "client",
			Name:      "http_duration_histogram_us",
			Help:      promDescHTTPDuration,
			Buckets:   buckets.MicrosecondBuckets,
		},
		[]string{"model"},
	)
	err = register(httpDurationHistogramMicros)
	if err != nil {
		return err
	}
	m.httpDurationHistogram = httpDurationHistogramMicros.WithLabelValues(model)

	httpClientDurationHistogramMicros := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mly",
			Subsystem: "client",
			Name:      "http_client_duration_histogram_us",
			Help:      promDescHTTPClientDuration,
			Buckets:   buckets.MicrosecondBuckets,
		},
		[]string{"model"},
	)

	err = register(httpClientDurationHistogramMicros)
	if err != nil {
		return err
	}
	m.httpClientDurationHistogram = httpClientDurationHistogramMicros.WithLabelValues(model)

	err = register(runErrorCounter)
	if err != nil {
		return err
	}

	m.runErrorEarlyCtxCounter = runErrorCounter.WithLabelValues(model, "earlyCtx")

	// convenience function
	mkBECs := func(bec *promc.BaseErrorCounters, counter *prometheus.CounterVec) {
		bec.OtherErrorCounter = counter.WithLabelValues(model, "error")
		bec.DeadlineExceededCounter = counter.WithLabelValues(model, "deadlineExceeded")
		bec.CanceledCounter = counter.WithLabelValues(model, "canceled")
	}

	mkBECs(&m.runBaseErrorCounters, runErrorCounter)

	err = register(httpErrorCounter)
	if err != nil {
		return err
	}
	mkBECs(&m.httpBaseErrorCounters, httpErrorCounter)

	m.httpDownCounter = httpErrorCounter.WithLabelValues(model, "down")

	err = register(httpClientErrorCounter)
	if err != nil {
		return err
	}
	mkBECs(&m.httpClientBaseErrorCounters, runErrorCounter)

	return nil
}

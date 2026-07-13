package triton

import (
	"context"
	"time"

	grpcProm "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/mly/shared/stat/buckets"
)

var (
	inferDurationMicrosHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mly",
			Subsystem: "triton",
			Name:      "infer_duration_histogram_us",
			Help:      "Duration of Triton ModelInfer RPCs, labeled by model name, successful only.",
			Buckets:   []float64{100, 1000, 10000},
		},
		[]string{"model"},
	)

	inferDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "triton",
			Name:       "infer_duration_summary_us",
			Help:       "Duration of Triton ModelInfer RPCs, labeled by model name, successful only.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{"model"},
	)

	serverReadyDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "triton",
			Name:       "server_ready_duration_summary_us",
			Help:       "Duration of Triton ServerReady RPCs, labeled by model name, successful only.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{},
	)

	modelReadyDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "triton",
			Name:       "model_ready_duration_summary_us",
			Help:       "Duration of Triton ModelReady RPCs, labeled by model name, successful only.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{"model"},
	)

	modelLoadDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "triton",
			Name:       "model_load_duration_summary_us",
			Help:       "Duration of Triton ModelLoad RPCs, labeled by model name, successful only.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{"model"},
	)

	modelUnloadDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "triton",
			Name:       "model_unload_duration_summary_us",
			Help:       "Duration of Triton ModelUnload RPCs, labeled by model name, successful only.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{"model"},
	)

	modelMetadataDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "triton",
			Name:       "model_metadata_duration_summary_us",
			Help:       "Duration of Triton ModelMetadata RPCs, labeled by model name, successful only.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{"model"},
	)
)

func init() {
	// go-grpc-prometheus seems to force us to use Prometheus's DefaultRegisterer.
	grpcProm.EnableClientHandlingTimeHistogram(grpcProm.WithHistogramBuckets(buckets.SecondBuckets))

	prometheus.MustRegister(inferDurationMicrosHist)
	prometheus.MustRegister(inferDurationMicrosSummary)
	prometheus.MustRegister(serverReadyDurationMicrosSummary)
	prometheus.MustRegister(modelReadyDurationMicrosSummary)
	prometheus.MustRegister(modelLoadDurationMicrosSummary)
	prometheus.MustRegister(modelUnloadDurationMicrosSummary)
	prometheus.MustRegister(modelMetadataDurationMicrosSummary)
}

type MeteredTritonClient struct {
	client TritonClient
}

func NewMeteredTritonClient(client TritonClient) *MeteredTritonClient {
	return &MeteredTritonClient{
		client: client,
	}
}

func withGatherers(fn func() error, fnObserve func(float64)) error {
	startTime := time.Now()

	err := fn()
	if err != nil {
		return err
	}

	duration := float64(time.Since(startTime).Microseconds())
	fnObserve(duration)
	return nil
}

func (c *MeteredTritonClient) ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) (map[string]interface{}, error) {
	var result map[string]interface{}
	var err error

	err = withGatherers(func() error {
		result, err = c.client.ModelInfer(ctx, modelName, inputs, indexToName)
		return err
	}, func(duration float64) {
		inferDurationMicrosHist.WithLabelValues(modelName).Observe(duration)
		inferDurationMicrosSummary.WithLabelValues(modelName).Observe(duration)
	})

	return result, err
}

func (c *MeteredTritonClient) ServerReady(ctx context.Context) error {
	err := withGatherers(func() error {
		return c.client.ServerReady(ctx)
	}, func(duration float64) {
		serverReadyDurationMicrosSummary.WithLabelValues().Observe(duration)
	})

	return err
}

func (c *MeteredTritonClient) ModelReady(ctx context.Context, modelName string) (bool, error) {
	var ready bool
	var err error
	err = withGatherers(func() error {
		ready, err = c.client.ModelReady(ctx, modelName)
		return err
	}, func(duration float64) {
		modelReadyDurationMicrosSummary.WithLabelValues(modelName).Observe(duration)
	})

	return ready, err
}

func (c *MeteredTritonClient) ModelLoad(ctx context.Context, modelName string) error {
	err := withGatherers(func() error {
		return c.client.ModelLoad(ctx, modelName)
	}, func(duration float64) {
		modelLoadDurationMicrosSummary.WithLabelValues(modelName).Observe(duration)
	})
	return err
}

func (c *MeteredTritonClient) ModelUnload(ctx context.Context, modelName string) error {
	err := withGatherers(func() error {
		return c.client.ModelUnload(ctx, modelName)
	}, func(duration float64) {
		modelUnloadDurationMicrosSummary.WithLabelValues(modelName).Observe(duration)
	})
	return err
}

func (c *MeteredTritonClient) ModelMetadata(ctx context.Context, modelName string) (*ModelMetadata, error) {
	var metadata *ModelMetadata
	var err error
	err = withGatherers(func() error {
		metadata, err = c.client.ModelMetadata(ctx, modelName)
		return err
	}, func(duration float64) {
		modelMetadataDurationMicrosSummary.WithLabelValues(modelName).Observe(duration)
	})

	return metadata, err
}

func (c *MeteredTritonClient) Close() error {
	return c.client.Close()
}

package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/mly/shared/stat/buckets"
)

var (
	routerReloadDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "router",
			Name:       "reload_duration_summary_us",
			Help:       "Duration of router reloads.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{"router", "mode"},
	)

	routerPredictDurationMicrosSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "mly",
			Subsystem:  "router",
			Name:       "predict_duration_summary_us",
			Help:       "Duration of router predictions.",
			Objectives: buckets.CommonSummaryObjectives,
		},
		[]string{"router", "fixed_only"},
	)

	routerWorkerChannelQueuedSummary = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: "mly",
			Subsystem: "router",
			Name:      "worker_channel_queued_summary",
			Help:      "Number of router predictions queued in the worker channel.",
		},
	)

	routerPredictDroppedCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "mly",
			Subsystem: "router",
			Name:      "predict_dropped_counter",
			Help:      "Number of router predictions dropped.",
		},
	)

	routerModelUnloadGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "mly",
			Subsystem: "router",
			Name:      "model_unloading",
			Help:      "Number of models currently being unloaded.",
		},
	)
)

func init() {
	prometheus.MustRegister(routerPredictDurationMicrosSummary)
	prometheus.MustRegister(routerReloadDurationMicrosSummary)
	prometheus.MustRegister(routerModelUnloadGauge)
	prometheus.MustRegister(routerPredictDroppedCounter)
	prometheus.MustRegister(routerWorkerChannelQueuedSummary)
}

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

	routerPredictDroppedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mly",
			Subsystem: "router",
			Name:      "predict_dropped_counter",
			Help:      "Number of router predictions dropped.",
		},
		[]string{"router"},
	)

	routerModelUnloadGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "mly",
			Subsystem: "router",
			Name:      "model_unloading",
			Help:      "Number of models currently being unloaded.",
		},
		[]string{"router"},
	)
)

func init() {
	prometheus.MustRegister(routerPredictDurationMicrosSummary)
	prometheus.MustRegister(routerReloadDurationMicrosSummary)
	prometheus.MustRegister(routerModelUnloadGauge)
	prometheus.MustRegister(routerPredictDroppedCounter)
}

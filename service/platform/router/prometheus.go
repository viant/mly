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

	routerRoutedModelsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mly",
			Subsystem: "router",
			Name:      "routed_models_counter",
			Help:      "Number of models routed, labeled by model name.",
		},
		[]string{"model", "router"},
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
	prometheus.MustRegister(routerRoutedModelsCounter)
	prometheus.MustRegister(routerReloadDurationMicrosSummary)
	prometheus.MustRegister(routerModelUnloadGauge)
}

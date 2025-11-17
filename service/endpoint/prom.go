package endpoint

import (
	"github.com/prometheus/client_golang/prometheus"
)

func registerPrometheusMetrics(promReg prometheus.Registerer) {
	// promReg.Register(prometheus.NewHistogram(prometheus.HistogramOpts{
	// 	Namespace: "mly",
	// 	Subsystem: "endpoint",
	// 	Name:      "request_duration_seconds",
	// 	Help:      "Request duration",
	// 	Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	// }))
}

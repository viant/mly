package prometheus

import (
	"net/http"
	"regexp"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Handler(pr *prom.Registry) http.HandlerFunc {
	r := regexp.MustCompile("\\/sched.*")
	collOpts := collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{r})
	pr.MustRegister(collectors.NewGoCollector(collOpts))

	promHandler := promhttp.HandlerFor(pr, promhttp.HandlerOpts{})
	return promHandler.ServeHTTP
}

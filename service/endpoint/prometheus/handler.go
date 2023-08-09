package prometheus

import (
	"net/http"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Handler() http.HandlerFunc {
	pr := prometheus.NewRegistry()

	r := regexp.MustCompile("\\/sched.*")
	collOpts := collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{r})
	pr.MustRegister(collectors.NewGoCollector(collOpts))

	//pr.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	promHandler := promhttp.HandlerFor(pr, promhttp.HandlerOpts{})
	return promHandler.ServeHTTP
}

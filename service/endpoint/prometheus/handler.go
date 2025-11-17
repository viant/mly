package prometheus

import (
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Handler(gatherer prom.Gatherer) http.HandlerFunc {
	promHandler := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
	return promHandler.ServeHTTP
}

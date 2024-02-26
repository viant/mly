package config

type MetricsConfig struct {
	Prometheus *PromConfig `json:",omitempty" yaml:",omitempty"`
}

type PromConfig struct {
	// See service/endpoint.Build
	ModelIdletimeBuckets []float64
}

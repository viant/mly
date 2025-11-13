package config

type TritonServer struct {
	ID string

	// HTTPBaseURL is the base URL for the Triton HTTP server.
	// Defaults to http://localhost:8000
	HTTPBaseURL string `json:",omitempty" yaml:",omitempty"`

	// GRPCBaseURL is the base URL for the Triton GRPC server.
	// Defaults to localhost:8001
	GRPCBaseURL string `json:",omitempty" yaml:",omitempty"`

	// StartupTimeoutSeconds is the timeout for the Triton server to start up.
	// Defaults to 30 seconds.
	StartupTimeoutSeconds int `json:",omitempty" yaml:",omitempty"`
}

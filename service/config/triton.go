package config

import "fmt"

type TritonServer struct {
	ID string

	// HTTPBaseURL is the base URL for the Triton HTTP server.
	// Defaults to http://localhost:8000
	HTTPBaseURL string `json:",omitempty" yaml:",omitempty"`

	// HTTPClientTimeoutMs is the timeout for the Triton HTTP client to respond to a request.
	// Defaults to 100 milliseconds.
	HTTPClientTimeoutMs int `json:",omitempty" yaml:",omitempty"`

	// GRPCBaseURL is the base URL for the Triton GRPC server.
	// Defaults to localhost:8001
	GRPCBaseURL string `json:",omitempty" yaml:",omitempty"`

	GRPCConnectParams GRPCConnectParams `json:",omitempty" yaml:",omitempty"`

	// StartupTimeoutSeconds is the timeout for the Triton server to start up.
	// Defaults to 10 seconds.
	StartupTimeoutSeconds int `json:",omitempty" yaml:",omitempty"`
}

// Copy of google.golang.org/grpc/backoff.Config (https://pkg.go.dev/google.golang.org/grpc/backoff#Config)

type GRPCConnectParams struct {
	// Defaults to 10 milliseconds.
	BaseDelayMs int `json:",omitempty" yaml:",omitempty"`

	// Defaults to 2.
	Multiplier float64 `json:",omitempty" yaml:",omitempty"`

	// Defaults to 0.1.
	Jitter float64 `json:",omitempty" yaml:",omitempty"`

	// Defaults to 150 milliseconds.
	MaxDelayMs int `json:",omitempty" yaml:",omitempty"`
}

func (t *TritonServer) Init() {
	if t.StartupTimeoutSeconds == 0 {
		t.StartupTimeoutSeconds = 10
	}

	if t.HTTPClientTimeoutMs == 0 {
		t.HTTPClientTimeoutMs = 100
	}

	if t.GRPCConnectParams.BaseDelayMs == 0 {
		t.GRPCConnectParams.BaseDelayMs = 10
	}

	if t.GRPCConnectParams.Multiplier == 0 {
		t.GRPCConnectParams.Multiplier = 2
	}

	if t.GRPCConnectParams.Jitter == 0 {
		t.GRPCConnectParams.Jitter = 0.1
	}

	if t.GRPCConnectParams.MaxDelayMs == 0 {
		t.GRPCConnectParams.MaxDelayMs = 150
	}
}

func (t *TritonServer) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("triton server ID is required")
	}

	if t.HTTPBaseURL == "" && t.GRPCBaseURL == "" {
		return fmt.Errorf("triton server HTTPBaseURL or GRPCBaseURL must be set for server %s", t.ID)
	}

	return nil
}

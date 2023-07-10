package config

import (
	"time"
)

// Endpoint represents an endpoint
type Endpoint struct {
	Port           int
	ReadTimeoutMs  int           `json:",omitempty" yaml:",omitempty"`
	WriteTimeoutMs int           `json:",omitempty" yaml:",omitempty"`
	WriteTimeout   time.Duration `json:",omitempty" yaml:",omitempty"`
	MaxHeaderBytes int           `json:",omitempty" yaml:",omitempty"`

	// HTTP data buffer pool - used when reading a payload, for saving memory
	PoolMaxSize int `json:",omitempty" yaml:",omitempty"`
	BufferSize  int `json:",omitempty" yaml:",omitempty"`

	MaxEvaluatorConcurrency int32 `json:",omitempty" yaml:",omitempty"`
}

//Init init applied default settings
func (e *Endpoint) Init() {
	if e.Port == 0 {
		e.Port = 8080
	}
	if e.ReadTimeoutMs == 0 {
		e.ReadTimeoutMs = 5000
	}
	if e.WriteTimeoutMs == 0 {
		e.WriteTimeoutMs = 5000
	}
	if e.WriteTimeout == 0 {
		e.WriteTimeout = time.Duration(e.WriteTimeoutMs) * time.Millisecond
	}

	if e.MaxHeaderBytes == 0 {
		e.MaxHeaderBytes = 8 * 1024
	}

	if e.PoolMaxSize == 0 {
		e.PoolMaxSize = 512
	}
	if e.BufferSize == 0 {
		e.BufferSize = 128 * 1024
	}

	if e.MaxEvaluatorConcurrency <= 0 {
		e.MaxEvaluatorConcurrency = 3000
	}
}

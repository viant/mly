package config

import (
	"time"
)

type Endpoint struct {
	Port           int
	ReadTimeoutMs  int
	WriteTimeoutMs int
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	PoolMaxSize    int
	BufferSize     int
}

//Init init applied default settings
func (e *Endpoint) Init() {
	if e.Port == 0 {
		e.Port = 8080
	}
	if e.ReadTimeoutMs == 0 {
		e.ReadTimeoutMs = 5000
	}
	if e.ReadTimeoutMs == 0 {
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
		e.BufferSize = 1024
	}
}

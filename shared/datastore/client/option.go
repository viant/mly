package client

import aero "github.com/aerospike/aerospike-client-go"

type Option func(srv *Service)

// WithClientPolicy will provide an initial ClientPolicy.
// Even when overridden, the timeout will be applied UNLESS using WithBypassConfiguredTimeout.
func WithClientPolicy(clientPolicy *aero.ClientPolicy) Option {
	return func(srv *Service) {
		srv.clientPolicy = clientPolicy
	}
}

// WithBasePolicy will provide an initial BasePolicy.
// Even when overridden, the timeout will be applied UNLESS using WithBypassConfiguredTimeout.
func WithBasePolicy(basePolicy *aero.BasePolicy) Option {
	return func(srv *Service) {
		srv.basePolicy = basePolicy
	}
}

// WithBypassConfiguredTimeout will bypass the configured timeout if using WithClientPolicy or WithBasePolicy.
func WithBypassConfiguredTimeout() Option {
	return func(srv *Service) {
		srv.bypassConfiguredTimeout = true
	}
}

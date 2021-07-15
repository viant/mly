package client

import (
	"crypto/x509"
	"sync"
)

var certPool *x509.CertPool
var certErr error
var certOnce = sync.Once{}

func getCertPool() (*x509.CertPool, error) {
	certOnce.Do(func() {
		certPool, certErr = x509.SystemCertPool()

	})
	return certPool, certErr
}

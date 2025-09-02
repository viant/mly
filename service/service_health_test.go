package service

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that HealthStatus field works correctly
func TestService_HealthStatusField(t *testing.T) {
	srv := &Service{}

	// Verify the field exists and can be used
	assert.Equal(t, int32(0), srv.HealthStatus)

	// Test atomic operations
	atomic.StoreInt32(&srv.HealthStatus, 1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&srv.HealthStatus))

	atomic.StoreInt32(&srv.HealthStatus, 0)
	assert.Equal(t, int32(0), atomic.LoadInt32(&srv.HealthStatus))

	// Test that it works with pointers (for health endpoint)
	healthPtr := &srv.HealthStatus
	atomic.StoreInt32(healthPtr, 1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&srv.HealthStatus))
}

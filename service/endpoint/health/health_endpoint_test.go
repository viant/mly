package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/config"
)

func TestHealthHandler_WithHealthStatus(t *testing.T) {
	handler := NewHealthHandler()

	// Test basic health reporting with new HealthStatus field
	srv := &service.Service{}
	atomic.StoreInt32(&srv.HealthStatus, 1) // Healthy

	model := &config.Model{ID: "test_model"}
	handler.Hook(model, srv)

	// Create HTTP request
	req := httptest.NewRequest("GET", "/v1/health", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response
	var response map[string]*int32
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify model is healthy
	require.Contains(t, response, "test_model")
	assert.Equal(t, int32(1), *response["test_model"])

	// Test dynamic health changes
	atomic.StoreInt32(&srv.HealthStatus, 0)

	req2 := httptest.NewRequest("GET", "/v1/health", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	var response2 map[string]*int32
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	require.NoError(t, err)
	assert.Equal(t, int32(0), *response2["test_model"])
}

// Test backward compatibility - health endpoint API unchanged
func TestHealthHandler_BackwardCompatibility(t *testing.T) {
	handler := NewHealthHandler()

	// The health endpoint API should remain the same for clients
	srv := &service.Service{}
	atomic.StoreInt32(&srv.HealthStatus, 1)

	model := &config.Model{ID: "compat_test"}
	handler.Hook(model, srv)

	// Create HTTP request
	req := httptest.NewRequest("GET", "/v1/health", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.ServeHTTP(w, req)

	// Response format should be the same as before
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response
	var response map[string]*int32
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should have same structure as before (map of model names to health pointers)
	require.Contains(t, response, "compat_test")
	assert.Equal(t, int32(1), *response["compat_test"])
}

package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/config"
)

// Test backward compatibility - health endpoint API unchanged
func TestHealthHandler_BackwardCompatibility(t *testing.T) {
	handler := NewHealthHandler()

	// The health endpoint API should remain the same for clients
	srv := &service.Service{
		ReloadOK: 1,
	}

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

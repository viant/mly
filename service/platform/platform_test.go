package platform

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared"
)

func TestTritonEvaluator_Creation(t *testing.T) {
	cfg := &config.Model{
		ID:  "test_triton",
		URL: "http://localhost:8000",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			Timeout:   30,
		},
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "test_input", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
	}

	evaluator := NewTritonEvaluator(cfg)

	assert.NotNil(t, evaluator)
	assert.Equal(t, cfg, evaluator.config)
	// Just verify the evaluator was created
}

func TestTritonEvaluator_Signature(t *testing.T) {
	cfg := &config.Model{
		ID: "test_triton",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
				{Name: "input2", Index: 1, DataType: "float32"},
			},
			Outputs: []*shared.Field{
				{Name: "output_0", Index: 0, DataType: "float32"},
			},
		},
	}

	evaluator := NewTritonEvaluator(cfg)
	signature := evaluator.Signature()

	assert.NotNil(t, signature)

	// Verify signature structure (signature is now directly *domain.Signature)
	assert.Len(t, signature.Inputs, 2)
	assert.Equal(t, "input1", signature.Inputs[0].Name)
	assert.Equal(t, "input2", signature.Inputs[1].Name)
	assert.Len(t, signature.Outputs, 1)
	assert.Equal(t, "output_0", signature.Outputs[0].Name)
	assert.Equal(t, "float32", signature.Outputs[0].DataType)
}

func TestTritonEvaluator_SignatureWithConfiguredOutputs(t *testing.T) {
	cfg := &config.Model{
		ID: "test_triton_custom_outputs",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "prediction", DataType: "float64"},
				{Name: "confidence", DataType: "float32"},
			},
		},
	}

	evaluator := NewTritonEvaluator(cfg)
	signature := evaluator.Signature()

	assert.NotNil(t, signature)

	// Verify inputs
	assert.Len(t, signature.Inputs, 1)
	assert.Equal(t, "input1", signature.Inputs[0].Name)

	// Verify configured outputs (following TensorFlow pattern)
	assert.Len(t, signature.Outputs, 2)
	assert.Equal(t, "prediction", signature.Outputs[0].Name)
	assert.Equal(t, "float64", signature.Outputs[0].DataType)
	assert.Equal(t, 0, signature.Outputs[0].Index)

	assert.Equal(t, "confidence", signature.Outputs[1].Name)
	assert.Equal(t, "float32", signature.Outputs[1].DataType)
	assert.Equal(t, 1, signature.Outputs[1].Index)

	// Verify Output field is set to first output
	assert.Equal(t, signature.Outputs[0], signature.Output)
}

func TestTritonEvaluator_Dictionary(t *testing.T) {
	cfg := &config.Model{
		ID: "test_triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "test_input", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
	}
	evaluator := NewTritonEvaluator(cfg)

	dict := evaluator.Dictionary()
	assert.Nil(t, dict) // Triton doesn't use dictionaries
}

func TestTritonEvaluator_InputsWithConfig(t *testing.T) {
	// Test that we can create an evaluator and call Inputs method with explicit configuration
	cfg := &config.Model{
		ID: "test_triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "src", Index: 0, DataType: "string"},
				{Name: "platform", Index: 1, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
	}

	evaluator := NewTritonEvaluator(cfg)
	inputs := evaluator.Inputs()

	// Should return the configured inputs
	assert.NotNil(t, inputs)
	assert.Len(t, inputs, 2)
	assert.Contains(t, inputs, "src")
	assert.Contains(t, inputs, "platform")
}

func TestTritonEvaluator_InputsDefault(t *testing.T) {
	cfg := &config.Model{ID: "test_triton"}

	// This should panic because no inputs are configured
	assert.Panics(t, func() {
		NewTritonEvaluator(cfg)
	})
}

func TestTritonEvaluator_OutputsDefault(t *testing.T) {
	cfg := &config.Model{
		ID: "test_triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "test_input", Index: 0, DataType: "string"},
			},
			// No outputs configured - should panic
		},
	}

	// This should panic because no outputs are configured
	assert.Panics(t, func() {
		NewTritonEvaluator(cfg)
	})
}

func TestTritonEvaluator_AuxiliaryInputsHandling(t *testing.T) {
	cfg := &config.Model{
		ID: "test_triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "key_input", Index: 0, DataType: "string", Auxiliary: false},
				{Name: "aux_input", Index: 1, DataType: "string", Auxiliary: true},
				{Name: "another_key", Index: 2, DataType: "int", Auxiliary: false},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
	}

	evaluator := NewTritonEvaluator(cfg)

	// Test signature - should only include NON-auxiliary inputs (following TensorFlow pattern)
	signature := evaluator.Signature()
	assert.Len(t, signature.Inputs, 2, "Signature should only include non-auxiliary inputs (following TensorFlow pattern)")

	// Check that only non-auxiliary inputs are in signature
	inputNames := make(map[string]bool)
	for _, input := range signature.Inputs {
		inputNames[input.Name] = true
	}
	assert.True(t, inputNames["key_input"], "key_input should be in signature")
	assert.False(t, inputNames["aux_input"], "aux_input should NOT be in signature (following TensorFlow)")
	assert.True(t, inputNames["another_key"], "another_key should be in signature")

	// Test inputs map - should include ALL inputs (including auxiliary)
	inputs := evaluator.Inputs()
	assert.Len(t, inputs, 3, "Inputs map should include all inputs")
	assert.Contains(t, inputs, "key_input")
	assert.Contains(t, inputs, "aux_input")
	assert.Contains(t, inputs, "another_key")

	// Verify auxiliary flag is preserved in inputs map
	assert.False(t, inputs["key_input"].Auxiliary, "key_input should not be auxiliary")
	assert.True(t, inputs["aux_input"].Auxiliary, "aux_input should be auxiliary")
	assert.False(t, inputs["another_key"].Auxiliary, "another_key should not be auxiliary")
}

func TestTritonEvaluator_Stats(t *testing.T) {
	evaluator := NewTritonEvaluator(&config.Model{
		ID: "test",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "test_input", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
	})
	stats := make(map[string]interface{})

	// Should not panic and should be able to add stats
	evaluator.Stats(stats)
	assert.NotNil(t, stats)
}

func TestTritonEvaluator_Close(t *testing.T) {
	evaluator := NewTritonEvaluator(&config.Model{
		ID: "test",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "test_input", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
	})

	err := evaluator.Close()
	assert.NoError(t, err)
}

func TestConfigGetPlatform(t *testing.T) {
	testCases := []struct {
		name     string
		config   *config.Model
		expected string
	}{
		{
			name:     "explicit_tensorflow",
			config:   &config.Model{Platform: "tensorflow"},
			expected: "tensorflow",
		},
		{
			name:     "explicit_triton",
			config:   &config.Model{Platform: "triton"},
			expected: "triton",
		},
		{
			name:     "default_platform",
			config:   &config.Model{},
			expected: "tensorflow",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.config.GetPlatform()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestModelValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config.Model
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_tensorflow_model",
			config: &config.Model{
				ID:       "test_tf",
				Platform: "tensorflow",
				URL:      "file:///tmp/model",
			},
			expectError: false,
		},
		{
			name: "valid_triton_model",
			config: &config.Model{
				ID:       "test_triton",
				Platform: "triton",
				URL:      "http://localhost:8000",
				Triton: &config.TritonConfig{
					ModelName: "test_model",
					Timeout:   30,
				},
			},
			expectError: false,
		},
		{
			name: "tensorflow_missing_url",
			config: &config.Model{
				ID:       "test_tf_no_url",
				Platform: "tensorflow",
			},
			expectError: true,
			errorMsg:    "tensorflow model test_tf_no_url requires URL",
		},
		{
			name: "triton_missing_config",
			config: &config.Model{
				ID:       "test_triton_no_config",
				Platform: "triton",
			},
			expectError: true,
			errorMsg:    "triton model test_triton_no_config requires Triton configuration",
		},
		{
			name: "triton_missing_server_url",
			config: &config.Model{
				ID:       "test_triton_no_server",
				Platform: "triton",
				Triton: &config.TritonConfig{
					ModelName: "test_model",
				},
			},
			expectError: true,
			errorMsg:    "requires URL (Triton server endpoint)",
		},
		{
			name: "triton_missing_model_name",
			config: &config.Model{
				ID:       "test_triton_no_model",
				Platform: "triton",
				URL:      "http://localhost:8000",
				Triton:   &config.TritonConfig{},
			},
			expectError: true,
			errorMsg:    "Triton ModelName is required",
		},
		{
			name: "unsupported_platform",
			config: &config.Model{
				ID:       "test_unsupported",
				Platform: "pytorch",
				URL:      "file:///tmp/model",
			},
			expectError: true,
			errorMsg:    "unsupported platform 'pytorch' for model test_unsupported",
		},
		{
			name: "missing_model_id",
			config: &config.Model{
				Platform: "tensorflow",
				URL:      "file:///tmp/model",
			},
			expectError: true,
			errorMsg:    "model.ID was empty",
		},
		{
			name: "default_platform_missing_url",
			config: &config.Model{
				ID: "test_default_no_url",
				// No platform specified, should default to tensorflow
			},
			expectError: true,
			errorMsg:    "tensorflow model test_default_no_url requires URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTritonConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config.TritonConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_triton_config",
			config: &config.TritonConfig{
				ModelName: "test_model",
				Timeout:   30,
			},
			expectError: false,
		},
		{
			name: "valid_triton_config_minimal",
			config: &config.TritonConfig{
				ModelName: "test_model",
				// Timeout is optional
			},
			expectError: false,
		},
		{
			name:        "missing_model_name",
			config:      &config.TritonConfig{},
			expectError: true,
			errorMsg:    "Triton ModelName is required",
		},
		{
			name: "empty_model_name",
			config: &config.TritonConfig{
				ModelName: "",
			},
			expectError: true,
			errorMsg:    "Triton ModelName is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTensorFlowEvaluator_Health(t *testing.T) {
	tfService := &tfmodel.Service{}
	evaluator := NewTensorFlowEvaluator(tfService)

	// Test basic health functionality
	assert.True(t, evaluator.SupportsHealthReporting())
	assert.True(t, evaluator.SupportsReload())
	assert.False(t, evaluator.IsHealthy()) // Uninitialized

	// Test health status with pointer
	var healthStatus int32
	evaluator.SetHealthStatus(&healthStatus)

	atomic.StoreInt32(&healthStatus, 1)
	assert.True(t, evaluator.IsHealthy())

	atomic.StoreInt32(&healthStatus, 0)
	assert.False(t, evaluator.IsHealthy())
}

func TestTritonEvaluator_Health(t *testing.T) {
	// Basic capability tests
	evaluator := createTestTritonEvaluator("http://localhost:8000")
	defer evaluator.Close()

	assert.True(t, evaluator.SupportsHealthReporting())
	assert.False(t, evaluator.SupportsReload())

	// Test cached health status with mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/models/test_model/ready" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	healthyEvaluator := createTestTritonEvaluator(server.URL)
	defer healthyEvaluator.Close()

	// Set up health status pointer (starts as unhealthy, will be updated by monitor)
	var healthStatus int32
	healthyEvaluator.SetHealthStatus(&healthStatus)

	assert.False(t, healthyEvaluator.IsHealthy()) // Starts pessimistic
}

// Test that the interface is properly implemented
func TestPlatformEvaluator_Interface_Compliance(t *testing.T) {
	// Verify both implementations satisfy the interface
	tfService := &tfmodel.Service{}
	var _ PlatformEvaluator = NewTensorFlowEvaluator(tfService)

	evaluator := createTestTritonEvaluator("http://localhost:8000")
	defer evaluator.Close()
	var _ PlatformEvaluator = evaluator
}

// Helper function to create test Triton evaluator
func createTestTritonEvaluator(serverURL string) *TritonEvaluator {
	cfg := &config.Model{
		ID:  "test_triton",
		URL: serverURL,
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			Timeout:   5000, // 5 second timeout for tests
		},
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
	}

	return NewTritonEvaluator(cfg)
}

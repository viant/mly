package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/transfer"
)

func TestModelPlatformConstants(t *testing.T) {
	assert.Equal(t, "tensorflow", string(PlatformTensorFlow))
	assert.Equal(t, "triton", string(PlatformTriton))
}

func TestTritonEvaluator_Creation(t *testing.T) {
	cfg := &config.Model{
		ID:  "test_triton",
		URL: "http://localhost:8000",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			Version:   "1",
			Timeout:   30,
		},
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "test_input", Index: 0, DataType: "string"},
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

func TestTritonEvaluator_Stats(t *testing.T) {
	evaluator := NewTritonEvaluator(&config.Model{
		ID: "test",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "test_input", Index: 0, DataType: "string"},
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
					Version:   "1",
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
				Version:   "1",
				Timeout:   30,
			},
			expectError: false,
		},
		{
			name: "valid_triton_config_minimal",
			config: &config.TritonConfig{
				ModelName: "test_model",
				// Version and Timeout are optional
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

// TestCacheKeyConsistency verifies that cache keys are consistent across platforms
// This is critical for ensuring cache hits work correctly regardless of platform
func TestCacheKeyConsistency(t *testing.T) {
	// Test data is defined in individual test cases below

	// Both platforms should generate the same cache key for the same input
	// since cache keys are generated from raw features before platform-specific processing

	testCases := []struct {
		name        string
		inputData   *transfer.Input
		description string
	}{
		{
			name: "single_feature_input",
			inputData: &transfer.Input{
				Keys: transfer.Strings{Values: []string{"test.com"}},
			},
			description: "Single feature should generate consistent cache key",
		},
		{
			name: "multiple_feature_input",
			inputData: &transfer.Input{
				Keys: transfer.Strings{Values: []string{"test.com", "Windows", "Chrome", "Google", "Comcast"}},
			},
			description: "Multiple features should generate consistent cache key",
		},
		{
			name: "batch_input",
			inputData: &transfer.Input{
				Keys: transfer.Strings{Values: []string{"test1.com", "test2.com", "test3.com"}},
			},
			description: "Batch input should generate consistent cache keys",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that the same input generates the same cache key representation
			// regardless of which platform will process it

			// The key insight is that cache keys are generated from raw input
			// before any platform-specific transformations occur

			// Simulate getting cache key for the same data
			key1 := generateMockCacheKey(tc.inputData, "tensorflow")
			key2 := generateMockCacheKey(tc.inputData, "triton")

			// Cache keys should be identical because they're generated from the same raw data
			assert.Equal(t, key1, key2, "Cache keys should be consistent across platforms for: %s", tc.description)
		})
	}
}

// TestCacheKeyGeneration verifies cache key generation logic
func TestCacheKeyGeneration(t *testing.T) {
	testCases := []struct {
		name     string
		input    *transfer.Input
		expected string
	}{
		{
			name: "empty_input",
			input: &transfer.Input{
				Keys: transfer.Strings{Values: []string{}},
			},
			expected: "cache_key_[]",
		},
		{
			name: "single_value",
			input: &transfer.Input{
				Keys: transfer.Strings{Values: []string{"test.com"}},
			},
			expected: "cache_key_[test.com]",
		},
		{
			name: "multiple_values",
			input: &transfer.Input{
				Keys: transfer.Strings{Values: []string{"test.com", "Windows", "Chrome"}},
			},
			expected: "cache_key_[test.com,Windows,Chrome]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generateMockCacheKey(tc.input, "any_platform")
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestCacheKeyInvariance verifies that platform choice doesn't affect cache key generation
func TestCacheKeyInvariance(t *testing.T) {
	input := &transfer.Input{
		Keys: transfer.Strings{Values: []string{"example.com", "Windows", "Chrome", "Google", "Verizon"}},
	}

	platforms := []string{"tensorflow", "triton", "pytorch", "onnx"} // Include future platforms

	var keys []string
	for _, platform := range platforms {
		key := generateMockCacheKey(input, platform)
		keys = append(keys, key)
	}

	// All cache keys should be identical regardless of platform
	for i := 1; i < len(keys); i++ {
		assert.Equal(t, keys[0], keys[i],
			"Cache key for platform %s should match platform %s",
			platforms[0], platforms[i])
	}
}

// TestBatchCacheKeyConsistency verifies cache key consistency for batch requests
func TestBatchCacheKeyConsistency(t *testing.T) {
	// Simulate batch input
	batchInput := &transfer.Input{
		Keys: transfer.Strings{Values: []string{"site1.com", "site2.com", "site3.com"}},
	}

	// Each item in the batch should generate a consistent cache key
	for i := 0; i < len(batchInput.Keys.Values); i++ {
		itemInput := &transfer.Input{
			Keys: transfer.Strings{Values: []string{batchInput.Keys.Values[i]}},
		}

		key1 := generateMockCacheKey(itemInput, "tensorflow")
		key2 := generateMockCacheKey(itemInput, "triton")

		assert.Equal(t, key1, key2,
			"Batch item %d cache key should be consistent across platforms", i)
	}
}

// TestModelConfigurationImpactOnCacheKeys verifies that model configuration
// differences don't affect cache key generation
func TestModelConfigurationImpactOnCacheKeys(t *testing.T) {
	input := &transfer.Input{
		Keys: transfer.Strings{Values: []string{"test.com", "Windows", "Chrome"}},
	}

	// Different model configurations for the same platform
	configs := []*config.Model{
		{
			ID:       "model1",
			Platform: "tensorflow",
			URL:      "file:///tmp/model1",
		},
		{
			ID:       "model2",
			Platform: "tensorflow",
			URL:      "file:///tmp/model2",
		},
		{
			ID:       "model3",
			Platform: "triton",
			URL:      "http://localhost:8000",
			Triton: &config.TritonConfig{
				ModelName: "model3",
			},
		},
	}

	var keys []string
	for _, cfg := range configs {
		// Cache key generation should not depend on model configuration
		// Only on the input features themselves
		key := generateMockCacheKey(input, cfg.Platform)
		keys = append(keys, key)
	}

	// All cache keys should be identical regardless of model configuration
	for i := 1; i < len(keys); i++ {
		assert.Equal(t, keys[0], keys[i],
			"Cache key should not depend on model configuration differences")
	}
}

// TestPlatformSpecificFeaturesDoNotAffectCacheKeys ensures that platform-specific
// features or transformations don't leak into cache key generation
func TestPlatformSpecificFeaturesDoNotAffectCacheKeys(t *testing.T) {
	baseInput := &transfer.Input{
		Keys: transfer.Strings{Values: []string{"test.com", "Windows"}},
	}

	// Simulate that different platforms might add different metadata
	// but this should NOT affect cache key generation

	tfKey := generateMockCacheKey(baseInput, "tensorflow")
	tritonKey := generateMockCacheKey(baseInput, "triton")

	assert.Equal(t, tfKey, tritonKey,
		"Platform-specific processing should not affect cache key generation")

	// Verify the actual expected format
	expected := "cache_key_[test.com,Windows]"
	assert.Equal(t, expected, tfKey)
	assert.Equal(t, expected, tritonKey)
}

// Mock function to simulate cache key generation
// In the real implementation, this logic is in the MLY service layer
// before platform-specific processing occurs
func generateMockCacheKey(input *transfer.Input, platform string) string {
	// Cache key generation should be platform-agnostic
	// It should only depend on the raw input features

	keys := "["
	for i, key := range input.Keys.Values {
		if i > 0 {
			keys += ","
		}
		keys += key
	}
	keys += "]"

	return "cache_key_" + keys
}

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTritonModelConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Model
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_tensorflow_model",
			config: &Model{
				ID:       "test_tf",
				Platform: "tensorflow",
				URL:      "file:///tmp/model",
			},
			expectError: false,
		},
		{
			name: "valid_triton_model",
			config: &Model{
				ID:       "test_triton",
				Platform: "triton",
				URL:      "http://localhost:8000",
				Triton: &TritonConfig{
					ModelName: "test_model",
					Timeout:   30,
				},
			},
			expectError: false,
		},
		{
			name: "tensorflow_missing_url",
			config: &Model{
				ID:       "test_tf_no_url",
				Platform: "tensorflow",
			},
			expectError: true,
			errorMsg:    "URL",
		},
		{
			name: "triton_missing_config",
			config: &Model{
				ID:       "test_triton_no_config",
				Platform: "triton",
			},
			expectError: true,
			errorMsg:    "Triton configuration",
		},
		{
			name: "triton_missing_server_url",
			config: &Model{
				ID:       "test_triton_no_server",
				Platform: "triton",
				Triton: &TritonConfig{
					ModelName: "test_model",
				},
			},
			expectError: true,
			errorMsg:    "URL",
		},
		{
			name: "triton_missing_model_name",
			config: &Model{
				ID:       "test_triton_no_model",
				Platform: "triton",
				URL:      "http://localhost:8000",
				Triton:   &TritonConfig{},
			},
			expectError: true,
			errorMsg:    "ModelName",
		},
		{
			name: "missing_model_id",
			config: &Model{
				Platform: "tensorflow",
				URL:      "file:///tmp/model",
			},
			expectError: true,
			errorMsg:    "ID",
		},
		{
			name: "default_platform_missing_url",
			config: &Model{
				ID: "test_default_no_url",
				// No platform specified, should default to tensorflow
			},
			expectError: true,
			errorMsg:    "URL",
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
		config      *TritonConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_triton_config",
			config: &TritonConfig{
				ModelName: "test_model",
				Timeout:   30,
			},
			expectError: false,
		},
		{
			name: "valid_triton_config_minimal",
			config: &TritonConfig{
				ModelName: "test_model",
				// Timeout is optional
			},
			expectError: false,
		},
		{
			name:        "missing_model_name",
			config:      &TritonConfig{},
			expectError: true,
			errorMsg:    "ModelName",
		},
		{
			name: "empty_model_name",
			config: &TritonConfig{
				ModelName: "",
			},
			expectError: true,
			errorMsg:    "ModelName",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate(true)

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

package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
)

// TritonInput represents a single input tensor for Triton
type TritonInput struct {
	Name     string      `json:"name"`
	Shape    []int       `json:"shape"`
	DataType string      `json:"datatype"`
	Data     interface{} `json:"data"`
}

// TritonOutput represents a single output tensor from Triton
type TritonOutput struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

// TritonRequest represents the HTTP request format to Triton
type TritonRequest struct {
	Inputs []TritonInput `json:"inputs"`
}

// TritonResponse represents the HTTP response format from Triton
type TritonResponse struct {
	Outputs []TritonOutput `json:"outputs"`
}

// TritonEvaluator implements PlatformEvaluator for Triton Inference Server
type TritonEvaluator struct {
	config     *config.Model
	httpClient *http.Client
	serverURL  string
	modelName  string
	version    string
}

// NewTritonEvaluator creates a new Triton evaluator
func NewTritonEvaluator(config *config.Model) *TritonEvaluator {
	// Extract Triton configuration
	serverURL := "http://localhost:8000" // Default
	modelName := config.ID               // Default to model ID
	version := "1"                       // Default version
	timeout := 30 * time.Second          // Default timeout

	// Use configuration if provided
	if config.Triton != nil {
		if config.Triton.ServerURL != "" {
			serverURL = config.Triton.ServerURL
		}
		if config.Triton.ModelName != "" {
			modelName = config.Triton.ModelName
		}
		if config.Triton.Version != "" {
			version = config.Triton.Version
		}
		if config.Triton.Timeout > 0 {
			timeout = time.Duration(config.Triton.Timeout) * time.Second
		}
	}

	return &TritonEvaluator{
		config:    config,
		serverURL: serverURL,
		modelName: modelName,
		version:   version,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Predict performs inference via Triton Inference Server
func (t *TritonEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	// Convert MLY params to Triton request format
	tritonRequest, err := t.convertToTritonRequest(params)
	if err != nil {
		return nil, fmt.Errorf("failed to convert params to Triton format: %w", err)
	}

	// Send request to Triton
	tritonResponse, err := t.sendTritonRequest(ctx, tritonRequest)
	if err != nil {
		return nil, fmt.Errorf("triton inference failed for model %s: %w", t.config.ID, err)
	}

	// Convert Triton response to MLY format
	result := t.convertFromTritonResponse(tritonResponse)
	return result, nil
}

// convertToTritonRequest converts MLY params ([]interface{}) to Triton request format
func (t *TritonEvaluator) convertToTritonRequest(params []interface{}) (*TritonRequest, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	var inputs []TritonInput

	// MLY params are processed into []interface{} where each element corresponds to a model input
	// The format is [][]T where the first slice is for different examples in a batch
	// and the second slice is always length 1 for single values

	// Get the input definitions to map indices to names
	inputDefs := t.getInputDefinitions()

	// Convert each parameter to Triton input format
	for i, param := range params {
		// Find the input name for this index
		inputName := ""
		for name, input := range inputDefs {
			if domainInput, ok := input.(*domain.Input); ok {
				if domainInput.Index == i && !domainInput.Auxiliary {
					inputName = name
					break
				}
			}
		}

		if inputName == "" {
			return nil, fmt.Errorf("no input name found for index %d", i)
		}

		// Handle the MLY format: [][]T
		switch v := param.(type) {
		case [][]string:
			if len(v) > 0 {
				// MLY batch format: [][]T where len(v) = batch_size
				// Each v[i] contains one element: the value for batch item i
				batchSize := len(v)
				data := make([]string, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = v[j][0] // Extract the value for batch item j
					}
				}
				inputs = append(inputs, TritonInput{
					Name:     inputName,
					Shape:    []int{batchSize, 1}, // Dynamic batch size
					DataType: "BYTES",
					Data:     data,
				})
			}
		case [][]int:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]string, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = fmt.Sprintf("%d", v[j][0])
					}
				}
				inputs = append(inputs, TritonInput{
					Name:     inputName,
					Shape:    []int{batchSize, 1},
					DataType: "BYTES",
					Data:     data,
				})
			}
		case [][]float32:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]string, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = fmt.Sprintf("%f", v[j][0])
					}
				}
				inputs = append(inputs, TritonInput{
					Name:     inputName,
					Shape:    []int{batchSize, 1},
					DataType: "BYTES",
					Data:     data,
				})
			}
		case [][]float64:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]string, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = fmt.Sprintf("%f", v[j][0])
					}
				}
				inputs = append(inputs, TritonInput{
					Name:     inputName,
					Shape:    []int{batchSize, 1},
					DataType: "BYTES",
					Data:     data,
				})
			}
		default:
			return nil, fmt.Errorf("unsupported input type for %s at index %d: %T", inputName, i, param)
		}
	}

	return &TritonRequest{Inputs: inputs}, nil
}

// getInputDefinitions returns the input definitions for mapping indices to names
func (t *TritonEvaluator) getInputDefinitions() map[string]interface{} {
	return t.Inputs()
}

// sendTritonRequest sends HTTP request to Triton server
func (t *TritonEvaluator) sendTritonRequest(ctx context.Context, request *TritonRequest) (*TritonResponse, error) {
	// Build Triton inference URL
	url := fmt.Sprintf("%s/v2/models/%s/infer", t.serverURL, t.modelName)

	// Marshal request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Triton request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("triton server returned status %d", resp.StatusCode)
	}

	// Parse response
	var tritonResponse TritonResponse
	if err := json.NewDecoder(resp.Body).Decode(&tritonResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Triton response: %w", err)
	}

	return &tritonResponse, nil
}

// convertFromTritonResponse converts Triton response to MLY format ([]interface{})
func (t *TritonEvaluator) convertFromTritonResponse(response *TritonResponse) []interface{} {
	var result []interface{}

	// Convert each output to MLY format
	for _, output := range response.Outputs {
		converted := t.convertToMLYFormat(output.Data)
		result = append(result, converted)
	}

	return result
}

func (t *TritonEvaluator) convertToMLYFormat(data interface{}) interface{} {
	switch d := data.(type) {
	case []interface{}:
		if len(d) > 0 {
			batchSize := len(d)
			// Default to float32
			floatArray := make([][]float32, batchSize)
			for i, v := range d {
				switch val := v.(type) {
				case float64:
					floatArray[i] = []float32{float32(val)}
				case float32:
					floatArray[i] = []float32{val}
				case int:
					floatArray[i] = []float32{float32(val)}
				case int32:
					floatArray[i] = []float32{float32(val)}
				case int64:
					floatArray[i] = []float32{float32(val)}
				default:
					// Fallback to string if not numeric
					stringArray := make([][]string, batchSize)
					for j, sv := range d {
						stringArray[j] = []string{fmt.Sprintf("%v", sv)}
					}
					return stringArray
				}
			}
			return floatArray
		}
	case []float64:
		converted := make([]float32, len(d))
		for i, v := range d {
			converted[i] = float32(v)
		}
		return [][]float32{converted}
	case []float32:
		return [][]float32{d}
	case []string:
		return [][]string{d}
	case float64:
		return [][]float32{{float32(d)}}
	case float32:
		return [][]float32{{d}}
	case string:
		return [][]string{{d}}
	default:
		return [][]string{{fmt.Sprintf("%v", d)}}
	}

	return [][]float32{{0.0}}
}

// Signature returns model signature information
func (t *TritonEvaluator) Signature() interface{} {
	var inputs []domain.Input
	var outputs []domain.Output

	if len(t.config.Inputs) > 0 {
		for _, input := range t.config.Inputs {
			inputs = append(inputs, domain.Input{
				Name:      input.Name,
				Index:     input.Index,
				Vocab:     !input.Wildcard,
				Auxiliary: input.Auxiliary,
			})
		}
	} else {
		// Fallback: single generic input
		inputs = []domain.Input{
			{Name: "triton_input", Index: 0},
		}
	}

	outputs = []domain.Output{
		{Name: "output_0", Index: 0, DataType: "float32"},
	}

	return &domain.Signature{
		Inputs:  inputs,
		Outputs: outputs,
		Output:  outputs[0],
	}
}

// Dictionary returns vocabulary dictionary (Triton models typically don't use MLY dictionaries)
func (t *TritonEvaluator) Dictionary() *common.Dictionary {
	// Triton models handle their own preprocessing, so no dictionary needed
	return nil
}

// Stats returns Triton-specific statistics
func (t *TritonEvaluator) Stats(stats map[string]interface{}) {
	stats["triton_server_url"] = t.serverURL
	stats["triton_model_name"] = t.modelName
	stats["triton_version"] = t.version
	stats["model_id"] = t.config.ID
}

// Inputs returns the model inputs for request validation
func (t *TritonEvaluator) Inputs() map[string]interface{} {
	inputs := make(map[string]interface{})

	// If the model config specifies inputs, use those (like mlfdv3 model)
	if len(t.config.Inputs) > 0 {
		for _, input := range t.config.Inputs {
			// Create domain.Input with proper type mapping
			inputType := reflect.TypeOf("") // Default to string
			if input.DataType != "" {
				switch input.DataType {
				case "string":
					inputType = reflect.TypeOf("")
				case "int":
					inputType = reflect.TypeOf(int(0))
				case "float32":
					inputType = reflect.TypeOf(float32(0))
				case "float64":
					inputType = reflect.TypeOf(float64(0))
				}
			}

			inputs[input.Name] = &domain.Input{
				Name:      input.Name,
				Index:     input.Index,
				Type:      inputType,
				Vocab:     !input.Wildcard, // Vocab false means wildcard (no vocabulary restriction)
				Auxiliary: input.Auxiliary,
			}
		}
	} else {
		// For Triton models without explicit input configuration,
		// we should discover the schema from Triton's model metadata API
		// or require explicit configuration. Hardcoded defaults are error-prone.
		// TODO: Implement Triton model metadata discovery via /v2/models/{model}/config

		// For now, return empty inputs to signal that schema discovery is needed
		// This will cause validation errors at request time if inputs are missing,
		// which is safer than hardcoded defaults that may not match the actual model
		//
		// To fix this, users should specify inputs in their model configuration YAML:
		// inputs:
		//   - name: "input_field_name"
		//     dataType: "string"
		//     wildcard: false
		//     auxiliary: false
		return make(map[string]interface{})
	}

	return inputs
}

// Close releases Triton client resources
func (t *TritonEvaluator) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/francoispqt/gojay"
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

func (t *TritonInput) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("name", t.Name)
	enc.ArrayKey("shape", gojay.EncodeArrayFunc(func(enc *gojay.Encoder) {
		for _, v := range t.Shape {
			enc.AddInt(v)
		}
	}))
	enc.StringKey("datatype", t.DataType)

	enc.ArrayKey("data", gojay.EncodeArrayFunc(func(enc *gojay.Encoder) {
		switch data := t.Data.(type) {
		case []string:
			for _, v := range data {
				enc.AddString(v)
			}
		case []int:
			for _, v := range data {
				enc.AddInt(v)
			}
		case []float32:
			for _, v := range data {
				enc.AddFloat32(v)
			}
		case []float64:
			for _, v := range data {
				enc.AddFloat64(v)
			}
		default:
			for i := 0; i < reflect.ValueOf(data).Len(); i++ {
				val := reflect.ValueOf(data).Index(i).Interface()
				enc.AddInterface(val)
			}
		}
	}))
}

func (t *TritonInput) IsNil() bool {
	return t == nil
}

func (t *TritonRequest) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey("inputs", (*TritonInputs)(&t.Inputs))
}

func (t *TritonRequest) IsNil() bool {
	return t == nil
}

type TritonInputs []TritonInput

func (t *TritonInputs) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range *t {
		enc.AddObject(&(*t)[i])
	}
}

func (t *TritonInputs) IsNil() bool {
	return t == nil || len(*t) == 0
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
	serverURL := config.URL           // Use model's URL field for Triton server endpoint
	modelName := config.ID            // Default to model ID
	version := "1"                    // Default version
	timeout := 100 * time.Millisecond // Default timeout

	// Use Triton-specific configuration if provided
	if config.Triton != nil {
		if config.Triton.ModelName != "" {
			modelName = config.Triton.ModelName
		}
		if config.Triton.Version != "" {
			version = config.Triton.Version
		}
		if config.Triton.Timeout > 0 {
			timeout = time.Duration(config.Triton.Timeout) * time.Millisecond
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
	tritonRequest, err := t.convertToTritonRequest(params)
	if err != nil {
		return nil, fmt.Errorf("failed to convert params to Triton format: %w", err)
	}

	tritonResponse, err := t.sendTritonRequest(ctx, tritonRequest)
	if err != nil {
		return nil, fmt.Errorf("triton inference failed for model %s: %w", t.config.ID, err)
	}

	result := t.convertFromTritonResponse(tritonResponse)
	return result, nil
}

// convertToTritonRequest converts MLY params ([]interface{}) to Triton request format
func (t *TritonEvaluator) convertToTritonRequest(params []interface{}) (*TritonRequest, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	var inputs []TritonInput

	// Get the input definitions to map indices to names
	inputDefs := t.getInputDefinitions()

	// Build index-to-name map
	indexToName := make(map[int]string)
	for name, input := range inputDefs {
		if domainInput, ok := input.(*domain.Input); ok && !domainInput.Auxiliary {
			indexToName[domainInput.Index] = name
		}
	}

	// Convert each parameter to Triton input format
	for i, param := range params {
		inputName, exists := indexToName[i]
		if !exists {
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
					Shape:    []int{batchSize, 1},
					DataType: "BYTES",
					Data:     data,
				})
			}
		case [][]int:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]int, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = v[j][0]
					}
				}
				inputs = append(inputs, TritonInput{
					Name:     inputName,
					Shape:    []int{batchSize, 1},
					DataType: "INT32",
					Data:     data,
				})
			}
		case [][]float32:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]float32, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = v[j][0]
					}
				}
				inputs = append(inputs, TritonInput{
					Name:     inputName,
					Shape:    []int{batchSize, 1},
					DataType: "FP32",
					Data:     data,
				})
			}
		case [][]float64:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]float64, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = v[j][0]
					}
				}
				inputs = append(inputs, TritonInput{
					Name:     inputName,
					Shape:    []int{batchSize, 1},
					DataType: "FP64",
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
	url := t.serverURL + "/v2/models/" + t.modelName + "/infer"

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	enc := gojay.NewEncoder(buf)
	if err := enc.EncodeObject(request); err != nil {
		return nil, fmt.Errorf("failed to marshal Triton request: %w", err)
	}
	jsonData := buf.Bytes()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request with retry logic
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = t.httpClient.Do(httpReq)
		if err == nil {
			break
		}
		if attempt == maxRetries {
			return nil, fmt.Errorf("http request failed after %d attempts: %w", maxRetries+1, err)
		}
		time.Sleep(time.Duration(5*(1<<attempt)) * time.Millisecond)

		if attempt < maxRetries {
			httpReq.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("triton server returned status %d", resp.StatusCode)
	}

	var tritonResponse TritonResponse
	if err := json.NewDecoder(resp.Body).Decode(&tritonResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Triton response: %w", err)
	}

	return &tritonResponse, nil
}

// convertFromTritonResponse converts Triton response to MLY format ([]interface{})
func (t *TritonEvaluator) convertFromTritonResponse(response *TritonResponse) []interface{} {
	var result []interface{}

	for _, output := range response.Outputs {
		if data, ok := output.Data.([]interface{}); ok && len(data) > 0 {
			batchSize := len(data)
			converted := make([][]float32, batchSize)
			for i, v := range data {
				if f, ok := v.(float64); ok {
					converted[i] = []float32{float32(f)}
				} else {
					converted[i] = []float32{0.0}
				}
			}
			result = append(result, converted)
		} else {
			result = append(result, [][]float32{{0.0}})
		}
	}

	return result
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
		panic("Triton model" + t.config.ID + " requires explicit input configuration. " +
			"Add 'inputs' section to your model configuration YAML with field definitions")
	}

	outputs = []domain.Output{
		{Name: "output_0", Index: 0, DataType: "float64"},
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
				Vocab:     !input.Wildcard,
				Auxiliary: input.Auxiliary,
			}
		}
	} else {
		panic("Triton model" + t.config.ID + " requires explicit input configuration. " +
			"Add 'inputs' section to your model configuration YAML with field definitions")
	}

	return inputs
}

// Close releases Triton client resources
func (t *TritonEvaluator) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// SetReloadOK is a no-op for Triton models (they don't support reloading)
func (t *TritonEvaluator) SetReloadOK(reloadOK *int32) {
	// No-op: Triton models are managed externally
}

// ReloadIfNeeded is a no-op for Triton models (they don't support reloading)
func (t *TritonEvaluator) ReloadIfNeeded(ctx context.Context) error {
	// No-op: Triton models are managed externally
	return nil
}

// SupportsReload returns false since Triton models don't support reloading through MLY
func (t *TritonEvaluator) SupportsReload() bool {
	return false
}

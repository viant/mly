package triton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/francoispqt/gojay"
)

// Deprecated: use GRPCClient instead
type HTTPClient struct {
	httpClient *http.Client
	serverURL  string

	debug bool
}

func (c *HTTPClient) sendRequestCheckStatus(ctx context.Context, method, path string) (*http.Response, error) {
	if c.debug {
		log.Printf("Sending request %s %s\n", method, path)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, c.serverURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.handleRequestWithRetry(ctx, httpReq, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("triton server http status code: %d for %s %s", resp.StatusCode, method, path)
	}

	return resp, nil
}

func (c *HTTPClient) ServerReady(ctx context.Context) error {
	path := "/v2/health/ready"
	_, err := c.sendRequestCheckStatus(ctx, "GET", path)
	return err
}

func (c *HTTPClient) ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) ([]interface{}, error) {
	tritonRequest, err := convertToTritonRequest(inputs, indexToName)
	if err != nil {
		return nil, err
	}

	tritonResponse, err := c.sendRequest(ctx, modelName, tritonRequest)
	if err != nil {
		return nil, err
	}

	return convertFromTritonResponse(tritonResponse)
}

func (c *HTTPClient) ModelReady(ctx context.Context, modelName string) (bool, error) {
	path := "/v2/models/" + modelName + "/ready"
	resp, err := c.sendRequestCheckStatus(ctx, "GET", path)
	if err != nil {
		return false, err
	}

	if resp.StatusCode == http.StatusBadRequest {
		// TODO "Model version not ready"
		return false, fmt.Errorf("triton server returned status %d", resp.StatusCode)
	} else if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("triton server returned status %d: %s", resp.StatusCode, string(body))
	}

	return true, nil
}

func (c *HTTPClient) ModelLoad(ctx context.Context, modelName string) error {
	path := "/v2/repository/models/" + modelName + "/load"
	_, err := c.sendRequestCheckStatus(ctx, "POST", path)
	return err
}

func (c *HTTPClient) ModelUnload(ctx context.Context, modelName string) error {
	path := "/v2/repository/models/" + modelName + "/unload"
	_, err := c.sendRequestCheckStatus(ctx, "POST", path)
	return err
}

func (c *HTTPClient) Close() error {
	return nil
}

// TritonInput represents a single input tensor for Triton
type TritonInput struct {
	Name     string      `json:"name"`
	Shape    []int       `json:"shape"`
	DataType string      `json:"datatype"`
	Data     interface{} `json:"data"`
}

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

func (c *HTTPClient) sendRequest(ctx context.Context, modelName string, request *TritonRequest) (*TritonResponse, error) {
	url := c.serverURL + "/v2/models/" + modelName + "/infer"

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

	resp, err := c.handleRequestWithRetry(ctx, httpReq, jsonData)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("triton server returned status %d: %s", resp.StatusCode, string(body))
	}

	var tritonResponse TritonResponse
	if err := json.NewDecoder(resp.Body).Decode(&tritonResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Triton response: %w", err)
	}

	return &tritonResponse, nil
}

func (c *HTTPClient) handleRequestWithRetry(ctx context.Context, httpReq *http.Request, jsonData []byte) (*http.Response, error) {
	var resp *http.Response
	var err error

	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = c.httpClient.Do(httpReq)

		if err == nil {
			break
		} else if resp == nil {
			// do nothing
		} else if resp.StatusCode >= 500 {
			// try again on 5xx
			resp.Body.Close()
		} else {
			break
		}

		if attempt == maxRetries {
			return nil, fmt.Errorf("http request failed after %d attempts: %w", maxRetries+1, err)
		}

		time.Sleep(time.Duration(5*(1<<attempt)) * time.Millisecond)

		if attempt < maxRetries {
			httpReq.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return resp, nil
}

// convertToTritonRequest converts Feeds ([numInputs]([batchSize][1]T)) to Triton request format
func convertToTritonRequest(params []interface{}, indexToName map[int]string) (*TritonRequest, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	var inputs []TritonInput

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

// convertFromTritonResponse converts Triton response to [numOutputs]([batchSize]D_T) - shape depends on model output.
func convertFromTritonResponse(response *TritonResponse) ([]interface{}, error) {
	var result []interface{}

	for outputOffset, output := range response.Outputs {
		if data, ok := output.Data.([]interface{}); ok && len(data) > 0 {
			batchSize := len(data)
			converted := make([][]float32, batchSize)
			for i, v := range data {
				if f, ok := v.(float64); ok {
					converted[i] = []float32{float32(f)}
				} else {
					return nil, fmt.Errorf("unsupported output type for %s: %T, for batch item %d of output offset %d", output.Name, v, i, outputOffset)
				}
			}
			result = append(result, converted)
		} else {
			return nil, fmt.Errorf("unsupported output type for %s: %T, for output offset %d", output.Name, output.Data, outputOffset)
		}
	}

	return result, nil
}

package platform

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	triton "github.com/viant/mly/proto/triton"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// preparedInput represents processed input data ready for gRPC transport
type preparedInput struct {
	name     string
	datatype string      // Triton datatype: "BYTES", "INT32", "INT64", "FP32", "FP64"
	shape    []int64     // Shape in int64 for gRPC compatibility
	data     interface{} // Flattened data: []string, []int32, []int64, []float32, []float64
}

// TritonEvaluator implements PlatformEvaluator for Triton Inference Server via gRPC
type TritonEvaluator struct {
	config    *config.Model
	serverURL string
	modelName string

	// gRPC-specific fields
	grpcConn   *grpc.ClientConn
	grpcClient triton.GRPCInferenceServiceClient
	timeout    time.Duration

	signature *domain.Signature
	inputs    map[string]*domain.Input

	// Health reporting support
	healthPtr       *int32
	stopHealthCheck chan struct{}
	initMonitorOnce sync.Once
}

// NewTritonEvaluator creates a new Triton evaluator
func NewTritonEvaluator(config *config.Model) *TritonEvaluator {
	serverURL := config.URL
	modelName := config.ID
	timeout := 100 * time.Millisecond

	if config.Triton != nil {
		if config.Triton.ModelName != "" {
			modelName = config.Triton.ModelName
		}
		if config.Triton.Timeout > 0 {
			timeout = time.Duration(config.Triton.Timeout) * time.Millisecond
		}
	}

	grpcAddr := parseGRPCAddress(serverURL)

	conn, err := grpc.NewClient(grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Triton gRPC client at %s: %v", grpcAddr, err))
	}

	evaluator := &TritonEvaluator{
		config:          config,
		serverURL:       grpcAddr,
		modelName:       modelName,
		grpcConn:        conn,
		grpcClient:      triton.NewGRPCInferenceServiceClient(conn),
		timeout:         timeout,
		stopHealthCheck: make(chan struct{}),
	}

	evaluator.signature = evaluator.computeSignature()
	evaluator.inputs = evaluator.computeInputs()

	return evaluator
}

func parseGRPCAddress(url string) string {
	addr := strings.TrimPrefix(url, "http://")
	addr = strings.TrimPrefix(addr, "https://")

	if !strings.Contains(addr, ":") {
		addr += ":8001"
	}

	return addr
}

// Predict performs inference via Triton Inference Server
func (t *TritonEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	preparedInputs, err := t.prepareBatchInputs(params)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare inputs: %w", err)
	}

	grpcRequest, err := t.buildGRPCRequest(preparedInputs)
	if err != nil {
		return nil, fmt.Errorf("failed to build gRPC request: %w", err)
	}

	requestCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	grpcResponse, err := t.grpcClient.ModelInfer(requestCtx, grpcRequest)
	if err != nil {
		return nil, fmt.Errorf("triton gRPC inference failed for model %s: %w", t.config.ID, err)
	}

	result, err := t.convertGRPCResponse(grpcResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert gRPC response: %w", err)
	}

	return result, nil
}

func (t *TritonEvaluator) prepareBatchInputs(params []interface{}) ([]preparedInput, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	var inputs []preparedInput

	inputDefs := t.Inputs()

	indexToName := make(map[int]string)
	for name, input := range inputDefs {
		if !input.Auxiliary {
			indexToName[input.Index] = name
		}
	}

	for i, param := range params {
		inputName, exists := indexToName[i]
		if !exists {
			return nil, fmt.Errorf("no input name found for index %d", i)
		}

		switch v := param.(type) {
		case [][]string:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]string, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = v[j][0]
					}
				}
				inputs = append(inputs, preparedInput{
					name:     inputName,
					shape:    []int64{int64(batchSize), 1},
					datatype: "BYTES",
					data:     data,
				})
			}
		case [][]int:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]int32, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = int32(v[j][0])
					}
				}
				inputs = append(inputs, preparedInput{
					name:     inputName,
					shape:    []int64{int64(batchSize), 1},
					datatype: "INT32",
					data:     data,
				})
			}
		case [][]int32:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]int32, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = v[j][0]
					}
				}
				inputs = append(inputs, preparedInput{
					name:     inputName,
					shape:    []int64{int64(batchSize), 1},
					datatype: "INT32",
					data:     data,
				})
			}
		case [][]int64:
			if len(v) > 0 {
				batchSize := len(v)
				data := make([]int64, batchSize)
				for j := 0; j < batchSize; j++ {
					if len(v[j]) > 0 {
						data[j] = v[j][0]
					}
				}
				inputs = append(inputs, preparedInput{
					name:     inputName,
					shape:    []int64{int64(batchSize), 1},
					datatype: "INT64",
					data:     data,
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
				inputs = append(inputs, preparedInput{
					name:     inputName,
					shape:    []int64{int64(batchSize), 1},
					datatype: "FP32",
					data:     data,
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
				inputs = append(inputs, preparedInput{
					name:     inputName,
					shape:    []int64{int64(batchSize), 1},
					datatype: "FP64",
					data:     data,
				})
			}
		default:
			return nil, fmt.Errorf("unsupported input type for %s at index %d: %T", inputName, i, param)
		}
	}

	return inputs, nil
}

func (t *TritonEvaluator) buildGRPCRequest(preparedInputs []preparedInput) (*triton.ModelInferRequest, error) {
	req := &triton.ModelInferRequest{
		ModelName: t.modelName,
		Inputs:    make([]*triton.ModelInferRequest_InferInputTensor, len(preparedInputs)),
	}

	for i, input := range preparedInputs {
		tensor := &triton.ModelInferRequest_InferInputTensor{
			Name:     input.name,
			Datatype: input.datatype,
			Shape:    input.shape,
			Contents: &triton.InferTensorContents{},
		}

		switch data := input.data.(type) {
		case []string:
			tensor.Contents.BytesContents = make([][]byte, len(data))
			for j, s := range data {
				tensor.Contents.BytesContents[j] = []byte(s)
			}
		case []int32:
			tensor.Contents.IntContents = data
		case []int64:
			tensor.Contents.Int64Contents = data
		case []float32:
			tensor.Contents.Fp32Contents = data
		case []float64:
			tensor.Contents.Fp64Contents = data
		default:
			return nil, fmt.Errorf("unsupported input data type %T for %s", data, input.name)
		}

		req.Inputs[i] = tensor
	}

	return req, nil
}

func (t *TritonEvaluator) convertGRPCResponse(response *triton.ModelInferResponse) ([]interface{}, error) {
	if len(response.Outputs) == 0 {
		return nil, fmt.Errorf("no outputs in response")
	}

	result := make([]interface{}, len(response.Outputs))
	useRawContents := len(response.RawOutputContents) > 0

	for i, output := range response.Outputs {
		batchSize := 1
		if len(output.Shape) > 0 {
			batchSize = int(output.Shape[0])
		}

		if useRawContents {
			if i >= len(response.RawOutputContents) {
				return nil, fmt.Errorf("raw output contents missing for output %d", i)
			}
			rawData := response.RawOutputContents[i]
			parsedData, err := t.parseRawOutput(rawData, output.Datatype, batchSize)
			if err != nil {
				return nil, fmt.Errorf("failed to parse raw output %s: %w", output.Name, err)
			}
			result[i] = parsedData
			continue
		}

		if output.Contents == nil {
			return nil, fmt.Errorf("output %s missing contents", output.Name)
		}

		switch output.Datatype {
		case "FP32":
			if len(output.Contents.Fp32Contents) == 0 {
				converted := make([][]float32, batchSize)
				for j := 0; j < batchSize; j++ {
					converted[j] = []float32{0.0}
				}
				result[i] = converted
			} else {
				converted := make([][]float32, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.Fp32Contents); j++ {
					converted[j] = []float32{output.Contents.Fp32Contents[j]}
				}
				result[i] = converted
			}

		case "FP64":
			if len(output.Contents.Fp64Contents) == 0 {
				converted := make([][]float64, batchSize)
				for j := 0; j < batchSize; j++ {
					converted[j] = []float64{0.0}
				}
				result[i] = converted
			} else {
				converted := make([][]float64, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.Fp64Contents); j++ {
					converted[j] = []float64{output.Contents.Fp64Contents[j]}
				}
				result[i] = converted
			}

		case "INT32":
			if len(output.Contents.IntContents) == 0 {
				converted := make([][]int32, batchSize)
				for j := 0; j < batchSize; j++ {
					converted[j] = []int32{0}
				}
				result[i] = converted
			} else {
				converted := make([][]int32, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.IntContents); j++ {
					converted[j] = []int32{output.Contents.IntContents[j]}
				}
				result[i] = converted
			}

		case "INT64":
			if len(output.Contents.Int64Contents) == 0 {
				converted := make([][]int64, batchSize)
				for j := 0; j < batchSize; j++ {
					converted[j] = []int64{0}
				}
				result[i] = converted
			} else {
				converted := make([][]int64, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.Int64Contents); j++ {
					converted[j] = []int64{output.Contents.Int64Contents[j]}
				}
				result[i] = converted
			}

		case "BYTES":
			if len(output.Contents.BytesContents) == 0 {
				converted := make([][]string, batchSize)
				for j := 0; j < batchSize; j++ {
					converted[j] = []string{""}
				}
				result[i] = converted
			} else {
				converted := make([][]string, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.BytesContents); j++ {
					converted[j] = []string{string(output.Contents.BytesContents[j])}
				}
				result[i] = converted
			}

		default:
			if len(output.Contents.Fp32Contents) > 0 {
				converted := make([][]float32, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.Fp32Contents); j++ {
					converted[j] = []float32{output.Contents.Fp32Contents[j]}
				}
				result[i] = converted
			} else {
				return nil, fmt.Errorf("unsupported output datatype %s for %s", output.Datatype, output.Name)
			}
		}
	}

	return result, nil
}

func (t *TritonEvaluator) parseRawOutput(rawData []byte, datatype string, batchSize int) (interface{}, error) {
	switch datatype {
	case "INT64":
		if len(rawData) != batchSize*8 {
			return nil, fmt.Errorf("INT64 raw data size mismatch: got %d bytes, expected %d", len(rawData), batchSize*8)
		}
		converted := make([][]int64, batchSize)
		for j := 0; j < batchSize; j++ {
			value := int64(binary.LittleEndian.Uint64(rawData[j*8 : (j+1)*8]))
			converted[j] = []int64{value}
		}
		return converted, nil

	case "INT32":
		if len(rawData) != batchSize*4 {
			return nil, fmt.Errorf("INT32 raw data size mismatch: got %d bytes, expected %d", len(rawData), batchSize*4)
		}
		converted := make([][]int32, batchSize)
		for j := 0; j < batchSize; j++ {
			value := int32(binary.LittleEndian.Uint32(rawData[j*4 : (j+1)*4]))
			converted[j] = []int32{value}
		}
		return converted, nil

	case "FP32":
		if len(rawData) != batchSize*4 {
			return nil, fmt.Errorf("FP32 raw data size mismatch: got %d bytes, expected %d", len(rawData), batchSize*4)
		}
		converted := make([][]float32, batchSize)
		for j := 0; j < batchSize; j++ {
			bits := binary.LittleEndian.Uint32(rawData[j*4 : (j+1)*4])
			value := math.Float32frombits(bits)
			converted[j] = []float32{value}
		}
		return converted, nil

	case "FP64":
		if len(rawData) != batchSize*8 {
			return nil, fmt.Errorf("FP64 raw data size mismatch: got %d bytes, expected %d", len(rawData), batchSize*8)
		}
		converted := make([][]float64, batchSize)
		for j := 0; j < batchSize; j++ {
			bits := binary.LittleEndian.Uint64(rawData[j*8 : (j+1)*8])
			value := math.Float64frombits(bits)
			converted[j] = []float64{value}
		}
		return converted, nil

	case "BYTES":
		converted := make([][]string, batchSize)
		offset := 0
		for j := 0; j < batchSize; j++ {
			if offset+4 > len(rawData) {
				return nil, fmt.Errorf("BYTES raw data truncated at element %d", j)
			}
			length := int(binary.LittleEndian.Uint32(rawData[offset : offset+4]))
			offset += 4
			if offset+length > len(rawData) {
				return nil, fmt.Errorf("BYTES raw data truncated at element %d: expected %d bytes", j, length)
			}
			value := string(rawData[offset : offset+length])
			offset += length
			converted[j] = []string{value}
		}
		return converted, nil

	default:
		return nil, fmt.Errorf("unsupported datatype for raw output: %s", datatype)
	}
}

func (t *TritonEvaluator) computeSignature() *domain.Signature {
	var inputs []domain.Input
	var outputs []domain.Output

	if len(t.config.Inputs) > 0 {
		for _, input := range t.config.Inputs {
			if !input.Auxiliary {
				inputs = append(inputs, domain.Input{
					Name:  input.Name,
					Index: input.Index,
				})
			}
		}
	} else {
		panic("Triton model " + t.config.ID + " requires explicit input configuration. " +
			"Add 'inputs' section to your model configuration YAML with field definitions")
	}

	if len(t.config.Outputs) > 0 {
		for i, output := range t.config.Outputs {
			outputs = append(outputs, domain.Output{
				Name:     output.Name,
				Index:    i,
				DataType: output.DataType,
			})
		}
	} else {
		panic("Triton model " + t.config.ID + " requires explicit output configuration. " +
			"Add 'outputs' section to your model configuration YAML with field definitions")
	}

	return &domain.Signature{
		Inputs:  inputs,
		Outputs: outputs,
		Output:  outputs[0],
	}
}

func (t *TritonEvaluator) Signature() *domain.Signature {
	return t.signature
}

func (t *TritonEvaluator) Dictionary() *common.Dictionary {
	return nil
}

func (t *TritonEvaluator) Stats(stats map[string]interface{}) {
	stats["triton_server_url"] = t.serverURL
	stats["triton_model_name"] = t.modelName
	stats["model_id"] = t.config.ID
}

func (t *TritonEvaluator) computeInputs() map[string]*domain.Input {
	inputs := make(map[string]*domain.Input)

	if len(t.config.Inputs) > 0 {
		for _, input := range t.config.Inputs {
			inputType := reflect.TypeOf("")
			if input.DataType != "" {
				switch input.DataType {
				case "string":
					inputType = reflect.TypeOf("")
				case "int":
					inputType = reflect.TypeOf(0)
				case "int32":
					inputType = reflect.TypeOf(int32(0))
				case "int64":
					inputType = reflect.TypeOf(int64(0))
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
				Vocab:     false,
				Auxiliary: input.Auxiliary,
			}
		}
	} else {
		panic("Triton model " + t.config.ID + " requires explicit input configuration. " +
			"Add 'inputs' section to your model configuration YAML with field definitions")
	}

	return inputs
}

func (t *TritonEvaluator) Inputs() map[string]*domain.Input {
	return t.inputs
}

func (t *TritonEvaluator) IsHealthy() bool {
	if t.healthPtr == nil {
		return false
	}
	return atomic.LoadInt32(t.healthPtr) == 1
}

func (t *TritonEvaluator) SetHealthStatus(healthPtr *int32) {
	t.healthPtr = healthPtr
	if healthPtr != nil {
		atomic.StoreInt32(healthPtr, 0)
		t.initMonitorOnce.Do(func() {
			go t.backgroundHealthMonitor()
		})
	}
}

func (t *TritonEvaluator) SupportsHealthReporting() bool {
	return true
}

func (t *TritonEvaluator) checkTritonModelHealth() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &triton.ModelReadyRequest{
		Name:    t.modelName,
		Version: "",
	}

	resp, err := t.grpcClient.ModelReady(ctx, req)
	if err != nil {
		return false
	}

	return resp.GetReady()
}

func (t *TritonEvaluator) backgroundHealthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if t.healthPtr == nil {
				return
			}

			if t.checkTritonModelHealth() {
				atomic.StoreInt32(t.healthPtr, 1)
			} else {
				atomic.StoreInt32(t.healthPtr, 0)
			}

		case <-t.stopHealthCheck:
			return
		}
	}
}

// Close releases Triton client resources and stops health monitoring
func (t *TritonEvaluator) Close() error {
	select {
	case t.stopHealthCheck <- struct{}{}:
	default:
	}

	if t.grpcConn != nil {
		return t.grpcConn.Close()
	}
	return nil
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

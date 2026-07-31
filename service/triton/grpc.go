package triton

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	triton "github.com/viant/mly/proto/triton"
	"google.golang.org/grpc"
)

type GRPCClient struct {
	// Note that this is dangerous to Close if the connection is shared.
	// See how TritonEvaluator handles Close().
	grpcConn   *grpc.ClientConn
	grpcClient triton.GRPCInferenceServiceClient
}

func NewGRPCClient(grpcConn *grpc.ClientConn) *GRPCClient {
	return &GRPCClient{
		grpcConn:   grpcConn,
		grpcClient: triton.NewGRPCInferenceServiceClient(grpcConn),
	}
}

func (c *GRPCClient) ServerReady(ctx context.Context) error {
	_, err := c.grpcClient.ServerReady(ctx, &triton.ServerReadyRequest{})
	return err
}

func (c *GRPCClient) ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) (map[string]interface{}, error) {
	grpcRequest, err := toGRPCRequest(modelName, inputs, indexToName)
	if err != nil {
		return nil, err
	}

	grpcResponse, err := c.grpcClient.ModelInfer(ctx, grpcRequest)
	if err != nil {
		return nil, err
	}

	return convertGRPCResponse(grpcResponse)
}

func (c *GRPCClient) ModelReady(ctx context.Context, modelName string) (bool, error) {
	grpcResponse, err := c.grpcClient.ModelReady(ctx, &triton.ModelReadyRequest{
		Name: modelName,
	})
	if err != nil {
		return false, err
	}
	return grpcResponse.Ready, nil
}

func (c *GRPCClient) ModelLoad(ctx context.Context, modelName string) error {
	_, err := c.grpcClient.RepositoryModelLoad(ctx, &triton.RepositoryModelLoadRequest{
		ModelName: modelName,
	})

	if err != nil {
		return err
	}
	return nil
}

func (c *GRPCClient) ModelUnload(ctx context.Context, modelName string) error {
	_, err := c.grpcClient.RepositoryModelUnload(ctx, &triton.RepositoryModelUnloadRequest{
		ModelName: modelName,
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *GRPCClient) ModelMetadata(ctx context.Context, modelName string) (*ModelMetadata, error) {
	grpcResponse, err := c.grpcClient.ModelMetadata(ctx, &triton.ModelMetadataRequest{
		Name: modelName,
	})
	if err != nil {
		return nil, err
	}
	return convertGRPCModelMetadataResponse(grpcResponse), nil
}

func convertGRPCModelMetadataResponse(response *triton.ModelMetadataResponse) *ModelMetadata {
	inputs := make([]MetadataTensor, len(response.Inputs))
	for i, input := range response.Inputs {
		inputs[i] = MetadataTensor{
			Name:     input.Name,
			Datatype: input.Datatype,
			Shape:    input.Shape,
		}
	}

	outputs := make([]MetadataTensor, len(response.Outputs))
	for i, output := range response.Outputs {
		outputs[i] = MetadataTensor{
			Name:     output.Name,
			Datatype: output.Datatype,
			Shape:    output.Shape,
		}
	}

	return &ModelMetadata{
		Inputs:  inputs,
		Outputs: outputs,
	}
}

func (c *GRPCClient) Close() error {
	return c.grpcConn.Close()
}

func toGRPCRequest(modelName string, params []interface{}, indexToName map[int]string) (*triton.ModelInferRequest, error) {
	req := &triton.ModelInferRequest{
		ModelName: modelName,
		Inputs:    make([]*triton.ModelInferRequest_InferInputTensor, len(params)),
	}

	for i, param := range params {
		inputName, exists := indexToName[i]
		if !exists {
			return nil, fmt.Errorf("no input name found for index %d", i)
		}

		inputContents := &triton.InferTensorContents{}

		inputTensor := &triton.ModelInferRequest_InferInputTensor{
			Name:     inputName,
			Contents: inputContents,
		}

		var batchSize int
		var datatype string

		switch v := param.(type) {
		case [][]string:
			if len(v) > 0 {
				batchSize = len(v)
				datatype = "BYTES"

				inputContents.BytesContents = make([][]byte, batchSize)
				for j := range batchSize {
					inputContents.BytesContents[j] = []byte(v[j][0])
				}
			}
		case [][]int:
			if len(v) > 0 {
				batchSize = len(v)
				datatype = "INT32"

				inputContents.IntContents = make([]int32, batchSize)
				for j := range batchSize {
					inputContents.IntContents[j] = int32(v[j][0])
				}
			}
		case [][]int32:
			if len(v) > 0 {
				batchSize = len(v)
				datatype = "INT32"

				inputContents.IntContents = make([]int32, batchSize)
				for j := range batchSize {
					inputContents.IntContents[j] = v[j][0]
				}

			}
		case [][]int64:
			if len(v) > 0 {
				batchSize = len(v)
				datatype = "INT64"

				inputContents.Int64Contents = make([]int64, batchSize)
				for j := range batchSize {
					inputContents.Int64Contents[j] = v[j][0]
				}

			}
		case [][]float32:
			if len(v) > 0 {
				batchSize = len(v)
				datatype = "FP32"

				inputContents.Fp32Contents = make([]float32, batchSize)
				for j := range batchSize {
					inputContents.Fp32Contents[j] = v[j][0]
				}
			}
		case [][]float64:
			if len(v) > 0 {
				batchSize = len(v)
				datatype = "FP64"

				inputContents.Fp64Contents = make([]float64, batchSize)
				for j := range batchSize {
					inputContents.Fp64Contents[j] = v[j][0]
				}
			}
		default:
			return nil, fmt.Errorf("unsupported input type for %s at index %d: %T", inputName, i, param)
		}

		inputTensor.Datatype = datatype
		inputTensor.Shape = []int64{int64(batchSize), 1}

		req.Inputs[i] = inputTensor
	}

	return req, nil
}

// parseRawOutput if output is provided in raw format.
func parseRawOutput(rawData []byte, datatype string, batchSize int) (interface{}, error) {
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

// convertGRPCResponse maps the response tensors into a map keyed by output name.
// Ordering into signature order is the evaluator's responsibility; keying by name
// here removes any reliance on the order Triton returns tensors in.
func convertGRPCResponse(response *triton.ModelInferResponse) (map[string]interface{}, error) {
	if len(response.Outputs) == 0 {
		return nil, fmt.Errorf("no outputs in response")
	}

	result := make(map[string]interface{}, len(response.Outputs))
	useRawContents := len(response.RawOutputContents) > 0

	for i, output := range response.Outputs {
		batchSize := 1
		if len(output.Shape) > 0 {
			batchSize = int(output.Shape[0])
		}

		var parsed interface{}

		if useRawContents {
			if i >= len(response.RawOutputContents) {
				return nil, fmt.Errorf("raw output contents missing for output %d", i)
			}
			rawData := response.RawOutputContents[i]
			parsedData, err := parseRawOutput(rawData, output.Datatype, batchSize)
			if err != nil {
				return nil, fmt.Errorf("failed to parse raw output %s: %w", output.Name, err)
			}
			parsed = parsedData
		} else {
			if output.Contents == nil {
				return nil, fmt.Errorf("output %s missing contents", output.Name)
			}

			switch output.Datatype {
			case "FP32":
				converted := make([][]float32, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.Fp32Contents); j++ {
					converted[j] = []float32{output.Contents.Fp32Contents[j]}
				}
				parsed = converted
			case "FP64":
				converted := make([][]float64, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.Fp64Contents); j++ {
					converted[j] = []float64{output.Contents.Fp64Contents[j]}
				}
				parsed = converted

			case "INT32":
				converted := make([][]int32, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.IntContents); j++ {
					converted[j] = []int32{output.Contents.IntContents[j]}
				}
				parsed = converted
			case "INT64":
				converted := make([][]int64, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.Int64Contents); j++ {
					converted[j] = []int64{output.Contents.Int64Contents[j]}
				}
				parsed = converted

			case "BYTES":
				converted := make([][]string, batchSize)
				for j := 0; j < batchSize && j < len(output.Contents.BytesContents); j++ {
					converted[j] = []string{string(output.Contents.BytesContents[j])}
				}
				parsed = converted

			default:
				return nil, fmt.Errorf("unsupported output datatype %s for %s", output.Datatype, output.Name)
			}
		}

		if _, dup := result[output.Name]; dup {
			return nil, fmt.Errorf("duplicate output name %q in response", output.Name)
		}
		result[output.Name] = parsed
	}

	return result, nil
}

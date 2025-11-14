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
	grpcConn   *grpc.ClientConn
	grpcClient triton.GRPCInferenceServiceClient
}

// preparedInput represents processed input data ready for gRPC transport
type preparedInput struct {
	name     string
	datatype string      // Triton datatype: "BYTES", "INT32", "INT64", "FP32", "FP64"
	shape    []int64     // Shape in int64 for gRPC compatibility
	data     interface{} // Flattened data: []string, []int32, []int64, []float32, []float64
}

func (c *GRPCClient) ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) ([]interface{}, error) {
	preparedInputs, err := prepareInputs(indexToName, inputs)
	if err != nil {
		return nil, err
	}

	grpcRequest, err := buildGRPCRequest(modelName, preparedInputs)
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

func (c *GRPCClient) Close() error {
	return c.grpcConn.Close()
}

func buildGRPCRequest(modelName string, preparedInputs []preparedInput) (*triton.ModelInferRequest, error) {
	req := &triton.ModelInferRequest{
		ModelName: modelName,
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

func prepareInputs(indexToName map[int]string, params []interface{}) ([]preparedInput, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	var inputs []preparedInput

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

func convertGRPCResponse(response *triton.ModelInferResponse) ([]interface{}, error) {
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
			parsedData, err := parseRawOutput(rawData, output.Datatype, batchSize)
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

package triton

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	triton "github.com/viant/mly/proto/triton"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// createMockTritonConn creates a gRPC client connected to the mock server
func createMockTritonConn(ctx context.Context, t *testing.T, listener *bufconn.Listener) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	return conn
}

// mockTritonServer implements triton.GRPCInferenceServiceServer for testing
type mockTritonServer struct {
	triton.UnimplementedGRPCInferenceServiceServer

	modelReady bool
	responses  map[string]*triton.ModelInferResponse
}

func (m *mockTritonServer) RepositoryModelLoad(ctx context.Context, req *triton.RepositoryModelLoadRequest) (*triton.RepositoryModelLoadResponse, error) {
	return &triton.RepositoryModelLoadResponse{}, nil
}

func (m *mockTritonServer) ModelReady(ctx context.Context, req *triton.ModelReadyRequest) (*triton.ModelReadyResponse, error) {
	return &triton.ModelReadyResponse{Ready: m.modelReady}, nil
}

func (m *mockTritonServer) ModelInfer(ctx context.Context, req *triton.ModelInferRequest) (*triton.ModelInferResponse, error) {
	if resp, ok := m.responses[req.ModelName]; ok {
		return resp, nil
	}

	// Return a default response
	return &triton.ModelInferResponse{
		ModelName: req.ModelName,
		Outputs: []*triton.ModelInferResponse_InferOutputTensor{
			{
				Name:     "output",
				Datatype: "FP32",
				Shape:    []int64{1, 1},
				Contents: &triton.InferTensorContents{
					Fp32Contents: []float32{0.5},
				},
			},
		},
	}, nil
}

// startMockGRPCServer starts an in-memory gRPC server for testing
func startMockGRPCServer(t *testing.T, mock *mockTritonServer) (*grpc.Server, *bufconn.Listener) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	triton.RegisterGRPCInferenceServiceServer(server, mock)

	go func() {
		// Serve returns nil once server.Stop() is called during teardown; a
		// non-nil error means startup failed. Use Errorf, not Fatalf: Fatalf
		// calls runtime.Goexit on this goroutine (not the test goroutine), so
		// it cannot fail the test and can panic if it fires after completion.
		if err := server.Serve(listener); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()

	return server, listener
}

func createClient(t *testing.T, ctx context.Context, mock *mockTritonServer) (func(), *GRPCClient) {
	server, listener := startMockGRPCServer(t, mock)

	grpcConn := createMockTritonConn(ctx, t, listener)
	client := &GRPCClient{
		grpcConn:   grpcConn,
		grpcClient: triton.NewGRPCInferenceServiceClient(grpcConn),
	}

	return func() {
		server.Stop()
	}, client
}

func TestGRPCClient_ModelLoad(t *testing.T) {
	ctx := context.Background()

	// Set up mock server
	mock := &mockTritonServer{
		modelReady: true,
	}

	stopper, client := createClient(t, ctx, mock)
	defer stopper()

	err := client.ModelLoad(ctx, "test_model")
	require.NoError(t, err)
}

func TestGRPCClient_ModelInfer(t *testing.T) {
	ctx := context.Background()

	// Set up mock server
	mock := &mockTritonServer{
		modelReady: true,
		responses: map[string]*triton.ModelInferResponse{
			"test_model": {
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "INT64",
						Shape:    []int64{2, 1},
						Contents: &triton.InferTensorContents{
							Int64Contents: []int64{42, 100},
						},
					},
				},
			},
		},
	}

	stopper, client := createClient(t, ctx, mock)
	defer stopper()

	// Test prediction
	params := []interface{}{
		[][]string{{"value1"}, {"value2"}}, // 2 batch items
	}

	results, err := client.ModelInfer(ctx, "test_model", params, map[int]string{0: "input1"})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Verify output format
	output, ok := results["output"].([][]int64)
	require.True(t, ok, "expected [][]int64, got %T", results["output"])
	require.Len(t, output, 2)
	assert.Equal(t, []int64{42}, output[0])
	assert.Equal(t, []int64{100}, output[1])
}

func TestGRPCClient_ModelInferWithRawOutputContents(t *testing.T) {
	ctx := context.Background()

	// Set up mock server with raw output contents
	mock := &mockTritonServer{
		modelReady: true,
		responses: map[string]*triton.ModelInferResponse{
			"test_model": {
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "FP32",
						Shape:    []int64{2, 1},
					},
				},
				RawOutputContents: [][]byte{
					{0x00, 0x00, 0x20, 0x41, 0x00, 0x00, 0x48, 0x42}, // 10.0, 50.0 in float32 little-endian
				},
			},
		},
	}

	stopper, client := createClient(t, ctx, mock)
	defer stopper()

	params := []interface{}{
		[][]string{{"value1"}, {"value2"}},
	}

	results, err := client.ModelInfer(ctx, "test_model", params, map[int]string{0: "input1"})
	require.NoError(t, err)
	require.Len(t, results, 1)

	output, ok := results["output"].([][]float32)
	require.True(t, ok, "expected [][]float32, got %T", results["output"])
	require.Len(t, output, 2)
	assert.InDelta(t, 10.0, output[0][0], 0.01)
	assert.InDelta(t, 50.0, output[1][0], 0.01)
}

func TestGRPCClient_ModelInferAllInputTypes(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		inputType    string
		tritonType   string
		inputData    interface{}
		expectedResp *triton.ModelInferResponse
	}{
		{
			name:       "int32_input",
			inputType:  "int32",
			tritonType: "INT32",
			inputData:  [][]int32{{10}, {20}, {30}},
			expectedResp: &triton.ModelInferResponse{
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "INT32",
						Shape:    []int64{3, 1},
						Contents: &triton.InferTensorContents{
							IntContents: []int32{100, 200, 300},
						},
					},
				},
			},
		},
		{
			name:       "int64_input",
			inputType:  "int64",
			tritonType: "INT64",
			inputData:  [][]int64{{100}, {200}},
			expectedResp: &triton.ModelInferResponse{
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "INT64",
						Shape:    []int64{2, 1},
						Contents: &triton.InferTensorContents{
							Int64Contents: []int64{1000, 2000},
						},
					},
				},
			},
		},
		{
			name:       "float32_input",
			inputType:  "float32",
			tritonType: "FP32",
			inputData:  [][]float32{{1.5}, {2.5}, {3.5}, {4.5}},
			expectedResp: &triton.ModelInferResponse{
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "FP32",
						Shape:    []int64{4, 1},
						Contents: &triton.InferTensorContents{
							Fp32Contents: []float32{10.5, 20.5, 30.5, 40.5},
						},
					},
				},
			},
		},
		{
			name:       "float64_input",
			inputType:  "float64",
			tritonType: "FP64",
			inputData:  [][]float64{{1.111}},
			expectedResp: &triton.ModelInferResponse{
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "FP64",
						Shape:    []int64{1, 1},
						Contents: &triton.InferTensorContents{
							Fp64Contents: []float64{11.111},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockTritonServer{
				modelReady: true,
				responses:  map[string]*triton.ModelInferResponse{"test_model": tc.expectedResp},
			}

			stopper, client := createClient(t, ctx, mock)
			defer stopper()

			results, err := client.ModelInfer(ctx, "test_model", []interface{}{tc.inputData}, map[int]string{0: "input1"})
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.NotNil(t, results["output"])
		})
	}
}

func TestGRPCClient_ModelInferBytesOutput(t *testing.T) {
	ctx := context.Background()

	mock := &mockTritonServer{
		modelReady: true,
		responses: map[string]*triton.ModelInferResponse{
			"test_model": {
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "BYTES",
						Shape:    []int64{2, 1},
						Contents: &triton.InferTensorContents{
							BytesContents: [][]byte{[]byte("result1"), []byte("result2")},
						},
					},
				},
			},
		},
	}

	stopper, client := createClient(t, ctx, mock)
	defer stopper()

	params := []interface{}{
		[][]string{{"input1"}, {"input2"}},
	}

	results, err := client.ModelInfer(ctx, "test_model", params, map[int]string{0: "input1"})
	require.NoError(t, err)
	require.Len(t, results, 1)

	output, ok := results["output"].([][]string)
	require.True(t, ok, "expected [][]string, got %T", results["output"])
	require.Len(t, output, 2)
	assert.Equal(t, []string{"result1"}, output[0])
	assert.Equal(t, []string{"result2"}, output[1])
}

func TestGRPCClient_ModelInferDifferentBatchSizes(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name      string
		batchSize int
	}{
		{name: "single_item", batchSize: 1},
		{name: "small_batch", batchSize: 4},
		{name: "medium_batch", batchSize: 16},
		{name: "large_batch", batchSize: 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate expected output
			expectedOutput := make([]int64, tc.batchSize)
			for i := 0; i < tc.batchSize; i++ {
				expectedOutput[i] = int64(i * 10)
			}

			mock := &mockTritonServer{
				modelReady: true,
				responses: map[string]*triton.ModelInferResponse{
					"test_model": {
						ModelName: "test_model",
						Outputs: []*triton.ModelInferResponse_InferOutputTensor{
							{
								Name:     "output",
								Datatype: "INT64",
								Shape:    []int64{int64(tc.batchSize), 1},
								Contents: &triton.InferTensorContents{
									Int64Contents: expectedOutput,
								},
							},
						},
					},
				},
			}

			stopper, client := createClient(t, ctx, mock)
			defer stopper()

			// Generate input batch
			inputBatch := make([][]string, tc.batchSize)
			for i := 0; i < tc.batchSize; i++ {
				inputBatch[i] = []string{fmt.Sprintf("input_%d", i)}
			}

			results, err := client.ModelInfer(ctx, "test_model", []interface{}{inputBatch}, map[int]string{0: "input1"})
			require.NoError(t, err)
			require.Len(t, results, 1)

			output, ok := results["output"].([][]int64)
			require.True(t, ok, "expected [][]int64, got %T", results["output"])
			assert.Len(t, output, tc.batchSize)
		})
	}
}

func TestGRPCClient_ModelInferUnsupportedType(t *testing.T) {
	ctx := context.Background()

	mock := &mockTritonServer{
		modelReady: true,
		responses: map[string]*triton.ModelInferResponse{
			"test_model": {
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "UNKNOWN_TYPE",
						Shape:    []int64{1, 1},
						Contents: &triton.InferTensorContents{
							// Has contents but unsupported type
							BytesContents: [][]byte{[]byte("data")},
						},
					},
				},
			},
		},
	}

	stopper, client := createClient(t, ctx, mock)
	defer stopper()

	params := []interface{}{
		[][]string{{"test"}},
	}

	_, err := client.ModelInfer(ctx, "test_model", params, map[int]string{0: "input1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestGRPCClient_ModelInferMissingOutput(t *testing.T) {
	ctx := context.Background()

	mock := &mockTritonServer{
		modelReady: true,
		responses: map[string]*triton.ModelInferResponse{
			"test_model": {
				ModelName: "test_model",
				Outputs: []*triton.ModelInferResponse_InferOutputTensor{
					{
						Name:     "output",
						Datatype: "INT64",
						Shape:    []int64{1, 1},
						// Missing Contents field
					},
				},
			},
		},
	}

	stopper, client := createClient(t, ctx, mock)
	defer stopper()

	params := []interface{}{
		[][]string{{"test"}},
	}

	_, err := client.ModelInfer(ctx, "test_model", params, map[int]string{0: "input1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing contents")
}

// TestConvertGRPCResponse_KeysByName ensures each response tensor is keyed by its
// own wire name, so a response whose tensor order differs from the model's
// metadata order still maps values to the correct names (no positional aliasing).
func TestConvertGRPCResponse_KeysByName(t *testing.T) {
	response := &triton.ModelInferResponse{
		ModelName: "test_model",
		Outputs: []*triton.ModelInferResponse_InferOutputTensor{
			{
				Name:     "calibration",
				Datatype: "FP32",
				Shape:    []int64{1, 1},
				Contents: &triton.InferTensorContents{Fp32Contents: []float32{0.25}},
			},
			{
				Name:     "score",
				Datatype: "FP32",
				Shape:    []int64{1, 1},
				Contents: &triton.InferTensorContents{Fp32Contents: []float32{0.90}},
			},
		},
	}

	result, err := convertGRPCResponse(response)
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, [][]float32{{0.25}}, result["calibration"])
	assert.Equal(t, [][]float32{{0.90}}, result["score"])
}

func TestConvertGRPCResponse_DuplicateName(t *testing.T) {
	response := &triton.ModelInferResponse{
		ModelName: "test_model",
		Outputs: []*triton.ModelInferResponse_InferOutputTensor{
			{
				Name:     "score",
				Datatype: "FP32",
				Shape:    []int64{1, 1},
				Contents: &triton.InferTensorContents{Fp32Contents: []float32{0.25}},
			},
			{
				Name:     "score",
				Datatype: "FP32",
				Shape:    []int64{1, 1},
				Contents: &triton.InferTensorContents{Fp32Contents: []float32{0.90}},
			},
		},
	}

	_, err := convertGRPCResponse(response)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate output name")
}

package triton

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	triton "github.com/viant/mly/proto/triton"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/shared"
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

func createMockGRPCTritonEvaluator(t *testing.T, cfg *config.Model, grpcConn *grpc.ClientConn) *TritonEvaluator {
	grpcClient := triton.NewGRPCInferenceServiceClient(grpcConn)
	cfg.Triton.Init()
	evaluator, err := NewTritonEvaluator(cfg, map[string]TritonClient{
		cfg.Triton.ServerID: &GRPCClient{
			grpcConn:   grpcConn,
			grpcClient: grpcClient,
		},
	})

	require.NoError(t, err)
	return evaluator
}

func TestTritonEvaluator_ReloadAndSupportsReload(t *testing.T) {
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

	server, listener := startMockGRPCServer(t, mock)
	defer server.Stop()

	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output1", Index: 0, DataType: "float32"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			ServerID:  "test_server",
		},
	}

	grpcConn := createMockTritonConn(ctx, t, listener)
	evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
	defer evaluator.Close()

	// ReloadIfNeeded should be a no-op
	err := evaluator.ReloadIfNeeded(context.Background())
	assert.NoError(t, err)
}

func TestTritonEvaluator_PredictWithMockServer(t *testing.T) {
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

	server, listener := startMockGRPCServer(t, mock)
	defer server.Stop()

	// Create evaluator with mock connection
	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "int64"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			ServerID:  "test_server",
		},
	}
	grpcConn := createMockTritonConn(ctx, t, listener)
	evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
	defer evaluator.Close()

	// Test prediction
	params := []interface{}{
		[][]string{{"value1"}, {"value2"}}, // 2 batch items
	}

	results, err := evaluator.Predict(ctx, params)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Verify output format
	output, ok := results[0].([][]int64)
	require.True(t, ok, "expected [][]int64, got %T", results[0])
	require.Len(t, output, 2)
	assert.Equal(t, []int64{42}, output[0])
	assert.Equal(t, []int64{100}, output[1])
}

func TestTritonEvaluator_PredictWithRawOutputContents(t *testing.T) {
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

	server, listener := startMockGRPCServer(t, mock)
	defer server.Stop()

	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			ServerID:  "test_server",
		},
	}

	grpcConn := createMockTritonConn(ctx, t, listener)
	evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
	defer evaluator.Close()

	params := []interface{}{
		[][]string{{"value1"}, {"value2"}},
	}

	results, err := evaluator.Predict(ctx, params)
	require.NoError(t, err)
	require.Len(t, results, 1)

	output, ok := results[0].([][]float32)
	require.True(t, ok, "expected [][]float32, got %T", results[0])
	require.Len(t, output, 2)
	assert.InDelta(t, 10.0, output[0][0], 0.01)
	assert.InDelta(t, 50.0, output[1][0], 0.01)
}

func TestTritonEvaluator_PredictAllInputTypes(t *testing.T) {
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

			server, listener := startMockGRPCServer(t, mock)
			grpcConn := createMockTritonConn(ctx, t, listener)
			defer server.Stop()

			cfg := &config.Model{
				ID:       "test_model",
				Platform: "triton",
				MetaInput: shared.MetaInput{
					Inputs: []*shared.Field{
						{Name: "input1", Index: 0, DataType: tc.inputType},
					},
					Outputs: []*shared.Field{
						{Name: "output", Index: 0, DataType: tc.inputType},
					},
				},
				Triton: &config.TritonConfig{
					ModelName: "test_model",
					ServerID:  "test_server",
				},
			}

			evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
			defer evaluator.Close()

			results, err := evaluator.Predict(ctx, []interface{}{tc.inputData})
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.NotNil(t, results[0])
		})
	}
}

// mockTritonServer implements triton.GRPCInferenceServiceServer for testing
type mockTritonServer struct {
	triton.UnimplementedGRPCInferenceServiceServer
	modelReady bool
	responses  map[string]*triton.ModelInferResponse
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
		if err := server.Serve(listener); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	return server, listener
}

func TestTritonEvaluator_PredictBytesOutput(t *testing.T) {
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

	server, listener := startMockGRPCServer(t, mock)
	defer server.Stop()

	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "string"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			ServerID:  "test_server",
		},
	}

	grpcConn := createMockTritonConn(ctx, t, listener)
	evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
	defer evaluator.Close()

	params := []interface{}{
		[][]string{{"input1"}, {"input2"}},
	}

	results, err := evaluator.Predict(ctx, params)
	require.NoError(t, err)
	require.Len(t, results, 1)

	output, ok := results[0].([][]string)
	require.True(t, ok, "expected [][]string, got %T", results[0])
	require.Len(t, output, 2)
	assert.Equal(t, []string{"result1"}, output[0])
	assert.Equal(t, []string{"result2"}, output[1])
}

func TestTritonEvaluator_PredictDifferentBatchSizes(t *testing.T) {
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

			server, listener := startMockGRPCServer(t, mock)
			defer server.Stop()

			cfg := &config.Model{
				ID:       "test_model",
				Platform: "triton",
				MetaInput: shared.MetaInput{
					Inputs: []*shared.Field{
						{Name: "input1", Index: 0, DataType: "string"},
					},
					Outputs: []*shared.Field{
						{Name: "output", Index: 0, DataType: "int64"},
					},
				},
				Triton: &config.TritonConfig{
					ModelName: "test_model",
					ServerID:  "test_server",
				},
			}

			grpcConn := createMockTritonConn(ctx, t, listener)
			evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
			defer evaluator.Close()

			// Generate input batch
			inputBatch := make([][]string, tc.batchSize)
			for i := 0; i < tc.batchSize; i++ {
				inputBatch[i] = []string{fmt.Sprintf("input_%d", i)}
			}

			results, err := evaluator.Predict(ctx, []interface{}{inputBatch})
			require.NoError(t, err)
			require.Len(t, results, 1)

			output, ok := results[0].([][]int64)
			require.True(t, ok, "expected [][]int64, got %T", results[0])
			assert.Len(t, output, tc.batchSize)
		})
	}
}

func TestTritonEvaluator_PredictUnsupportedType(t *testing.T) {
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

	server, listener := startMockGRPCServer(t, mock)
	defer server.Stop()

	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "string"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			ServerID:  "test_server",
		},
	}

	grpcConn := createMockTritonConn(ctx, t, listener)
	evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
	defer evaluator.Close()

	params := []interface{}{
		[][]string{{"test"}},
	}

	_, err := evaluator.Predict(ctx, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestTritonEvaluator_PredictEmptyBatch(t *testing.T) {
	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "int64"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			ServerID:  "test_server",
		},
	}

	evaluator := createMockGRPCTritonEvaluator(t, cfg, nil)
	defer evaluator.Close()

	_, err := evaluator.Predict(context.Background(), []interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input parameters")
}

func TestTritonEvaluator_PredictMissingOutput(t *testing.T) {
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

	server, listener := startMockGRPCServer(t, mock)
	defer server.Stop()

	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "int64"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
			ServerID:  "test_server",
		},
	}

	grpcConn := createMockTritonConn(ctx, t, listener)
	evaluator := createMockGRPCTritonEvaluator(t, cfg, grpcConn)
	defer evaluator.Close()

	params := []interface{}{
		[][]string{{"test"}},
	}

	_, err := evaluator.Predict(ctx, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing contents")
}

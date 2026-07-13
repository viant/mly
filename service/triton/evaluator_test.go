package triton

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/shared"
)

type mockTritonClient struct {
	mu sync.Mutex

	readyState map[string]bool

	unloadCh     chan string
	modelLoadErr map[string]error

	metadata *ModelMetadata

	// inferResult, when non-nil, is returned by ModelInfer keyed by output name.
	inferResult map[string]interface{}
	inferErr    error
}

func (m *mockTritonClient) ServerReady(ctx context.Context) error { return nil }

func (m *mockTritonClient) ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) (map[string]interface{}, error) {
	return m.inferResult, m.inferErr
}

func (m *mockTritonClient) ModelReady(ctx context.Context, modelName string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ready, ok := m.readyState[modelName]; ok {
		return ready, nil
	}

	return true, nil
}

func (m *mockTritonClient) ModelLoad(ctx context.Context, modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.modelLoadErr[modelName]; err != nil {
		return err
	}

	if m.readyState == nil {
		m.readyState = make(map[string]bool)
	}

	m.readyState[modelName] = true

	return nil
}

func (m *mockTritonClient) ModelUnload(ctx context.Context, modelName string) error {
	ch := m.unloadCh
	if ch != nil {
		ch <- modelName
	}
	return nil
}

func (m *mockTritonClient) ModelMetadata(ctx context.Context, modelName string) (*ModelMetadata, error) {
	if m.metadata == nil {
		m.metadata = &ModelMetadata{
			Inputs: []MetadataTensor{
				{Name: "input1", Datatype: "BYTES"},
				{Name: "input2", Datatype: "BYTES"},
			},
			Outputs: []MetadataTensor{
				{Name: "output1", Datatype: "FP32"},
			},
		}
	}

	return m.metadata, nil
}

func (m *mockTritonClient) Close() error { return nil }

func newTritonEvaluator(cfg *config.Model, mockClient *mockTritonClient) *TritonEvaluator {
	cfg.Triton.Init(cfg.IsRouter())

	evaluator := &TritonEvaluator{
		modelName:          cfg.Triton.ModelName,
		isPrivateClient:    true,
		repositoryExplicit: false,
		service:            &Service{Client: mockClient},
		configuredInputs:   cfg.MetaInput.Inputs,
	}

	evaluator.ReloadIfNeeded(context.Background())

	return evaluator
}

func TestTritonEvaluator_Signature(t *testing.T) {
	cfg := &config.Model{
		ID: "test_model",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
	}

	evaluator := newTritonEvaluator(cfg, &mockTritonClient{})

	defer evaluator.Close()

	sig := evaluator.Signature()
	require.NotNil(t, sig)
	assert.Equal(t, 2, len(sig.Inputs))
	assert.Equal(t, 1, len(sig.Outputs))
	assert.Equal(t, "input1", sig.Inputs[0].Name)
	assert.Equal(t, "output1", sig.Outputs[0].Name)
}

func TestTritonEvaluator_Inputs(t *testing.T) {
	cfg := &config.Model{
		ID: "test_model",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
				{Name: "input2", Index: 0, DataType: "string"},
				{Name: "input_aux", Index: 0, DataType: "string", Auxiliary: true},
			},
		},
	}

	evaluator := newTritonEvaluator(cfg, &mockTritonClient{})

	defer evaluator.Close()

	inputMap := evaluator.Inputs()
	require.NotNil(t, inputMap)
	assert.Equal(t, 3, len(inputMap))

	i1 := inputMap["input1"]
	assert.Equal(t, "input1", i1.Name)
	assert.Equal(t, false, i1.Auxiliary)
	assert.Equal(t, 0, i1.Index)

	i2 := inputMap["input2"]
	assert.Equal(t, "input2", i2.Name)
	assert.Equal(t, false, i2.Auxiliary)
	assert.Equal(t, 1, i2.Index)

	iAux := inputMap["input_aux"]
	assert.Equal(t, "input_aux", iAux.Name)
	assert.Equal(t, true, iAux.Auxiliary)
	assert.Equal(t, 2, iAux.Index)
}

func TestTritonEvaluator_SignatureWithAuxiliaryInputs(t *testing.T) {
	cfg := &config.Model{
		ID: "test_model",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input_aux", Index: 0, DataType: "string", Auxiliary: true},
			},
		},
	}

	evaluator := newTritonEvaluator(cfg, &mockTritonClient{
		metadata: &ModelMetadata{
			Inputs: []MetadataTensor{
				{Name: "input1", Datatype: "BYTES"},
			},
			Outputs: []MetadataTensor{
				{Name: "output1", Datatype: "FP32"},
			},
		},
	})

	defer evaluator.Close()

	sig := evaluator.Signature()
	require.NotNil(t, sig)
	assert.Equal(t, 1, len(sig.Inputs))
	assert.Equal(t, "input1", sig.Inputs[0].Name)
}

func TestTritonEvaluator_PredictEmptyBatch(t *testing.T) {
	cfg := &config.Model{
		ID: "test_model",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
	}

	evaluator := newTritonEvaluator(cfg, &mockTritonClient{})
	defer evaluator.Close()

	_, err := evaluator.Predict(context.Background(), []interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input parameters")
}

// TestTritonEvaluator_PredictOrdersOutputsByName guards the core invariant of the
// name-keyed ModelInfer contract: the client returns outputs keyed by name (no
// order), and Predict must place them into signature.Outputs order. The map is
// deliberately built so its keys do not align positionally with the signature.
func TestTritonEvaluator_PredictOrdersOutputsByName(t *testing.T) {
	cfg := &config.Model{
		ID: "test_model",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
	}

	mock := &mockTritonClient{
		metadata: &ModelMetadata{
			Inputs: []MetadataTensor{
				{Name: "input1", Datatype: "FP32"},
			},
			// signature output order is [score, calibration]
			Outputs: []MetadataTensor{
				{Name: "score", Datatype: "FP32"},
				{Name: "calibration", Datatype: "FP32"},
			},
		},
		inferResult: map[string]interface{}{
			"calibration": [][]float32{{0.25}},
			"score":       [][]float32{{0.90}},
		},
	}

	evaluator := newTritonEvaluator(cfg, mock)
	defer evaluator.Close()

	sig := evaluator.Signature()
	require.NotNil(t, sig)
	require.Equal(t, "score", sig.Outputs[0].Name)
	require.Equal(t, "calibration", sig.Outputs[1].Name)

	results, err := evaluator.Predict(context.Background(), []interface{}{[][]float32{{1.0}}})
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, [][]float32{{0.90}}, results[0], "index 0 must be score, per signature order")
	assert.Equal(t, [][]float32{{0.25}}, results[1], "index 1 must be calibration, per signature order")
}

func TestTritonEvaluator_PredictMissingOutput(t *testing.T) {
	cfg := &config.Model{
		ID: "test_model",
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
	}

	mock := &mockTritonClient{
		metadata: &ModelMetadata{
			Inputs:  []MetadataTensor{{Name: "input1", Datatype: "FP32"}},
			Outputs: []MetadataTensor{{Name: "score", Datatype: "FP32"}, {Name: "calibration", Datatype: "FP32"}},
		},
		inferResult: map[string]interface{}{
			"score": [][]float32{{0.90}},
		},
	}

	evaluator := newTritonEvaluator(cfg, mock)
	defer evaluator.Close()

	_, err := evaluator.Predict(context.Background(), []interface{}{[][]float32{{1.0}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing output")
}

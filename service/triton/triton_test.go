package triton

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/shared"
)

func newTritonEvaluator(t *testing.T, cfg *config.Model) *TritonEvaluator {
	cfg.Triton.Init()
	evaluator, err := NewTritonEvaluator(cfg, nil)
	require.NoError(t, err)
	return evaluator
}

func TestTritonEvaluator_Signature(t *testing.T) {
	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		URL:      "http://localhost:8000",

		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
				{Name: "input2", Index: 1, DataType: "int64"},
			},
			Outputs: []*shared.Field{
				{Name: "output1", Index: 0, DataType: "float32"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
	}

	evaluator := newTritonEvaluator(t, cfg)
	defer evaluator.Close()

	sig := evaluator.Signature()
	require.NotNil(t, sig)
	assert.Equal(t, 2, len(sig.Inputs))
	assert.Equal(t, 1, len(sig.Outputs))
	assert.Equal(t, "input1", sig.Inputs[0].Name)
	assert.Equal(t, "output1", sig.Outputs[0].Name)
}

func TestTritonEvaluator_SignatureWithAuxiliaryInputs(t *testing.T) {
	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		URL:      "http://localhost:8000",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "input1", Index: 0, DataType: "string"},
				{Name: "auxiliary_input", Index: 1, DataType: "int64", Auxiliary: true},
			},
			Outputs: []*shared.Field{
				{Name: "output1", Index: 0, DataType: "float32"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
	}

	evaluator := newTritonEvaluator(t, cfg)
	defer evaluator.Close()

	sig := evaluator.Signature()
	require.NotNil(t, sig)
	// Auxiliary inputs should be excluded from signature
	assert.Equal(t, 1, len(sig.Inputs))
	assert.Equal(t, "input1", sig.Inputs[0].Name)
}

func TestTritonEvaluator_Dictionary(t *testing.T) {
	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		URL:      "http://localhost:8000",
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
		},
	}

	evaluator := newTritonEvaluator(t, cfg)
	defer evaluator.Close()

	dict := evaluator.Dictionary()
	assert.Nil(t, dict)
}

func TestTritonEvaluator_InputsMapping(t *testing.T) {
	cfg := &config.Model{
		ID:       "test_model",
		Platform: "triton",
		URL:      "http://localhost:8000",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{Name: "string_input", Index: 0, DataType: "string"},
				{Name: "int64_input", Index: 1, DataType: "int64"},
				{Name: "int32_input", Index: 2, DataType: "int32"},
				{Name: "float32_input", Index: 3, DataType: "float32"},
				{Name: "float64_input", Index: 4, DataType: "float64"},
			},
			Outputs: []*shared.Field{
				{Name: "output", Index: 0, DataType: "float32"},
			},
		},
		Triton: &config.TritonConfig{
			ModelName: "test_model",
		},
	}

	evaluator := newTritonEvaluator(t, cfg)
	defer evaluator.Close()

	inputs := evaluator.Inputs()
	assert.Len(t, inputs, 5)
	assert.Contains(t, inputs, "string_input")
	assert.Contains(t, inputs, "int64_input")
	assert.Contains(t, inputs, "int32_input")
	assert.Contains(t, inputs, "float32_input")
	assert.Contains(t, inputs, "float64_input")
}

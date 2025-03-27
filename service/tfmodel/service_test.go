package tfmodel

import (
	"reflect"
	"testing"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared"
)

func TestReconcileIOFromSignature(t *testing.T) {
	// Create test config
	cfg := &config.Model{
		ID: "test_model",
		MetaInput: shared.MetaInput{
			Inputs: []*shared.Field{
				{
					Name:     "input1",
					DataType: "string",
				},
				{
					Name:      "input2",
					DataType:  "int32",
					Auxiliary: true,
				},
			},
			Outputs: []*shared.Field{
				{
					Name: "output1",
				},
			},
		},
	}

	cfg.Init(nil)

	output := domain.Output{
		Name: "output1",
	}

	output.SetType(reflect.TypeOf(float64(0)))

	// Create test signature
	signature := &domain.Signature{
		Inputs: []domain.Input{
			{
				Name: "input1",
				Type: reflect.TypeOf(""),
			},
		},
		Outputs: []domain.Output{
			output,
		},
	}

	reconciled := reconcileIOFromSignature(cfg, signature)

	if len(reconciled) != 2 {
		t.Fatalf("expected 2 reconciled inputs, but got %v", len(reconciled))
	}

	input1, exists := reconciled["input1"]
	if !exists {
		t.Fatal("expected input1 to exist in reconciled inputs")
	}

	if input1.Type != reflect.TypeOf("") {
		t.Errorf("expected input1.Type to be string, got %v", input1.Type)
	}

	if cfg.Outputs[0].DataType != "float64" {
		t.Errorf("expected cfg.Outputs[0].DataType to be float64, got %v", cfg.Outputs[0].DataType)
	}
}

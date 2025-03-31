package tfmodel

import (
	"log"
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

	// emulate model signature

	output := domain.Output{
		Name: "output1",
	}

	output.SetType(reflect.TypeOf(float64(0)))

	strType := reflect.TypeOf("")
	// Create test signature
	signature := &domain.Signature{
		Inputs: []domain.Input{
			{
				Name: "input1",
				Type: strType,
			},
			{
				Name: "input3",
				Type: strType,
			},
		},
		Outputs: []domain.Output{
			output,
		},
	}

	reconciled := reconcileIOFromSignature(cfg, signature)

	for _, input := range cfg.Inputs {
		log.Printf("%s: %+v", input.Name, *input)
	}

	if len(reconciled) != 3 {
		t.Fatalf("expected 3 reconciled inputs, but got %v", len(reconciled))
	}

	input1, exists := reconciled["input1"]
	if !exists {
		t.Fatal("expected input1 to exist in reconciled inputs")
	}

	if input1.Type != reflect.TypeOf("") {
		t.Errorf("expected input1.Type to be string, got %v", input1.Type)
	}

	if len(cfg.Outputs) != 1 {
		t.Fatalf("expected 1 output, but got %v", len(cfg.Outputs))
	}

	if cfg.Outputs[0].DataType != "float64" {
		t.Errorf("expected cfg.Outputs[0].DataType to be float64, got %v", cfg.Outputs[0].DataType)
	}
}

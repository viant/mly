package triton

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

// TritonEvaluator implements PlatformEvaluator for Triton Inference Server via gRPC
type TritonEvaluator struct {
	client TritonClient

	modelName string

	// if true, this client is used only for this instance
	isPrivateClient    bool
	repositoryExplicit bool

	timeout time.Duration

	signature   *domain.Signature
	indexToName map[int]string

	inputs map[string]*domain.Input
}

// NewTritonEvaluator creates a new Triton evaluator
func NewTritonEvaluator(config *config.Model, tritonClients map[string]TritonClient) (*TritonEvaluator, error) {
	var client TritonClient

	isPrivateClient := config.URL != ""
	timeout := time.Duration(config.Triton.Timeout) * time.Millisecond

	if isPrivateClient {
		// "Private" URL configuration will only support HTTP
		client = &HTTPClient{
			httpClient: &http.Client{
				Timeout: timeout,
			},
			serverURL: config.URL,
		}
	} else {
		client = tritonClients[config.Triton.ServerID]
		if client == nil {
			return nil, fmt.Errorf("client not found for Triton, server ID: %s", config.Triton.ServerID)
		}
	}

	evaluator := &TritonEvaluator{
		client:    client,
		modelName: config.Triton.ModelName,
		timeout:   timeout,

		isPrivateClient: isPrivateClient,

		// clients defined in TritonServers are assumed to be in EXPLICIT mode
		repositoryExplicit: !isPrivateClient || config.Triton.RepositoryExplicit,
	}

	if err := evaluator.handleIO(&config.MetaInput); err != nil {
		return nil, err
	}

	return evaluator, nil
}

func NewRoutedTritonEvaluator(modelName string, client TritonClient, timeoutMs int, indexToName map[int]string) (*TritonEvaluator, error) {
	return &TritonEvaluator{
		client:             client,
		modelName:          modelName,
		timeout:            time.Duration(timeoutMs) * time.Millisecond,
		repositoryExplicit: true,
		indexToName:        indexToName,
	}, nil
}

// Predict performs inference via Triton Inference Server
func (t *TritonEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	requestCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	return t.client.ModelInfer(requestCtx, t.modelName, params, t.indexToName)
}

func (t *TritonEvaluator) handleIO(io *shared.MetaInput) error {
	var inputs []domain.Input
	var outputs []domain.Output

	indexToName := make(map[int]string)

	mappedInputs := make(map[string]*domain.Input)

	if len(io.Inputs) > 0 {
		for _, input := range io.Inputs {
			if !input.Auxiliary {
				inputs = append(inputs, domain.Input{
					Name:  input.Name,
					Index: input.Index,
				})

				indexToName[input.Index] = input.Name
			}

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

			mappedInputs[input.Name] = &domain.Input{
				Name:      input.Name,
				Index:     input.Index,
				Type:      inputType,
				Vocab:     false,
				Auxiliary: input.Auxiliary,
			}
		}
	} else {
		return fmt.Errorf("missing input configuration for Triton evaluator. " +
			"Add 'inputs' section to your model configuration YAML with field definitions")
	}

	if len(io.Outputs) > 0 {
		for i, output := range io.Outputs {
			outputs = append(outputs, domain.Output{
				Name:     output.Name,
				Index:    i,
				DataType: output.DataType,
			})
		}
	} else {
		return fmt.Errorf("missing output configuration for Triton evaluator. " +
			"Add 'outputs' section to your model configuration YAML with field definitions")
	}

	t.indexToName = indexToName

	t.signature = &domain.Signature{
		Inputs:  inputs,
		Outputs: outputs,
		Output:  outputs[0],
	}

	t.inputs = mappedInputs

	return nil
}

func (t *TritonEvaluator) Signature() *domain.Signature {
	return t.signature
}

func (t *TritonEvaluator) Dictionary() *common.Dictionary {
	return nil
}

func (t *TritonEvaluator) Stats(stats map[string]interface{}) {
	// no stats
}

func (t *TritonEvaluator) Inputs() map[string]*domain.Input {
	return t.inputs
}

// Close releases Triton client resources and stops health monitoring
func (t *TritonEvaluator) Close() error {
	if t.isPrivateClient {
		return t.client.Close()
	}

	return nil
}

// ReloadIfNeeded for independent Triton models, reloading is not supported.
func (t *TritonEvaluator) ReloadIfNeeded(ctx context.Context) error {
	ready, err := t.client.ModelReady(ctx, t.modelName)
	if err != nil {
		return fmt.Errorf("failed to check Triton model %s health: %w", t.modelName, err)
	}

	if !t.repositoryExplicit {
		return fmt.Errorf("model %s not ready and Triton is not in EXPLICIT Model Control Mode", t.modelName)
	}

	if ready {
		return nil
	}

	err = t.client.ModelLoad(ctx, t.modelName)
	if err != nil {
		return fmt.Errorf("failed to load Triton model %s: %w", t.modelName, err)
	}

	ready, err = t.client.ModelReady(ctx, t.modelName)
	if err != nil {
		return fmt.Errorf("failed to check Triton model %s health after loading: %w", t.modelName, err)
	}

	if !ready {
		return fmt.Errorf("model %s is not ready after loading", t.modelName)
	}

	return nil
}

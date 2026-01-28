package triton

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

// TritonEvaluator implements service/platform.PlatformEvaluator.
type TritonEvaluator struct {
	service   *Service
	modelName string

	// if true, this client is used only for this instance
	isPrivateClient    bool
	repositoryExplicit bool

	timeout time.Duration

	modelID string
	debug   bool

	signature *domain.Signature

	// maps Feeds index to input name
	indexToName map[int]string

	configuredInputs []*shared.Field

	inputs map[string]*domain.Input
}

// NewTritonEvaluator creates a new Triton evaluator
func NewTritonEvaluator(config *config.Model, tritonClients map[string]*Service) (*TritonEvaluator, error) {
	evaluator, err := createEvaluator(config, tritonClients)
	if err != nil {
		return nil, err
	}
	err = evaluator.registerUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to register usage for Triton evaluator: %w", err)
	}

	return evaluator, nil
}

func createEvaluator(config *config.Model, tritonClients map[string]*Service) (*TritonEvaluator, error) {
	var service *Service

	isPrivateClient := config.URL != ""
	timeout := time.Duration(config.Triton.Timeout) * time.Millisecond

	if isPrivateClient {
		// "Private" URL configuration will only support HTTP
		client := &HTTPClient{
			httpClient: &http.Client{
				Timeout: timeout,
			},
			serverURL: config.URL,
			debug:     config.Debug,
		}

		service = &Service{
			Client: client,
		}
	} else {
		service = tritonClients[config.Triton.ServerID]
		if service == nil {
			return nil, fmt.Errorf("client not found for Triton, server ID: %s", config.Triton.ServerID)
		}
	}

	evaluator := &TritonEvaluator{
		service: service,

		modelName: config.Triton.ModelName,
		timeout:   timeout,

		isPrivateClient: isPrivateClient,

		// clients defined in TritonServers are assumed to be in EXPLICIT mode
		repositoryExplicit: !isPrivateClient || config.Triton.RepositoryExplicit,

		configuredInputs: config.MetaInput.Inputs,

		modelID: config.ID,
		debug:   config.Debug,
	}

	return evaluator, nil

}

// Upward dependency, but provides Evaluators as needed for the service/platform/router module.
func NewRoutedTritonEvaluator(modelName string, config *config.Model, tritonClients map[string]*Service) (*TritonEvaluator, error) {
	evaluator, err := createEvaluator(config, tritonClients)
	if err != nil {
		return nil, fmt.Errorf("failed to create Triton Routed evaluator: %w", err)
	}

	evaluator.modelName = modelName
	evaluator.configuredInputs = nil // routed evaluators must not have any additional inputs

	err = evaluator.registerUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to register usage for Triton Routed evaluator: %w", err)
	}

	return evaluator, nil
}

func (t *TritonEvaluator) registerUsage() error {
	if t.modelName == "" {
		return fmt.Errorf("model name is required for registering usage")
	}

	t.service.RegisterUsage(t.modelID, t.modelName)
	return nil
}

// Predict performs inference via Triton Inference Server
func (t *TritonEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters")
	}

	requestCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	return t.service.Client.ModelInfer(requestCtx, t.modelName, params, t.indexToName)
}

func (t *TritonEvaluator) Signature() *domain.Signature {
	return t.signature
}

func (t *TritonEvaluator) Dictionary() *common.Dictionary {
	// no dictionary
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
		return t.service.Client.Close()
	}

	return nil
}

// For independent Triton server models, reloading is not supported.
func (t *TritonEvaluator) ReloadIfNeeded(ctx context.Context) error {
	ready, err := t.service.Client.ModelReady(ctx, t.modelName)
	if err != nil {
		return fmt.Errorf("failed to check Triton model %s health: %w", t.modelName, err)
	}

	if ready && t.signature != nil {
		// only a health check
		return nil
	}

	if !ready {
		if !t.repositoryExplicit {
			return fmt.Errorf("model %s not ready and Triton is not in EXPLICIT Model Control Mode: %w", t.modelName, err)
		}

		err = t.service.Client.ModelLoad(ctx, t.modelName)
		if err != nil {
			return fmt.Errorf("failed to load Triton model %s: %w", t.modelName, err)
		}

		ready, err = t.service.Client.ModelReady(ctx, t.modelName)
		if err != nil {
			return fmt.Errorf("failed to check Triton model %s health after loading: %w", t.modelName, err)
		}
	}

	if !ready {
		return fmt.Errorf("model %s is not ready after loading", t.modelName)
	}

	// we need to get the model metadata and consolidate the signature
	metadata, err := t.service.Client.ModelMetadata(ctx, t.modelName)
	if err != nil || metadata == nil {
		return fmt.Errorf("failed to get Triton model %s metadata: %w", t.modelName, err)
	}

	mappedInputs := make(map[string]*domain.Input)
	indexedInputNames := make(map[int]string)

	signatureInputs := make([]domain.Input, len(metadata.Inputs))
	for i, input := range metadata.Inputs {
		goType := TritonToGoType(input.Datatype)
		di := domain.Input{
			Name: input.Name,
			// for now, since the request provides a []interface{}, we populate the Index
			Index:     i,
			Type:      goType,
			Vocab:     false,
			Auxiliary: false,
		}

		if t.debug {
			log.Printf("[%s] Triton[%s] input:%s index:%d datatype:%s goType:%s",
				t.modelID, t.modelName, input.Name, di.Index, input.Datatype, goType.Name())
		}

		signatureInputs[i] = di
		mappedInputs[input.Name] = &di
		indexedInputNames[i] = input.Name
	}

	t.indexToName = indexedInputNames

	for _, input := range t.configuredInputs {
		iName := input.Name
		if _, ok := mappedInputs[iName]; !ok {
			goType, err := common.DataType(input.DataType)
			if err != nil {
				return fmt.Errorf("failed to get data type for %s: %w", iName, err)
			}

			mappedInputs[iName] = &domain.Input{
				Name:      iName,
				Index:     len(mappedInputs),
				Type:      goType,
				Vocab:     false,
				Auxiliary: true,
			}

			if t.debug {
				log.Printf("[%s] Triton[%s] auxiliary input:%s goType:%s",
					t.modelID, t.modelName, iName, goType.Name())
			}
		}
	}

	t.inputs = mappedInputs

	outputs := make([]domain.Output, len(metadata.Outputs))
	for i, output := range metadata.Outputs {
		o := domain.Output{
			Name:  output.Name,
			Index: len(outputs),
		}

		goType := TritonToGoType(output.Datatype)
		o.SetType(goType)
		o.DataType = goType.Name()
		o.DataTypeKind = goType.Kind()

		outputs[i] = o
	}

	t.signature = &domain.Signature{
		Inputs:  signatureInputs,
		Outputs: outputs,
		Output:  outputs[0],
	}

	return nil
}

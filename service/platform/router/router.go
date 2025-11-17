package router

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"

	"github.com/viant/afs"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/files"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/service/request/shape"
	tricli "github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/config/router"
	"gopkg.in/yaml.v2"
)

// Router implements the PlatformEvaluator interface for router mode.
type Router struct {
	configURL      string
	fs             afs.Service
	configLock     sync.RWMutex
	configModified *config.Modified
	routerConfig   *router.RouterConfig

	routingTableLock sync.RWMutex
	routingMap       map[int]string
	routingTable     map[string]platform.PlatformEvaluator

	maxConcurrency int

	globalModel     platform.PlatformEvaluator
	fixedEvaluator  platform.Predictor
	modelOutputName string

	modelConfig  *config.Model
	tritonClient tricli.TritonClient

	signature   *domain.Signature
	indexToName map[int]string
	inputs      map[string]*domain.Input

	// router input offset is the index of the router input in the inputs array
	routerInputOffset int
}

func NewRouter(cfg *config.Model, fs afs.Service, tritonClients map[string]tricli.TritonClient) (*Router, error) {
	if cfg.Router == nil {
		return nil, fmt.Errorf("router configuration is required")
	}

	tritonClient, ok := tritonClients[cfg.Triton.ServerID]
	if !ok {
		return nil, fmt.Errorf("triton client not found for server ID: %s", cfg.Triton.ServerID)
	}

	r := &Router{
		configURL: cfg.Router.ConfigURL,
		fs:        fs,

		modelConfig:  cfg,
		tritonClient: tritonClient,
	}

	if err := r.handleIO(cfg); err != nil {
		return nil, fmt.Errorf("failed to handle IO: %w", err)
	}

	if cfg.Router.MaxConcurrency != 0 {
		r.maxConcurrency = cfg.Router.MaxConcurrency
	}

	return r, nil
}

type preparedReplacement struct {
	typ   string
	value interface{}
}

func (t *Router) handleIO(cfg *config.Model) error {
	io := &cfg.MetaInput

	if len(io.Inputs) == 0 {
		return fmt.Errorf("input configuration is required for a router")
	}

	if len(io.Outputs) == 0 {
		return fmt.Errorf("output configuration is required for a router")
	}

	var inputs []domain.Input

	// for declaring the router's inputs
	mappedInputs := make(map[string]*domain.Input)

	// generate backend input
	indexToName := make(map[int]string)

	i := 0
	for _, input := range io.Inputs {
		if !input.Auxiliary && input.Name != cfg.Router.InputName {
			inputs = append(inputs, domain.Input{
				Name:  input.Name,
				Index: input.Index,
			})

			indexToName[i] = input.Name
			i++
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
			case "float32", "float":
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

	var outputs []domain.Output
	outputByName := make(map[string]domain.Output)

	for i, output := range io.Outputs {
		outputs = append(outputs, domain.Output{
			Name:     output.Name,
			Index:    i,
			DataType: output.DataType,
		})

		outputByName[output.Name] = outputs[i]
	}

	modelOutputName := cfg.Router.Output.FieldName
	hasModelOutputName := modelOutputName != ""

	if !cfg.Router.Global.Exists {
		replacementsByName := make(map[string]config.PredictionReplacement)
		for _, repl := range cfg.Router.Global.PredictionReplacements {
			replacementsByName[repl.Name] = repl
		}

		replacementOutputs := make([]config.PredictionReplacement, 0, len(outputs))
		for _, output := range outputs {
			if hasModelOutputName && output.Name == modelOutputName {
				// model-used output field name is handled in a different way
				continue
			}

			if _, ok := replacementsByName[output.Name]; !ok {
				return fmt.Errorf("replacement for output %s not found", output.Name)
			}

			replacementOutputs = append(replacementOutputs, replacementsByName[output.Name])
		}

		fixedEvaluator, err := newFixedEvaluator(replacementOutputs)
		if err != nil {
			return fmt.Errorf("failed to create fixed evaluator: %w", err)
		}

		t.fixedEvaluator = fixedEvaluator
	}

	var modelOutputInOutputs bool = !hasModelOutputName
	if hasModelOutputName {
		_, modelOutputInOutputs = outputByName[modelOutputName]
	}

	if !modelOutputInOutputs {
		outputs = append(outputs, domain.Output{
			Name:     modelOutputName,
			DataType: "string",
			Index:    len(outputs),
		})
	}

	t.modelOutputName = modelOutputName

	t.indexToName = indexToName

	t.signature = &domain.Signature{
		Inputs:  inputs,
		Outputs: outputs,
		Output:  outputs[0],
	}

	t.inputs = mappedInputs

	return nil
}

type offsetResults struct {
	offset  int
	results []interface{}
}

// Predict performs model inference with the given parameters
// params is expected to be [numInputs]([batchSize][1]T) (see service/request.Request.Feeds)
func (r *Router) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	expectedBatchSize, err := shape.DetermineBatchSize(params)
	if err != nil {
		return nil, err
	}

	numInputs := len(params)

	r.routingTableLock.RLock()
	defer r.routingTableLock.RUnlock()

	globalExists := r.modelConfig.Router.Global.Exists
	reportedGlobalModelName := r.modelConfig.Router.Output.GlobalModelOverride
	noModelName := r.modelConfig.Router.Output.NoModelID

	predictWaitGroup := sync.WaitGroup{}
	predictWaitGroup.Add(expectedBatchSize)

	errCh := make(chan error, expectedBatchSize)
	resultsCh := make(chan offsetResults, expectedBatchSize)

	for batchOffset := range expectedBatchSize {
		// 1 input is reserved for the router input
		request := make([]interface{}, numInputs-1)

		var routingValueBatched interface{}

		for inputOffset := range numInputs {
			debatched, err := shape.Debatch(params[inputOffset], batchOffset)
			if err != nil {
				return nil, fmt.Errorf("failed to debatch for row %d and input %d: %w", batchOffset, inputOffset, err)
			}

			if inputOffset < r.routerInputOffset {
				request[inputOffset] = debatched
			} else if inputOffset == r.routerInputOffset {
				routingValueBatched = debatched
			} else {
				request[inputOffset-1] = debatched
			}
		}

		routingValue, err := shape.SqueezeBatch(routingValueBatched)
		if err != nil {
			return nil, fmt.Errorf("failed to extract from batch for row %d: %w", batchOffset, err)
		}

		var ok bool = true
		var routingValueInt int
		switch routingValue := routingValue.(type) {
		case int:
			routingValueInt = routingValue
		case int32:
			routingValueInt = int(routingValue)
		case int64:
			routingValueInt = int(routingValue)
		default:
			ok = false
		}

		if !ok {
			return nil, fmt.Errorf("routing value is not an int: %v, is %T, for row %d", routingValue, routingValue, batchOffset)
		}

		routingValueString, ok := r.routingMap[routingValueInt]

		var evaluator platform.Predictor
		if !ok {
			if globalExists {
				// fallback to global model
				evaluator = r.globalModel

				// override model name
				if reportedGlobalModelName != "" {
					routingValueString = reportedGlobalModelName
				}
			} else {
				routingValueString = noModelName
				evaluator = r.fixedEvaluator
			}
		} else {
			var ok bool
			evaluator, ok = r.routingTable[routingValueString]
			if !ok {
				return nil, fmt.Errorf("no evaluator found for routing value: %v", routingValue)
			}
		}

		go func(batchOffset int, request []interface{}, routingValueString string, evaluator platform.Predictor) {
			defer predictWaitGroup.Done()

			results, err := evaluator.Predict(ctx, request)
			if err != nil {
				errCh <- fmt.Errorf("failed to predict for row %d: %w", batchOffset, err)
			}

			if r.modelOutputName != "" {
				// TODO ensure ordering

				results = append(results, [][]string{{routingValueString}})
			}

			resultsCh <- offsetResults{offset: batchOffset, results: results}
		}(batchOffset, request, routingValueString, evaluator)
	}

	predictWaitGroup.Wait()
	close(errCh)
	close(resultsCh)

	for err := range errCh {
		return nil, err
	}

	allResults := make([][]interface{}, expectedBatchSize)
	for results := range resultsCh {
		allResults[results.offset] = results.results
	}

	endResults := make([]interface{}, len(r.signature.Outputs))
	for i, results := range allResults {
		endResults, err = shape.ConcatAxis0(endResults, results)
		if err != nil {
			return nil, fmt.Errorf("failed to concatenate results for row %d: %w", i, err)
		}
	}

	return endResults, nil
}

func (r *Router) Signature() *domain.Signature {
	return r.signature
}

func (r *Router) Dictionary() *common.Dictionary {
	return nil
}

func (r *Router) Inputs() map[string]*domain.Input {
	return r.inputs
}

func (r *Router) Stats(stats map[string]interface{}) {

}

func (r *Router) Close() error {
	return nil
}

// TODO refactor with service/tfmodel/service.isModified()?
func (r *Router) isModified(snapshot *config.Modified) bool {
	if r.routerConfig == nil || r.configModified == nil {
		return true
	}

	if snapshot.Max.IsZero() {
		return false
	}

	r.configLock.RLock()
	modified := r.configModified
	r.configLock.RUnlock()

	return !(modified.Max.Equal(snapshot.Max) && modified.Min.Equal(snapshot.Min))
}

func (r *Router) ReloadIfNeeded(ctx context.Context) error {
	// fetch and check router configuration file
	snapshot, err := files.ModifiedSnapshot(ctx, r.fs, r.configURL, nil)
	if err != nil {
		return fmt.Errorf("failed to check router configuration file: %w", err)
	}

	if !r.isModified(snapshot) {
		// check health of all underlying models
		var wg sync.WaitGroup

		errChannels := len(r.routingTable)
		if r.globalModel != nil {
			errChannels++
		}

		errCh := make(chan error, errChannels)

		if r.globalModel != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := r.globalModel.ReloadIfNeeded(ctx)
				if err != nil {
					errCh <- fmt.Errorf("failed to reload global model: %w", err)
				}
			}()
		}

		for m, p := range r.routingTable {
			wg.Add(1)
			go func(m string, p platform.PlatformEvaluator) {
				defer wg.Done()
				err := p.ReloadIfNeeded(ctx)
				if err != nil {
					errCh <- fmt.Errorf("failed to reload model %s: %w", m, err)
				}
			}(m, p)
		}

		wg.Wait()
		close(errCh)

		if len(errCh) > 0 {
			var errStrings []string
			for err := range errCh {
				errStrings = append(errStrings, err.Error())
			}

			err = fmt.Errorf("one or more model reloading errors: %s", strings.Join(errStrings, "; "))
		}

		return err
	}

	// otherwise just abandon the routing table status checks

	r.configLock.Lock()
	defer r.configLock.Unlock()

	r.configModified = snapshot

	// load router configuration file
	rawReader, err := r.fs.OpenURL(ctx, r.configURL)
	if err != nil {
		return fmt.Errorf("failed to open router configuration file: %w", err)
	}

	defer rawReader.Close()
	var reader io.Reader = rawReader
	if strings.HasSuffix(r.configURL, ".gz") {
		if reader, err = gzip.NewReader(rawReader); err != nil {
			return fmt.Errorf("failed to create gzip reader for router configuration file: %w", err)
		}
	}

	var newConfig router.RouterConfig

	// TODO move this check earlier
	if strings.Contains(r.configURL, ".yaml") {
		decoder := yaml.NewDecoder(reader)
		err = decoder.Decode(&newConfig)
	} else if strings.Contains(r.configURL, ".json") {
		err = json.NewDecoder(reader).Decode(&newConfig)
	} else {
		return fmt.Errorf("unsupported router configuration file type: %s", r.configURL)
	}

	if err != nil {
		return fmt.Errorf("failed to decode router configuration file: %w", err)
	}

	modelsToLoad := make(map[string]struct{})
	modelsToUnload := make(map[string]struct{})

	oldConfig := r.routerConfig
	if oldConfig != nil {
		for _, entity := range oldConfig.EntityMapping {
			modelsToUnload[entity.ModelName] = struct{}{}
		}

		if oldConfig.GlobalModelName != "" {
			modelsToUnload[oldConfig.GlobalModelName] = struct{}{}
		}
	}

	newModelMapping := make(map[int]string)
	for _, entity := range newConfig.EntityMapping {
		newModelMapping[entity.EntityID] = entity.ModelName
		delete(modelsToUnload, entity.ModelName)
		modelsToLoad[entity.ModelName] = struct{}{}
	}

	if newConfig.GlobalModelName != "" {
		if _, ok := modelsToUnload[newConfig.GlobalModelName]; ok {
			// don't unload
			delete(modelsToUnload, newConfig.GlobalModelName)
		} else {
			modelsToLoad[newConfig.GlobalModelName] = struct{}{}
		}
	}

	// Launch goroutines to load models concurrently, collecting errors.
	errCh := make(chan error, len(modelsToLoad))
	var wg sync.WaitGroup

	for model := range modelsToLoad {
		wg.Add(1)
		go func(model string) {
			defer wg.Done()
			err := r.tritonClient.ModelLoad(ctx, model)
			if err != nil {
				errCh <- fmt.Errorf("failed to load model %s: %w", model, err)
			}
		}(model)
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		var errStrings []string
		for err := range errCh {
			errStrings = append(errStrings, err.Error())
		}
		return fmt.Errorf("one or more model loading errors: %s", strings.Join(errStrings, "; "))
	}

	newRoutingTable := make(map[string]platform.PlatformEvaluator)
	for model := range modelsToLoad {
		evaluator, err := tricli.NewRoutedTritonEvaluator(
			model,
			r.tritonClient,
			r.modelConfig.Triton.Timeout,
			r.indexToName,
		)
		if err != nil {
			return fmt.Errorf("failed to create Triton evaluator for model %s: %w", model, err)
		}

		newRoutingTable[model] = evaluator
	}

	var globalEvaluator platform.PlatformEvaluator
	if newConfig.GlobalModelName != "" {
		globalEvaluator, err = tricli.NewRoutedTritonEvaluator(
			newConfig.GlobalModelName,
			r.tritonClient,
			r.modelConfig.Triton.Timeout,
			r.indexToName,
		)

		if err != nil {
			return fmt.Errorf("failed to create Triton evaluator for global model %s: %w", newConfig.GlobalModelName, err)
		}
	}

	// swap table
	func() {
		r.routingTableLock.Lock()
		defer r.routingTableLock.Unlock()
		r.globalModel = globalEvaluator
		r.routingMap = newModelMapping
		r.routerConfig = &newConfig
		r.routingTable = newRoutingTable
	}()

	// unload obsolete models, ignore errors...
	for model := range modelsToUnload {
		wg.Add(1)
		go func(model string) {
			defer wg.Done()
			err := r.tritonClient.ModelUnload(ctx, model)
			if err != nil {
				log.Printf("failed to unload model %s: %v\n", model, err)
			}
		}(model)
	}

	return nil
}

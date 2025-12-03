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
	"time"

	"github.com/viant/afs"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/files"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/service/request/shape"
	tricli "github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/config/router"
	"gopkg.in/yaml.v2"
)

type IOState struct {
	inputs    map[string]*domain.Input
	signature *domain.Signature

	// router input offset is the index of the router input in the inputs array
	routerInputOffset int
}

type UnloadService interface {
	UnloadModel(ctx context.Context, mlyModelID string, tritonModelName string) error
}

// Router implements the PlatformEvaluator interface for router mode.
type Router struct {
	configURL string
	fs        afs.Service

	// config lock only protects the configModified field
	configLock     sync.RWMutex
	configModified *config.Modified

	// routingTableLock protects:
	// - routerConfig
	// - routingMap
	// - routingTable
	// - globalModel
	// - ioState
	routingTableLock sync.RWMutex

	// routingConfig contains the last loaded routing configuration
	routingConfig *router.RoutingConfig

	hasGlobalModel      bool
	makeRoutedEvaluator func(modelName string) (platform.PlatformEvaluator, error)

	routerInputFieldName string

	routingMap   map[int]string
	routingTable map[string]platform.PlatformEvaluator

	// TODO see if this can be removed, may just need to map via model name
	globalModel platform.PlatformEvaluator

	// fixedEvaluator is non-nil IFF there is no global model configured
	fixedEvaluator platform.Predictor

	// fixedEvaluatorFields is for checking all outputs in the signature are replaced
	fixedEvaluatorFields map[string]struct{}
	outputConfig         config.OutputConfig

	workCh chan *workRequest

	modelOutputName string

	routerName string
	debug      bool
	unloader   UnloadService

	configuredInputs []*shared.Field
	ioState          *IOState
}

// NewRouter creates a new Router instance.
// cfg is expected to be Init()'d and Validate()'d before calling this function.
// makeEvaluator is expected to register usage for every created Evaluator.
func NewRouter(cfg *config.Model, fs afs.Service, tritonServices map[string]*tricli.Service, makeEvaluator func(modelName string) (platform.PlatformEvaluator, error)) (*Router, error) {
	unloaders := make(map[string]UnloadService)
	for serverID, tritonService := range tritonServices {
		unloaders[serverID] = tritonService
	}

	return newRouter(cfg, fs, unloaders, makeEvaluator)
}

// newRouter uses a map[string]ModelUnloader, where ModelUnloader is-a triton.TritonClient, for testing.
func newRouter(cfg *config.Model, fs afs.Service, unloaders map[string]UnloadService, makeEvaluator func(modelName string) (platform.PlatformEvaluator, error)) (*Router, error) {
	if cfg.Router == nil {
		return nil, fmt.Errorf("router configuration is required")
	}

	unloader, ok := unloaders[cfg.Triton.ServerID]
	if !ok {
		return nil, fmt.Errorf("triton client not found for server ID: %s", cfg.Triton.ServerID)
	}

	var fixedEvaluator *fixedEvaluator
	var fixedEvaluatorFields map[string]struct{}
	if !cfg.Router.Global.Exists {
		replacementsByName := make(map[string]config.PredictionReplacement)
		for _, repl := range cfg.Router.Global.PredictionReplacements {
			replacementsByName[repl.Name] = repl
		}

		var err error

		fixedEvaluator, err = newFixedEvaluator(cfg.Router.Global.PredictionReplacements)
		if err != nil {
			return nil, fmt.Errorf("failed to create fixed evaluator: %w", err)
		}

		fixedEvaluatorFields = make(map[string]struct{}, len(replacementsByName))
		for name := range replacementsByName {
			fixedEvaluatorFields[name] = struct{}{}
		}
	}

	r := &Router{
		debug:      cfg.Debug,
		routerName: cfg.ID,

		configURL:           cfg.Router.ConfigURL,
		fs:                  fs,
		makeRoutedEvaluator: makeEvaluator,

		unloader:       unloader,
		outputConfig:   cfg.Router.Output,
		hasGlobalModel: cfg.Router.Global.Exists,

		modelOutputName: cfg.Router.Output.FieldName,

		fixedEvaluator:       fixedEvaluator,
		fixedEvaluatorFields: fixedEvaluatorFields,

		configuredInputs:     cfg.Inputs,
		routerInputFieldName: cfg.Router.InputName,
	}

	// spawn worker routines
	r.workCh = make(chan *workRequest, cfg.Router.MaxQueueSize)
	for i := 0; i < cfg.Router.Workers; i++ {
		go handleWorkRequests(r.workCh, routerWorkerChannelQueuedSummary.WithLabelValues(r.routerName))
	}

	return r, nil
}

type preparedReplacement struct {
	typ   string
	value interface{}
}

// Predict performs model inference with the given parameters
// params is expected to be [numInputs]([batchSize][1]T) (see service/request.Request.Feeds)
func (r *Router) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	metricFixedOnly := true
	start := time.Now()
	defer func() {
		var fos string
		if metricFixedOnly {
			fos = "true"
		} else {
			fos = "false"
		}

		routerPredictDurationMicrosSummary.WithLabelValues(r.routerName, fos).Observe(float64(time.Since(start).Microseconds()))
	}()

	expectedBatchSize, err := shape.DetermineBatchSize(params)
	if err != nil {
		return nil, err
	}

	errCh := make(chan error, expectedBatchSize)
	resultsCh := make(chan offsetResults, expectedBatchSize)

	predictWaitGroup := sync.WaitGroup{}
	predictWaitGroup.Add(expectedBatchSize)

	var signature *domain.Signature

	err = func() error {
		r.routingTableLock.RLock()
		defer r.routingTableLock.RUnlock()

		if r.ioState == nil {
			return fmt.Errorf("ioState was not initialized")
		}

		// this assignment isn't strictly required to be atomic as it should never change after initialization
		signature = r.ioState.signature
		routerInputOffset := r.ioState.routerInputOffset

		globalExists := r.fixedEvaluator != nil

		reportedGlobalModelName := r.outputConfig.GlobalModelOverride
		noModelName := r.outputConfig.NoModelID

		numInputs := len(params)

		for batchOffset := range expectedBatchSize {
			// 1 input is reserved for the router input
			request := make([]interface{}, numInputs-1)

			var routingValueBatched interface{}

			// TODO support different input ordering per evaluator - see applyRouterConfig() regarding signatures
			for inputOffset := range numInputs {
				debatched, err := shape.Debatch(params[inputOffset], batchOffset)
				if err != nil {
					return fmt.Errorf("failed to debatch for row %d and input %d: %w", batchOffset, inputOffset, err)
				}

				if inputOffset < routerInputOffset {
					request[inputOffset] = debatched
				} else if inputOffset == routerInputOffset {
					routingValueBatched = debatched
				} else {
					request[inputOffset-1] = debatched
				}
			}

			routingValue, err := shape.SqueezeBatch(routingValueBatched)
			if err != nil {
				return fmt.Errorf("failed to extract from batch for row %d: %w", batchOffset, err)
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
				return fmt.Errorf("routing value is not an int: %v, is %T, for row %d", routingValue, routingValue, batchOffset)
			}

			routingValueString, ok := r.routingMap[routingValueInt]

			var evaluator platform.Predictor
			if !ok {
				if globalExists {
					metricFixedOnly = false
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
				metricFixedOnly = false

				var ok bool
				evaluator, ok = r.routingTable[routingValueString]
				if !ok {
					return fmt.Errorf("no evaluator found for routing value: %v", routingValue)
				}
			}

			select {
			case r.workCh <- &workRequest{
				wg: &predictWaitGroup,

				predictor: evaluator,
				ctx:       ctx,
				request:   request,

				queuedTime:         time.Now(),
				offset:             batchOffset,
				modelOutputEnabled: r.modelOutputName != "",
				routingValueString: routingValueString,

				responseCh: resultsCh,
				errCh:      errCh,
			}:
				// continue
			default:
				routerPredictDroppedCounter.WithLabelValues(r.routerName).Inc()
				return fmt.Errorf("work channel is full")
			}
		}

		return nil
	}()

	predictWaitGroup.Wait()
	close(errCh)
	close(resultsCh)

	if err != nil {
		return nil, err
	}

	for err := range errCh {
		return nil, err
	}

	allResults := make([][]interface{}, expectedBatchSize)
	for results := range resultsCh {
		allResults[results.offset] = results.results
	}

	endResults := make([]interface{}, len(signature.Outputs))
	for i, results := range allResults {
		endResults, err = shape.ConcatAxis0(endResults, results)
		if err != nil {
			return nil, fmt.Errorf("failed to concatenate results for row %d: %w", i, err)
		}
	}

	return endResults, nil
}

func (r *Router) Signature() *domain.Signature {
	r.routingTableLock.RLock()
	defer r.routingTableLock.RUnlock()
	return r.ioState.signature
}

func (r *Router) Dictionary() *common.Dictionary {
	return nil
}

func (r *Router) Inputs() map[string]*domain.Input {
	r.routingTableLock.RLock()
	defer r.routingTableLock.RUnlock()
	return r.ioState.inputs
}

func (r *Router) Stats(stats map[string]interface{}) {
	// do nothing
}

func (r *Router) Close() error {
	return nil
}

func (r *Router) debugLogf(format string, args ...interface{}) {
	if r.debug {
		prefix := "[%s Router] "
		log.Printf(prefix+format, append([]interface{}{r.routerName}, args...)...)
	}
}

type modelSignature struct {
	name      string
	signature *domain.Signature
}

// TODO refactor with service/tfmodel/service.isModified()?
func (r *Router) isModified(snapshot *config.Modified) bool {
	if r.routingConfig == nil || r.configModified == nil {
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
	start := time.Now()
	isFullReload := false
	defer func() {
		var mode string
		if isFullReload {
			mode = "full"
		} else {
			mode = "checks"
		}
		routerReloadDurationMicrosSummary.WithLabelValues(r.routerName, mode).Observe(float64(time.Since(start).Microseconds()))
	}()

	// fetch and check router configuration file
	snapshot, err := files.ModifiedSnapshot(ctx, r.fs, r.configURL, nil)
	if err != nil {
		return fmt.Errorf("failed to check router configuration file: %w", err)
	}

	if !r.isModified(snapshot) {
		// check health of all underlying models
		var wg sync.WaitGroup

		r.configLock.RLock()
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

			err = fmt.Errorf("reloading errors: %s", strings.Join(errStrings, "; "))
		}

		r.configLock.RUnlock()
		return err
	}

	isFullReload = true

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

	newConfig := new(router.RoutingConfig)

	// TODO move this check earlier
	if strings.Contains(r.configURL, ".yaml") {
		decoder := yaml.NewDecoder(reader)
		err = decoder.Decode(newConfig)
	} else if strings.Contains(r.configURL, ".json") {
		err = json.NewDecoder(reader).Decode(newConfig)
	} else {
		return fmt.Errorf("unsupported router configuration file type: %s", r.configURL)
	}

	if err != nil {
		return fmt.Errorf("failed to decode router configuration file: %w", err)
	}

	if err := r.applyRouterConfig(ctx, newConfig); err != nil {
		return err
	}

	return nil
}

// applyRouterConfig will both update evaluators to new configuration state and verify and build the signature
func (r *Router) applyRouterConfig(ctx context.Context, newConfig *router.RoutingConfig) error {
	modelsToUnload := make(map[string]struct{})
	reuseEvaluators := make(map[string]platform.PlatformEvaluator)
	var reuseGlobal platform.PlatformEvaluator

	var finalSignature *domain.Signature
	var oldConfig *router.RoutingConfig
	func() {
		r.routingTableLock.RLock()
		defer r.routingTableLock.RUnlock()
		if r.ioState != nil {
			finalSignature = r.ioState.signature
		}

		reuseGlobal = r.globalModel
		oldConfig = r.routingConfig
	}()

	if oldConfig != nil {
		for _, entity := range oldConfig.EntityMapping {
			modelsToUnload[entity.ModelName] = struct{}{}
			if evaluator, ok := r.routingTable[entity.ModelName]; ok {
				reuseEvaluators[entity.ModelName] = evaluator
			}
		}

		if oldConfig.GlobalModelName != "" {
			modelsToUnload[oldConfig.GlobalModelName] = struct{}{}
		}
	}

	newModelMapping := make(map[int]string)
	for _, entity := range newConfig.EntityMapping {
		r.debugLogf("add mapping: %d -> %s", entity.EntityID, entity.ModelName)

		newModelMapping[entity.EntityID] = entity.ModelName
		delete(modelsToUnload, entity.ModelName)
	}

	globalModelName := newConfig.GlobalModelName
	if globalModelName == "" && r.hasGlobalModel {
		return fmt.Errorf("global model name is missing")
	}

	if globalModelName != "" {
		r.debugLogf("global model: %s", globalModelName)
		delete(modelsToUnload, globalModelName)
	}

	newRoutingTable := make(map[string]platform.PlatformEvaluator)
	for _, entity := range newConfig.EntityMapping {
		model := entity.ModelName
		if _, ok := newRoutingTable[model]; ok {
			continue
		}

		if evaluator, ok := reuseEvaluators[model]; ok {
			newRoutingTable[model] = evaluator
			continue
		}

		evaluator, err := r.makeRoutedEvaluator(model)

		if err != nil {
			return fmt.Errorf("failed to create Routed Evaluator for model %s: %w", model, err)
		}

		newRoutingTable[model] = evaluator
	}

	var globalEvaluator platform.PlatformEvaluator
	if globalModelName != "" {
		if oldConfig != nil && globalModelName == oldConfig.GlobalModelName && reuseGlobal != nil {
			globalEvaluator = reuseGlobal
		} else if evaluator, ok := newRoutingTable[globalModelName]; ok {
			globalEvaluator = evaluator
		} else if evaluator, ok := reuseEvaluators[globalModelName]; ok {
			globalEvaluator = evaluator
		} else {
			var err error
			globalEvaluator, err = r.makeRoutedEvaluator(globalModelName)
			if err != nil {
				return fmt.Errorf("failed to create Routed Evaluator for global model %s: %w", globalModelName, err)
			}
		}
	}

	wg := sync.WaitGroup{}

	numWorkers := len(newRoutingTable)
	if globalEvaluator != nil {
		numWorkers++
	}

	errCh := make(chan error, numWorkers)
	signatureCh := make(chan modelSignature, numWorkers)

	if globalEvaluator != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := globalEvaluator.ReloadIfNeeded(ctx); err != nil {
				errCh <- fmt.Errorf("failed to reload global model %s: %w", globalModelName, err)
			}
		}()
	}

	for model := range newRoutingTable {
		wg.Add(1)
		go func(model string) {
			defer wg.Done()

			r.debugLogf("reload model: %s", model)

			modelEvaluator := newRoutingTable[model]
			if err := modelEvaluator.ReloadIfNeeded(ctx); err != nil {
				r.debugLogf("failed to reload model: %s: %v", model, err)
				errCh <- fmt.Errorf("failed to reload model %s: %w", model, err)
			}

			evalSig := modelEvaluator.Signature()

			if evalSig == nil {
				errCh <- fmt.Errorf("model %s signature is nil", model)
				return
			}

			signatureCh <- modelSignature{
				name:      model,
				signature: evalSig,
			}
		}(model)
	}

	r.debugLogf("wait for reloads")

	wg.Wait()
	close(errCh)
	close(signatureCh)

	if len(errCh) > 0 {
		var errStrings []string
		for err := range errCh {
			errStrings = append(errStrings, err.Error())
		}
		return fmt.Errorf("one or more model reloading errors: %s", strings.Join(errStrings, "; "))
	}

	sigInputMap := make(map[string]*domain.Input)
	sigOutputMap := make(map[string]*domain.Output)

	// we only create ioState on the first reload
	var ioState *IOState = new(IOState)
	if finalSignature != nil {
		for _, input := range finalSignature.Inputs {
			sigInputMap[input.Name] = &input
		}

		for _, output := range finalSignature.Outputs {
			sigOutputMap[output.Name] = &output
		}
	}

	for signature := range signatureCh {
		// accept first available signature as the final signature
		if finalSignature == nil {
			// in the creation of the signature, include the routing input
			// DANGER: this uses the pointer to the signature, so since the signature is modified, the original signature will be modified!
			// This doesn't happen in practice, but can cause issues in tests.
			finalSignature = signature.signature

			inputOffset := len(finalSignature.Inputs)

			routerInput := domain.Input{
				Name: r.routerInputFieldName,
				Type: reflect.TypeOf(int64(0)),
			}

			ioState.routerInputOffset = inputOffset

			finalSignature.Inputs = append(finalSignature.Inputs, routerInput)

			for _, input := range finalSignature.Inputs {
				sigInputMap[input.Name] = &input
			}

			for _, input := range r.configuredInputs {
				_, ok := sigInputMap[input.Name]

				if ok {
					// the input is configured and already in the self-reported signature
					continue
				}

				if !input.Auxiliary {
					return fmt.Errorf("non-auxiliary input %s for model %s was not in model inputs", input.Name, signature.name)
				}

				sigInputMap[input.Name] = &domain.Input{
					Name:      input.Name,
					Type:      input.RawType(),
					Auxiliary: input.Auxiliary,
				}
			}

			if r.modelOutputName != "" {
				// also, add the selected model output
				modelOutput := domain.Output{
					Name:     r.modelOutputName,
					Index:    len(finalSignature.Outputs),
					DataType: "string",
				}

				finalSignature.Outputs = append(finalSignature.Outputs, modelOutput)
			}

			for _, output := range finalSignature.Outputs {
				sigOutputMap[output.Name] = &output
			}

			continue
		}

		thisSignature := signature.signature
		// validate signature consistency
		thisSignatureOutputMap := make(map[string]*domain.Output)
		for _, output := range thisSignature.Outputs {
			oldOutput, ok := sigOutputMap[output.Name]
			if !ok {
				return fmt.Errorf("signature output %s for model %s not found in the previous signature", output.Name, signature.name)
			}

			thisSignatureOutputMap[output.Name] = &output

			// TODO permit this
			if oldOutput.Index != output.Index {
				return fmt.Errorf("signature output %s for model %s has index %d, and the previous signature has index %d", output.Name, signature.name, output.Index, oldOutput.Index)
			}

			if oldOutput.DataType != output.DataType {
				return fmt.Errorf("signature output %s for model %s has data type %s, and the previous signature has data type %s", output.Name, signature.name, output.DataType, oldOutput.DataType)
			}
		}

		for expectedOutput := range sigOutputMap {
			if _, ok := thisSignatureOutputMap[expectedOutput]; !ok && expectedOutput != r.modelOutputName {
				return fmt.Errorf("signature output %s for was not found in model %s signature", expectedOutput, signature.name)
			}
		}

		thisSignatureInputMap := make(map[string]*domain.Input)
		for _, input := range thisSignature.Inputs {
			oldInput, ok := sigInputMap[input.Name]
			if !ok {
				return fmt.Errorf("signature input %s for model %s not found in the previous signature", input.Name, signature.name)
			}

			thisSignatureInputMap[input.Name] = &input

			if oldInput.Auxiliary {
				continue
			}

			// TODO permit this
			if oldInput.Index != input.Index {
				return fmt.Errorf("signature input %s for model %s has index %d, and the previous signature has index %d", input.Name, signature.name, input.Index, oldInput.Index)
			}

			if !oldInput.Type.ConvertibleTo(input.Type) {
				return fmt.Errorf("signature input %s for model %s has data type %s, and the previous signature has data type %s", input.Name, signature.name, input.Type.String(), oldInput.Type.String())
			}
		}

		for expectedInput := range sigInputMap {
			if _, ok := thisSignatureInputMap[expectedInput]; !ok && expectedInput != r.routerInputFieldName {
				return fmt.Errorf("signature input %s for was not found in model %s signature", expectedInput, signature.name)
			}
		}
	}

	if r.fixedEvaluatorFields != nil {
		// TODO this is actually an acceptable case, but needs to be addressed elsewhere first before it is permitted
		for field := range r.fixedEvaluatorFields {
			if _, ok := sigOutputMap[field]; !ok {
				return fmt.Errorf("fixed evaluator field: %s was not found in the signature outputs", field)
			}
		}

		for _, field := range sigOutputMap {
			if _, ok := r.fixedEvaluatorFields[field.Name]; !ok && field.Name != r.modelOutputName {
				return fmt.Errorf("signature output %s is not replaced", field.Name)
			}
		}
	}

	ioState.signature = finalSignature
	ioState.inputs = sigInputMap

	if globalEvaluator != nil {
		if _, exists := newRoutingTable[globalModelName]; !exists {
			newRoutingTable[globalModelName] = globalEvaluator
		}
	}

	func() {
		r.routingTableLock.Lock()
		defer r.routingTableLock.Unlock()

		r.routingConfig = newConfig

		r.routingMap = newModelMapping
		r.routingTable = newRoutingTable

		r.globalModel = globalEvaluator

		if r.ioState == nil {
			r.ioState = ioState
		}
	}()

	for model := range modelsToUnload {
		routerModelUnloadGauge.WithLabelValues(r.routerName).Inc()

		go func(modelName string) {
			defer routerModelUnloadGauge.WithLabelValues(r.routerName).Dec()

			ctxTo, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			r.debugLogf("request to unload model: %s", modelName)

			if err := r.unloadModel(ctxTo, modelName); err != nil {
				r.debugLogf("failed to unload model %s: %v\n", modelName, err)
			}
		}(model)
	}

	return nil
}

func (r *Router) unloadModel(ctx context.Context, modelName string) error {
	defer routerModelUnloadGauge.WithLabelValues(r.routerName).Dec()
	if err := r.unloader.UnloadModel(ctx, r.routerName, modelName); err != nil {
		return fmt.Errorf("failed to unload model %s: %w", modelName, err)
	}
	return nil
}

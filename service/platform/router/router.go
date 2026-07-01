package router

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/afs"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/service/request/shape"
	tricli "github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/config/router"
)

const queueSizeExceededError = "queue size exceeded"

type IOState struct {
	inputs    map[string]*domain.Input
	signature *domain.Signature

	// router input offset is the index of the routing input in the inputs array
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
	fixedEvaluator *fixedEvaluator

	// fixedEvaluatorFields is for checking all outputs in the signature are replaced
	fixedEvaluatorFields map[string]struct{}
	outputConfig         config.OutputConfig

	modelOutputName string

	routerName  string
	debug       bool
	unloader    UnloadService
	unloadGauge prometheus.Gauge

	configuredInputs  []*shared.Field
	configuredOutputs []*shared.Field
	ioState           *IOState

	// forceBatchSize1 when true uses legacy per-sample dispatch; when false (default) uses batched dispatch
	forceBatchSize1 bool

	// workerSemaphore limits concurrent model evaluations
	workerSemaphore chan struct{}

	// maxQueueSize limits queued batches before rejection
	maxQueueSize uint64

	queued                *atomic.Uint64
	queueDurationObserver prometheus.Observer
	droppedCounter        prometheus.Counter
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
	rtCfg := cfg.Router
	if rtCfg == nil {
		return nil, fmt.Errorf("router configuration is required")
	}

	unloader, ok := unloaders[cfg.Triton.ServerID]
	if !ok {
		return nil, fmt.Errorf("triton client not found for server ID: %s", cfg.Triton.ServerID)
	}

	var fixedEvaluator *fixedEvaluator
	var fixedEvaluatorFields map[string]struct{}
	if !rtCfg.Global.Exists {
		replacementsByName := make(map[string]config.PredictionReplacement)
		for _, repl := range rtCfg.Global.PredictionReplacements {
			replacementsByName[repl.Name] = repl
		}

		var err error

		fixedEvaluator, err = newFixedEvaluator(rtCfg.Global.PredictionReplacements)
		if err != nil {
			return nil, fmt.Errorf("failed to create fixed evaluator: %w", err)
		}

		fixedEvaluatorFields = make(map[string]struct{}, len(replacementsByName))
		for name := range replacementsByName {
			fixedEvaluatorFields[name] = struct{}{}
		}
	}

	routerName := cfg.ID
	r := &Router{
		debug:      cfg.Debug,
		routerName: routerName,

		configURL:           rtCfg.ConfigURL,
		fs:                  fs,
		makeRoutedEvaluator: makeEvaluator,

		unloader:    unloader,
		unloadGauge: routerModelUnloadGauge.WithLabelValues(routerName),

		outputConfig:   rtCfg.Output,
		hasGlobalModel: rtCfg.Global.Exists,

		modelOutputName: rtCfg.Output.FieldName,

		fixedEvaluator:       fixedEvaluator,
		fixedEvaluatorFields: fixedEvaluatorFields,

		configuredInputs:     cfg.Inputs,
		configuredOutputs:    cfg.Outputs,
		routerInputFieldName: rtCfg.InputName,

		forceBatchSize1: rtCfg.ForceBatchSize1,

		workerSemaphore: make(chan struct{}, rtCfg.Workers),
		maxQueueSize:    uint64(rtCfg.MaxQueueSize),
		queued:          &atomic.Uint64{},

		queueDurationObserver: routerQueueDurationMicrosSummary.WithLabelValues(routerName),
		droppedCounter:        routerPredictDroppedCounter.WithLabelValues(routerName),
	}

	return r, nil
}

// modelBatch holds accumulated rows destined for a single model evaluator
type modelBatch struct {
	evaluator    platform.PlatformEvaluator // need Signature() for input reordering
	isFixedEval  bool                       // true skips input reordering
	modelName    string
	inputsByName map[string]interface{} // keyed by input name - accumulated batched inputs
	rowOffsets   []int                  // original positions in the incoming batch
}

// batchResult holds the result from a batched model prediction
type batchResult struct {
	modelName   string
	results     []interface{}
	offsets     []int
	err         error
	outputNames []string // output names in the order returned by evaluator (for reordering)
}

// Predict performs model inference with the given parameters.
// params is expected to be [numInputs]([batchSize][1]T) (see service/request.Request.Feeds).
//
// Rows are grouped into batches based on their target model evaluator.
// When forceBatchSize1 is true, each row forms its own batch (batch size 1).
// When forceBatchSize1 is false (default), rows destined for the same model are batched together.
func (r *Router) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no input parameters provided")
	}

	// metricFixedOnly is true if the request is only using the fixedEvaluator
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

	r.routingTableLock.RLock()
	defer r.routingTableLock.RUnlock()

	var signature *domain.Signature
	batches := make(map[string]*modelBatch)

	// Phase 1: Group rows into batches by model name
	err = func() error {
		if r.ioState == nil {
			return fmt.Errorf("ioState was not initialized")
		}

		signature = r.ioState.signature
		routerInputOffset := r.ioState.routerInputOffset

		hasFixedEvaluator := r.fixedEvaluator != nil
		// Prefer the global model name from the live router config (router.yaml)
		// so the reported inference_model_id reflects the actual global model
		// artifact. Falls back to the static Output.GlobalModelOverride.
		reportedGlobalModelName := r.outputConfig.GlobalModelOverride
		if r.routingConfig != nil && r.routingConfig.GlobalModelName != "" {
			reportedGlobalModelName = r.routingConfig.GlobalModelName
		}
		noModelName := r.outputConfig.NoModelID

		numInputs := len(params)

		routerInputBatch := params[routerInputOffset]
		for batchOffset := range expectedBatchSize {
			// Extract routing value for this row
			routingValueBatched, err := shape.Debatch(routerInputBatch, batchOffset)
			if err != nil {
				return fmt.Errorf("failed to debatch routing value for row %d: %w", batchOffset, err)
			}

			routingValue, err := shape.SqueezeBatch(routingValueBatched)
			if err != nil {
				return fmt.Errorf("failed to extract routing value for row %d: %w", batchOffset, err)
			}

			var routingValueInt int
			switch rv := routingValue.(type) {
			case int:
				routingValueInt = rv
			case int32:
				routingValueInt = int(rv)
			case int64:
				routingValueInt = int(rv)
			default:
				return fmt.Errorf("routing value is not an int: %v, is %T, for row %d", routingValue, routingValue, batchOffset)
			}

			routingValueString, ok := r.routingMap[routingValueInt]

			var evaluator platform.PlatformEvaluator
			isFixedEval := false
			if !ok {
				if hasFixedEvaluator {
					// No global model, use fixed evaluator
					routingValueString = noModelName
					isFixedEval = true
				} else {
					metricFixedOnly = false
					evaluator = r.globalModel
					if reportedGlobalModelName != "" {
						routingValueString = reportedGlobalModelName
					}
				}
			} else {
				metricFixedOnly = false
				evaluator, ok = r.routingTable[routingValueString]
				if !ok {
					return fmt.Errorf("no evaluator found for routing value: %v", routingValue)
				}
			}

			// Determine batch key: unique per row when forceBatchSize1, otherwise by model name
			batchKey := routingValueString
			if r.forceBatchSize1 {
				batchKey = strconv.Itoa(batchOffset)
			}

			batch, exists := batches[batchKey]
			if !exists {
				batch = &modelBatch{
					evaluator:    evaluator,
					isFixedEval:  isFixedEval,
					modelName:    routingValueString,
					inputsByName: make(map[string]interface{}),
					rowOffsets:   make([]int, 0, 1),
				}

				batches[batchKey] = batch
			}

			// Append this row's inputs to the batch (excluding router input)
			for paramOffset := range numInputs {
				if paramOffset == routerInputOffset {
					continue
				}

				inputName := signature.Inputs[paramOffset].Name
				debatched, err := shape.Debatch(params[paramOffset], batchOffset)
				if err != nil {
					return fmt.Errorf("failed to debatch for row %d, input %s: %w", batchOffset, inputName, err)
				}

				batch.inputsByName[inputName], err = shape.AppendRowToBatch(batch.inputsByName[inputName], debatched)
				if err != nil {
					return fmt.Errorf("failed to append row %d to batch for input %s: %w", batchOffset, inputName, err)
				}
			}

			batch.rowOffsets = append(batch.rowOffsets, batchOffset)
		}

		return nil
	}()

	if err != nil {
		return nil, err
	}

	// early queue size check
	currentQ := r.queued.Load()
	if uint64(len(batches))+currentQ > r.maxQueueSize {
		r.droppedCounter.Inc()
		return nil, fmt.Errorf(queueSizeExceededError)
	}

	// Phase 2: Execute predictions in parallel with bounded concurrency
	resultCh := make(chan batchResult, len(batches))
	var wg sync.WaitGroup

	for _, batch := range batches {
		wg.Add(1)

		// this must be decremented if queue is full and once no longer in queue
		nowQueued := r.queued.Add(1)
		startQueueTime := time.Now()

		if nowQueued > r.maxQueueSize {
			r.queued.Add(^uint64(0))
			r.droppedCounter.Inc()
			return nil, fmt.Errorf(queueSizeExceededError)
		}

		go func(b *modelBatch) {
			defer wg.Done()

			// Acquire semaphore slot
			r.workerSemaphore <- struct{}{}

			r.queued.Add(^uint64(0))
			r.queueDurationObserver.Observe(float64(time.Since(startQueueTime).Microseconds()))

			defer func() {
				<-r.workerSemaphore
			}()

			// Reorder inputs to match each evaluator's expected order before calling Predict
			var results []interface{}
			var err error

			// Capture output names for reordering in Phase 3
			var outputNames []string
			bs := len(b.rowOffsets)

			if b.isFixedEval {
				results, err = r.fixedEvaluator.Predict(bs)
				outputNames = r.fixedEvaluator.OutputNames()
			} else {
				// Reorder inputs to match this evaluator's expected order
				evalSig := b.evaluator.Signature()
				orderedInputs := make([]interface{}, len(evalSig.Inputs))
				for i, sigInput := range evalSig.Inputs {
					inputData, exists := b.inputsByName[sigInput.Name]
					if !exists {
						err = fmt.Errorf("input %s not found in batch for model %s", sigInput.Name, b.modelName)
						break
					}
					orderedInputs[i] = inputData
				}

				if err == nil {
					// Rely on downstream for timeouts
					results, err = b.evaluator.Predict(ctx, orderedInputs)

					outputNames = make([]string, len(evalSig.Outputs))
					for i, out := range evalSig.Outputs {
						outputNames[i] = out.Name
					}
				}
			}

			// Append model name to results if configured
			if r.modelOutputName != "" && err == nil {
				modelNames := make([][]string, bs)
				for i := range modelNames {
					modelNames[i] = []string{b.modelName}
				}

				results = append(results, modelNames)
				outputNames = append(outputNames, r.modelOutputName)
			}

			resultCh <- batchResult{
				modelName:   b.modelName,
				results:     results,
				offsets:     b.rowOffsets,
				err:         err,
				outputNames: outputNames,
			}
		}(batch)
	}

	wg.Wait()
	close(resultCh)

	// Phase 3: Reassemble results in original order
	// Build router output name -> index mapping for reordering
	// TODO see if memoizing this provides material performance boosts
	routerOutputIndex := make(map[string]int, len(signature.Outputs))
	for i, out := range signature.Outputs {
		routerOutputIndex[out.Name] = i
	}

	// allResults will be [expectedBatchSize][len(signature.Outputs)]
	allResults := make([][]interface{}, expectedBatchSize)

	for res := range resultCh {
		if res.err != nil {
			return nil, fmt.Errorf("prediction failed for model %s: %w", res.modelName, res.err)
		}

		// Extract individual rows from the batched result and place at original offsets
		for evalOffset, originalOffset := range res.offsets {
			rowResult := make([]interface{}, len(signature.Outputs))

			// Reorder outputs to match router's expected output order
			for evalOutputIdx, outputBatch := range res.results {
				extracted, err := shape.ExtractRowFromBatch(outputBatch, evalOffset)
				if err != nil {
					return nil, fmt.Errorf("failed to extract row %d from model %s output index %d: %w",
						evalOffset, res.modelName, evalOutputIdx, err)
				}

				// Map evaluator output index to router output index by name
				var originalOutputIdx int
				if res.outputNames == nil {
					// Fallback: assume same order (shouldn't happen in normal operation)
					originalOutputIdx = evalOutputIdx
				} else {
					outputName := res.outputNames[evalOutputIdx]

					var exists bool
					originalOutputIdx, exists = routerOutputIndex[outputName]
					if !exists {
						return nil, fmt.Errorf("output %s from model %s not found in router signature",
							outputName, res.modelName)
					}
				}

				rowResult[originalOutputIdx] = extracted
			}

			allResults[originalOffset] = rowResult
		}
	}

	// Reshape all values into [outputs][batch][M]
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

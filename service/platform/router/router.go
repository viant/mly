package router

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

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

	modelOutputName string

	routerName string
	debug      bool
	unloader   UnloadService

	configuredInputs []*shared.Field
	ioState          *IOState

	// forceBatchSize1 when true uses legacy per-sample dispatch; when false (default) uses batched dispatch
	forceBatchSize1 bool
	// workers limits concurrent model evaluations (used as semaphore capacity in batch mode)
	workers int
	// maxQueueSize limits queued batches before rejection
	maxQueueSize int
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

		forceBatchSize1: cfg.Router.ForceBatchSize1,
		workers:         cfg.Router.Workers,
		maxQueueSize:    cfg.Router.MaxQueueSize,
	}

	return r, nil
}

type preparedReplacement struct {
	typ   string
	value interface{}
}

// modelBatch holds accumulated rows destined for a single model evaluator
type modelBatch struct {
	evaluator  platform.Predictor
	modelName  string
	inputs     []interface{} // [numInputs][]interface{} - accumulated batched inputs
	rowOffsets []int         // original positions in the incoming batch
}

// batchResult holds the result from a batched model prediction
type batchResult struct {
	modelName string
	results   []interface{}
	offsets   []int
	err       error
}

// Predict performs model inference with the given parameters.
// params is expected to be [numInputs]([batchSize][1]T) (see service/request.Request.Feeds).
//
// Rows are grouped into batches based on their target model evaluator.
// When ForceBatchSize1 is true, each row forms its own batch (batch size 1).
// When ForceBatchSize1 is false (default), rows destined for the same model are grouped together.
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

	var signature *domain.Signature
	batches := make(map[string]*modelBatch)

	// Hold read lock to ensure evaluator references remain valid during prediction.
	r.routingTableLock.RLock()
	defer r.routingTableLock.RUnlock()

	// Phase 1: Group rows into batches
	// When forceBatchSize1 is true, each row gets a unique batch key (row offset as string)
	// When false, rows are grouped by model name
	err = func() error {
		if r.ioState == nil {
			return fmt.Errorf("ioState was not initialized")
		}

		signature = r.ioState.signature
		routerInputOffset := r.ioState.routerInputOffset

		globalExists := r.fixedEvaluator != nil
		reportedGlobalModelName := r.outputConfig.GlobalModelOverride
		noModelName := r.outputConfig.NoModelID

		numInputs := len(params)
		numModelInputs := numInputs - 1 // exclude router input

		for batchOffset := range expectedBatchSize {
			// Extract routing value for this row
			routingValueBatched, err := shape.Debatch(params[routerInputOffset], batchOffset)
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

			var evaluator platform.Predictor
			if !ok {
				if globalExists {
					metricFixedOnly = false
					evaluator = r.globalModel
					if reportedGlobalModelName != "" {
						routingValueString = reportedGlobalModelName
					}
				} else {
					routingValueString = noModelName
					evaluator = r.fixedEvaluator
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
				batchKey = routingValueString + "#" + strconv.Itoa(batchOffset)
			}

			// Get or create batch
			batch, exists := batches[batchKey]
			if !exists {
				batch = &modelBatch{
					evaluator:  evaluator,
					modelName:  routingValueString,
					inputs:     make([]interface{}, numModelInputs),
					rowOffsets: make([]int, 0, 1),
				}
				batches[batchKey] = batch
			}

			// Append this row's inputs to the batch (excluding router input)
			inputIdx := 0
			for paramOffset := range numInputs {
				if paramOffset == routerInputOffset {
					continue
				}

				debatched, err := shape.Debatch(params[paramOffset], batchOffset)
				if err != nil {
					return fmt.Errorf("failed to debatch for row %d, input %d: %w", batchOffset, paramOffset, err)
				}

				batch.inputs[inputIdx], err = shape.AppendRowToBatch(batch.inputs[inputIdx], debatched)
				if err != nil {
					return fmt.Errorf("failed to append row %d to batch for input %d: %w", batchOffset, paramOffset, err)
				}
				inputIdx++
			}

			batch.rowOffsets = append(batch.rowOffsets, batchOffset)
		}

		return nil
	}()

	if err != nil {
		return nil, err
	}

	// Check queue size limit
	if len(batches) > r.maxQueueSize {
		routerPredictDroppedCounter.WithLabelValues(r.routerName).Inc()
		return nil, fmt.Errorf("too many batches (%d) exceeds max queue size (%d)", len(batches), r.maxQueueSize)
	}

	// Phase 2: Execute predictions in parallel with bounded concurrency
	resultCh := make(chan batchResult, len(batches))
	semaphore := make(chan struct{}, r.workers)
	var wg sync.WaitGroup

	for _, batch := range batches {
		wg.Add(1)
		go func(b *modelBatch) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Rely on downstream for timeouts
			results, err := b.evaluator.Predict(ctx, b.inputs)

			// Append model name to results if configured
			if r.modelOutputName != "" && err == nil {
				modelNames := make([][]string, len(b.rowOffsets))
				for i := range modelNames {
					modelNames[i] = []string{b.modelName}
				}
				results = append(results, modelNames)
			}

			resultCh <- batchResult{
				modelName: b.modelName,
				results:   results,
				offsets:   b.rowOffsets,
				err:       err,
			}
		}(batch)
	}

	wg.Wait()
	close(resultCh)

	// Phase 3: Reassemble results in original order
	allResults := make([][]interface{}, expectedBatchSize)

	for res := range resultCh {
		if res.err != nil {
			return nil, fmt.Errorf("prediction failed for model %s: %w", res.modelName, res.err)
		}

		// Extract individual rows from the batched result and place at original offsets
		for localIdx, originalOffset := range res.offsets {
			rowResult := make([]interface{}, len(res.results))
			for outputIdx, outputBatch := range res.results {
				extracted, err := shape.ExtractRowFromBatch(outputBatch, localIdx)
				if err != nil {
					return nil, fmt.Errorf("failed to extract row %d from model %s output %d: %w",
						localIdx, res.modelName, outputIdx, err)
				}
				rowResult[outputIdx] = extracted
			}
			allResults[originalOffset] = rowResult
		}
	}

	// Concatenate all rows into final output
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

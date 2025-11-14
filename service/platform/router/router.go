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
	"github.com/viant/mly/shared"
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
	globalModel      *platform.PlatformEvaluator
	fixedEvaluator   Predictor

	modelConfig  *config.Model
	tritonClient tricli.TritonClient

	signature   *domain.Signature
	indexToName map[int]string

	inputs map[string]*domain.Input

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

	if !cfg.Router.Global.Exists {
		r.fixedEvaluator = newFixedEvaluator(cfg.Router.Global.PredictionReplacements)
	}

	r.handleIO(&cfg.MetaInput)

	return r, nil
}

type Predictor interface {
	Predict(ctx context.Context, params []interface{}) ([]interface{}, error)
}

type fixedEvaluator struct {
	prepared []preparedReplacement
}

type preparedReplacement struct {
	typ   string
	value interface{}
}

func newFixedEvaluator(repls []config.PredictionReplacement) *fixedEvaluator {
	prepared := make([]preparedReplacement, 0, len(repls))
	for _, r := range repls {
		switch r.Type {
		case "string":
			v, ok := r.Value.(string)
			if !ok {
				v = fmt.Sprintf("%v", r.Value)
			}
			prepared = append(prepared, preparedReplacement{typ: "string", value: v})
		case "int":
			switch n := r.Value.(type) {
			case int:
				prepared = append(prepared, preparedReplacement{typ: "int", value: n})
			case int32:
				prepared = append(prepared, preparedReplacement{typ: "int", value: int(n)})
			case int64:
				prepared = append(prepared, preparedReplacement{typ: "int", value: int(n)})
			case float32:
				prepared = append(prepared, preparedReplacement{typ: "int", value: int(n)})
			case float64:
				prepared = append(prepared, preparedReplacement{typ: "int", value: int(n)})
			default:
				panic(fmt.Errorf("router replacement %q: value %T not coercible to int", r.Name, r.Value))
			}
		case "int32":
			switch n := r.Value.(type) {
			case int:
				prepared = append(prepared, preparedReplacement{typ: "int32", value: int32(n)})
			case int32:
				prepared = append(prepared, preparedReplacement{typ: "int32", value: n})
			case int64:
				prepared = append(prepared, preparedReplacement{typ: "int32", value: int32(n)})
			case float32:
				prepared = append(prepared, preparedReplacement{typ: "int32", value: int32(n)})
			case float64:
				prepared = append(prepared, preparedReplacement{typ: "int32", value: int32(n)})
			default:
				panic(fmt.Errorf("router replacement %q: value %T not coercible to int32", r.Name, r.Value))
			}
		case "int64":
			switch n := r.Value.(type) {
			case int:
				prepared = append(prepared, preparedReplacement{typ: "int64", value: int64(n)})
			case int32:
				prepared = append(prepared, preparedReplacement{typ: "int64", value: int64(n)})
			case int64:
				prepared = append(prepared, preparedReplacement{typ: "int64", value: n})
			case float32:
				prepared = append(prepared, preparedReplacement{typ: "int64", value: int64(n)})
			case float64:
				prepared = append(prepared, preparedReplacement{typ: "int64", value: int64(n)})
			default:
				panic(fmt.Errorf("router replacement %q: value %T not coercible to int64", r.Name, r.Value))
			}
		case "float", "float32":
			switch n := r.Value.(type) {
			case int:
				prepared = append(prepared, preparedReplacement{typ: "float32", value: float32(n)})
			case int32:
				prepared = append(prepared, preparedReplacement{typ: "float32", value: float32(n)})
			case int64:
				prepared = append(prepared, preparedReplacement{typ: "float32", value: float32(n)})
			case float32:
				prepared = append(prepared, preparedReplacement{typ: "float32", value: n})
			case float64:
				prepared = append(prepared, preparedReplacement{typ: "float32", value: float32(n)})
			default:
				panic(fmt.Errorf("router replacement %q: value %T not coercible to float32", r.Name, r.Value))
			}
		case "float64":
			switch n := r.Value.(type) {
			case int:
				prepared = append(prepared, preparedReplacement{typ: "float64", value: float64(n)})
			case int32:
				prepared = append(prepared, preparedReplacement{typ: "float64", value: float64(n)})
			case int64:
				prepared = append(prepared, preparedReplacement{typ: "float64", value: float64(n)})
			case float32:
				prepared = append(prepared, preparedReplacement{typ: "float64", value: float64(n)})
			case float64:
				prepared = append(prepared, preparedReplacement{typ: "float64", value: n})
			default:
				panic(fmt.Errorf("router replacement %q: value %T not coercible to float64", r.Name, r.Value))
			}
		default:
			panic(fmt.Errorf("unsupported router replacement type %q for %q", r.Type, r.Name))
		}
	}
	return &fixedEvaluator{prepared: prepared}
}

func (f *fixedEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	batchSize, err := shape.DetermineBatchSize(params)
	if err != nil {
		return nil, err
	}

	makeString := func(v string) [][]string {
		out := make([][]string, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []string{v}
		}
		return out
	}

	makeInt32 := func(v int32) [][]int32 {
		out := make([][]int32, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []int32{v}
		}
		return out
	}

	makeInt64 := func(v int64) [][]int64 {
		out := make([][]int64, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []int64{v}
		}
		return out
	}

	makeInt := func(v int) [][]int {
		out := make([][]int, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []int{v}
		}
		return out
	}

	makeFloat32 := func(v float32) [][]float32 {
		out := make([][]float32, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []float32{v}
		}
		return out
	}

	makeFloat64 := func(v float64) [][]float64 {
		out := make([][]float64, batchSize)
		for i := 0; i < batchSize; i++ {
			out[i] = []float64{v}
		}
		return out
	}

	results := make([]interface{}, len(f.prepared))
	for i, repl := range f.prepared {
		switch repl.typ {
		case "string":
			results[i] = makeString(repl.value.(string))
		case "int":
			results[i] = makeInt(repl.value.(int))
		case "int32":
			results[i] = makeInt32(repl.value.(int32))
		case "int64":
			results[i] = makeInt64(repl.value.(int64))
		case "float":
			results[i] = makeFloat32(repl.value.(float32))
		case "float32":
			results[i] = makeFloat32(repl.value.(float32))
		case "float64":
			results[i] = makeFloat64(repl.value.(float64))
		default:
			return nil, fmt.Errorf("unsupported replacement type %q", repl.typ)
		}
	}

	return results, nil
}

func (t *Router) handleIO(io *shared.MetaInput) error {
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

	allResults := make([]interface{}, 0, expectedBatchSize)

	r.routingTableLock.RLock()
	defer r.routingTableLock.RUnlock()

	for batchOffset := range expectedBatchSize {
		// 1 input is reserved for the router input
		request := make([]interface{}, numInputs-1)

		var routingValueBatched interface{}

		for inputOffset := range numInputs {
			debatched, err := debatch(params[inputOffset], batchOffset)
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

		routingValue, err := squeezeBatch(routingValueBatched)
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

		var evaluator Predictor
		if !ok {
			if r.modelConfig.Router.Global.Exists {
				// fallback to global model
				evaluator = *r.globalModel
			} else {
				evaluator = r.fixedEvaluator
			}
		} else {
			var ok bool
			evaluator, ok = r.routingTable[routingValueString]
			if !ok {
				return nil, fmt.Errorf("no evaluator found for routing value: %v", routingValue)
			}
		}

		results, err := evaluator.Predict(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("failed to predict for row %d: %w", batchOffset, err)
		}

		// TODO dynamic output batch shape detection
		allResults = append(allResults, results)
	}

	// TODO this is going to crash
	return allResults, nil
}

func squeezeBatch(untypedBatch interface{}) (interface{}, error) {
	switch typedBatch := untypedBatch.(type) {
	case [][]int32:
		return typedBatch[0][0], nil
	case [][]int64:
		return typedBatch[0][0], nil
	case [][]float32:
		return typedBatch[0][0], nil
	case [][]float64:
		return typedBatch[0][0], nil
	case [][]string:
		return typedBatch[0][0], nil
	}

	return nil, fmt.Errorf("unexpected batch type: %T", untypedBatch)
}

func debatch(untypedBatch interface{}, i int) (interface{}, error) {
	switch typedBatch := untypedBatch.(type) {
	case [][]int32:
		return [][]int32{{typedBatch[0][i]}}, nil
	case [][]int64:
		return [][]int64{{typedBatch[0][i]}}, nil
	case [][]float32:
		return [][]float32{{typedBatch[0][i]}}, nil
	case [][]float64:
		return [][]float64{{typedBatch[0][i]}}, nil
	case [][]string:
		return [][]string{{typedBatch[0][i]}}, nil
	}

	return nil, fmt.Errorf("unexpected batch type: %T", untypedBatch)
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
		var wg sync.WaitGroup
		errCh := make(chan error, len(r.routingTable))

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

	for _, entity := range newConfig.EntityMapping {
		if _, ok := modelsToUnload[entity.ModelName]; ok {
			// don't unload
			delete(modelsToUnload, entity.ModelName)
		} else {
			modelsToLoad[entity.ModelName] = struct{}{}
		}
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
		evaluator, err := tricli.NewTritonEvaluator(
			r.modelConfig,
			map[string]tricli.TritonClient{r.modelConfig.Triton.ServerID: r.tritonClient},
		)
		if err != nil {
			return fmt.Errorf("failed to create Triton evaluator for model %s: %w", model, err)
		}
		newRoutingTable[model] = evaluator
	}

	// swap table
	func() {
		r.routingTableLock.Lock()
		defer r.routingTableLock.Unlock()
		r.routerConfig = &newConfig
		r.routingTable = newRoutingTable
	}()

	// unload obsolete models, ignore errors...
	for model := range modelsToUnload {
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

package router

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/service/request/shape"
	"github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared/common"
)

// mockEvaluator currently handles all test case behaviors.
// This may be a sign that the router itself needs to be refactored into separate parts.
type mockEvaluator struct {
	// reload-related objects
	modelName string

	// see reload_test.go
	tritonServer *mockTritonServer

	// used in batching tests
	predictCalls int
	mu           sync.Mutex

	// force an error
	err error

	// for queueing tests
	waitFor   *sync.WaitGroup
	doneGroup *sync.WaitGroup

	predictor func(params []interface{}, signature *domain.Signature) ([]interface{}, error)
	signature func() *domain.Signature
}

func (m *mockEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	m.mu.Lock()
	m.predictCalls++
	m.mu.Unlock()

	if m.waitFor != nil {
		m.waitFor.Wait()
		m.waitFor = nil
	}

	if m.doneGroup != nil {
		defer m.doneGroup.Done()
		m.doneGroup = nil
	}

	if m.err != nil {
		return nil, m.err
	}

	sig := m.signature()
	inputs := sig.Inputs
	if len(inputs) != len(params) {
		return nil, fmt.Errorf("mock error: expected %d inputs, got %d", len(inputs), len(params))
	}

	if m.predictor == nil {
		// for test cases that do not validate results like reload-centric cases
		return nil, nil
	}

	return m.predictor(params, sig)
}

func (m *mockEvaluator) Signature() *domain.Signature     { return m.signature() }
func (m *mockEvaluator) Dictionary() *common.Dictionary   { return nil }
func (m *mockEvaluator) Inputs() map[string]*domain.Input { return nil }
func (m *mockEvaluator) Stats(map[string]interface{})     {}
func (m *mockEvaluator) Close() error                     { return nil }

func (m *mockEvaluator) ReloadIfNeeded(ctx context.Context) error {
	if m.tritonServer == nil {
		return nil
	}

	return m.tritonServer.ModelLoad(ctx, m.modelName)
}

func (m *mockEvaluator) getPredictCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.predictCalls
}

// appendFloatPredict expects []interface{[][]string} and returns []interface{}{[][]float32}
// The output field names are expected to be an integer in string form, which will be parsed and appended.
func appendFloatPredict(params []interface{}, signature *domain.Signature) ([]interface{}, error) {
	batchSize, err := shape.BatchSize(params[0])
	if err != nil {
		return nil, fmt.Errorf("could not determine batch size: %w", err)
	}

	batchConcats := make([]*strings.Builder, batchSize)
	for _, param := range params {
		switch batch := param.(type) {
		case [][]string:
			for bi, s := range batch {
				if batchConcats[bi] == nil {
					batchConcats[bi] = &strings.Builder{}
				}

				sb := batchConcats[bi]
				sb.WriteString(s[0])
			}
		default:
			return nil, fmt.Errorf("unexpected input type: %T", param)
		}
	}

	numOutputs := len(signature.Outputs)
	retVal := make([]interface{}, numOutputs)
	for oi := range numOutputs {
		outputBatch := make([][]float32, batchSize)
		for i, v := range batchConcats {
			outputName := signature.Outputs[oi].Name
			floatStr := outputName + v.String()
			floatVal, err := strconv.ParseFloat(floatStr, 32)
			if err != nil {
				return nil, fmt.Errorf("could not parse float: %v", floatStr)
			}

			outputBatch[i] = []float32{float32(floatVal)}
		}

		retVal[oi] = outputBatch
	}

	return retVal, nil
}

type configVariant struct {
	numOutputs       int
	hasGlobalModel   bool
	hasOutputName    bool
	forceBatchSize1  bool
	reverseFPOutputs bool
}

func makeConfig(cv configVariant) *config.RouterConfig {
	var prs []config.PredictionReplacement
	if !cv.hasGlobalModel {
		prs = make([]config.PredictionReplacement, cv.numOutputs)
		for i := range cv.numOutputs {
			prs[i] = config.PredictionReplacement{
				Name:  strconv.Itoa(i),
				Type:  "float32",
				Value: float32(i) + 0.1,
			}
		}
	}

	if cv.reverseFPOutputs {
		slices.Reverse(prs)
	}

	var outputName = ""
	if cv.hasOutputName {
		outputName = "model_id"
	}

	gmo := ""
	if cv.hasGlobalModel {
		gmo = "global"
	}

	rc := &config.RouterConfig{
		InputName:       "router_id",
		ConfigURL:       "memory://router-config",
		ForceBatchSize1: cv.forceBatchSize1,
		Global: config.GlobalModelConfig{
			Exists:                 cv.hasGlobalModel,
			PredictionReplacements: prs,
		},
		Output: config.OutputConfig{
			FieldName:           outputName,
			GlobalModelOverride: gmo,
		},
		MaxQueueSize: 100,
		Workers:      3,
	}

	return rc
}

type predictTestCase struct {
	name         string
	routerConfig *config.RouterConfig

	reverseInputs  bool
	reverseOutputs bool

	stringInputs [][]string
	routerInputs []int

	// outputs
	expectedOutputs  [][]float32
	expectCallCounts map[string]int
}

// predictTest creates the signature and evaluators, runs Predict, and runs the tests
func predictTest(t *testing.T, test predictTestCase) {
	routerInputs, router, mockEvaluators, globalModelName, params := prepareTestRouter(t, test)

	dl, _ := t.Deadline()
	ctx, cancel := context.WithDeadline(context.Background(), dl)
	defer cancel()
	results, err := router.Predict(ctx, params)
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	// model name is appended at the end, first check normal outputs
	for oi, outputBatch := range test.expectedOutputs {
		actualOutput := results[oi]
		switch aob := actualOutput.(type) {
		case [][]float32:
			for obi, ov := range outputBatch {
				actualValue := aob[obi][0]
				log.Printf("model output %d expected:%f actual:%f", obi, ov, actualValue)

				if actualValue != ov {
					t.Fatalf("input %d offset %d expected %f, got %f", oi, obi, ov, actualValue)
				}
			}
		default:
			t.Fatalf("input %d expected [][]float32, got %T", oi, actualOutput)
		}
	}

	// then check model name outputs
	if test.routerConfig.Output.FieldName != "" {
		actualOutput := results[len(test.expectedOutputs)]
		switch aob := actualOutput.(type) {
		case [][]string:
			for obi, ov := range aob {
				routedNumber := routerInputs[obi]
				routedModel, ok := router.routingMap[routedNumber]
				if !ok {
					if router.globalModel == nil {
						routedModel = test.routerConfig.Output.NoModelID
					} else {
						routedModel = globalModelName
					}
				}

				log.Printf("model name expected:%s actual:%s", routedModel, ov[0])

				if ov[0] != routedModel {
					t.Fatalf("model name output expected %s, got %s", "model"+strconv.Itoa(obi), ov[0])
				}
			}
		default:
			t.Fatalf("model name output expected [][]string, got %T", actualOutput)
		}
	}

	// check number of predict calls
	for _, evaluator := range mockEvaluators {
		expectCallCount, hasExpect := test.expectCallCounts[evaluator.modelName]
		if !hasExpect {
			continue
		}

		predictCalls := evaluator.getPredictCalls()
		log.Printf("predict calls %s expected:%d, actual:%d", evaluator.modelName, expectCallCount, predictCalls)

		if predictCalls != expectCallCount {
			t.Fatalf("predict calls expected %d, got %d", expectCallCount, predictCalls)
		}
	}
}

func prepareTestRouter(t *testing.T, test predictTestCase) ([]int, *Router, map[string]*mockEvaluator, string, []interface{}) {
	tritonServerID := "test_server"
	cfg := &config.Model{
		ID:       test.name,
		Mode:     "router",
		Platform: "triton",
		Router:   test.routerConfig,
		Triton: &config.TritonConfig{
			ServerID: tritonServerID,
		},
	}

	cfg.Init(nil)

	signature := &domain.Signature{}
	for i := range test.stringInputs {
		signature.Inputs = append(signature.Inputs, domain.Input{
			Name:  strconv.Itoa(i),
			Index: i,
			Type:  reflect.TypeOf(""),
		})
	}

	for i := range test.expectedOutputs {
		signature.Outputs = append(signature.Outputs, domain.Output{
			Name:     strconv.Itoa(i),
			Index:    i,
			DataType: "float32",
		})
	}

	var routerInputs []int = test.routerInputs
	if routerInputs == nil {
		sampledInput := test.stringInputs[0]
		routerInputs = make([]int, len(sampledInput))
		for j := range len(sampledInput) {
			routerInputs[j] = j
		}
	}

	router, err := newRouter(cfg, nil, map[string]UnloadService{
		tritonServerID: &triton.Service{},
	}, nil)

	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	// manually generate router configuration

	// see how model names are constructed later
	model0Name := "model0"
	model1Name := "model1"
	router.routingMap = map[int]string{
		0: model0Name,
		1: model1Name,
	}

	routerInputName := cfg.Router.InputName
	routerInput := domain.Input{Name: routerInputName, Index: 0, Type: reflect.TypeOf(int64(0))}

	routerOutputs := make([]domain.Output, len(signature.Outputs))
	copy(routerOutputs, signature.Outputs)

	if test.routerConfig.Output.FieldName != "" {
		routerOutputs = append(routerOutputs, domain.Output{
			Name:     test.routerConfig.Output.FieldName,
			Index:    len(routerOutputs),
			DataType: "string",
		})
	}

	// initialize ioState with base outputs
	router.ioState = &IOState{
		inputs: map[string]*domain.Input{
			routerInputName: &routerInput,
		},
		signature: &domain.Signature{
			Inputs: []domain.Input{
				routerInput,
			},
			Outputs: routerOutputs,
		},
		routerInputOffset: 0,
	}

	// add inputs
	for i, sigInput := range signature.Inputs {
		router.ioState.inputs[sigInput.Name] = &signature.Inputs[i]
		router.ioState.signature.Inputs = append(router.ioState.signature.Inputs, sigInput)
	}

	model1Signature := &domain.Signature{
		Inputs:  make([]domain.Input, len(signature.Inputs)),
		Outputs: make([]domain.Output, len(signature.Outputs)),
	}

	copy(model1Signature.Inputs, signature.Inputs)
	copy(model1Signature.Outputs, signature.Outputs)

	if test.reverseInputs {
		slices.Reverse(model1Signature.Inputs)
	}

	if test.reverseOutputs {
		slices.Reverse(model1Signature.Outputs)
	}

	mockEvaluators := map[string]*mockEvaluator{
		model0Name: {
			signature: func() *domain.Signature { return signature },
			predictor: appendFloatPredict,
			modelName: model0Name,
		},
		model1Name: {
			signature: func() *domain.Signature { return model1Signature },
			predictor: appendFloatPredict,
			modelName: model1Name,
		},
	}

	router.routingTable = make(map[string]platform.PlatformEvaluator)
	for modelName, evaluator := range mockEvaluators {
		router.routingTable[modelName] = evaluator
	}

	globalModelName := cfg.Router.Output.GlobalModelOverride
	if router.hasGlobalModel {
		router.globalModel = &mockEvaluator{
			signature: func() *domain.Signature { return signature },
			predictor: appendFloatPredict,
			modelName: globalModelName,
		}

		router.routingTable[globalModelName] = router.globalModel
	}

	// reshape inputs
	params := []interface{}{}

	paramRouterInputs := make([][]int64, len(routerInputs))
	for pri, ri := range routerInputs {
		paramRouterInputs[pri] = []int64{int64(ri)}
	}
	params = append(params, paramRouterInputs)

	for _, input := range test.stringInputs {
		inputVals := [][]string{}
		for _, inputVal := range input {
			inputVals = append(inputVals, []string{inputVal})
		}

		params = append(params, inputVals)
	}
	return routerInputs, router, mockEvaluators, globalModelName, params
}

func TestRouter_Predict_GlobalModel(t *testing.T) {
	tests := []predictTestCase{
		{
			name:         "with_global_model",
			routerConfig: makeConfig(configVariant{numOutputs: 1, hasGlobalModel: true, hasOutputName: false}),
			stringInputs: [][]string{
				{"1", "2", "3"},
				{"4", "5", "6"},
			},
			expectedOutputs: [][]float32{
				{14, 25, 36},
			},
		},
		{
			name:         "without_global_model",
			routerConfig: makeConfig(configVariant{numOutputs: 1, hasGlobalModel: false, hasOutputName: false}),
			stringInputs: [][]string{
				{"1", "2", "3"},
				{"4", "5", "6"},
			},
			expectedOutputs: [][]float32{
				// third offset should get fixed prediction for output 0
				{14, 25, 0.1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			predictTest(t, test)
		})
	}
}

func TestRouter_Predict_ModelName(t *testing.T) {
	t.Run("no global model", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{numOutputs: 1, hasGlobalModel: false, hasOutputName: true}),
			stringInputs: [][]string{
				{"1", "2", "3"},
				{"4", "5", "6"},
			},
			expectedOutputs: [][]float32{
				{14, 25, 0.1},
			},
		})
	})

	t.Run("with global model", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{numOutputs: 1, hasGlobalModel: true, hasOutputName: true}),
			stringInputs: [][]string{
				{"1", "2", "3"},
				{"4", "5", "6"},
			},
			expectedOutputs: [][]float32{
				{14, 25, 36},
			},
		})
	})
}

func TestRouter_Predict_BatchingBehavior(t *testing.T) {
	t.Run("batch to model", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{
				numOutputs:     1,
				hasGlobalModel: false,
				hasOutputName:  true,
			}),
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			routerInputs: []int{
				0, 1, 0, 1, 0, 1,
			},
			expectedOutputs: [][]float32{
				{11, 22, 33, 44, 55, 66},
			},
			expectCallCounts: map[string]int{
				"model0": 1,
				"model1": 1,
			},
		})
	})

	t.Run("force batch size 1", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{
				numOutputs:      1,
				hasGlobalModel:  false,
				hasOutputName:   true,
				forceBatchSize1: true,
			}),
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			routerInputs: []int{
				0, 1, 0, 1, 0, 1,
			},
			expectedOutputs: [][]float32{
				{11, 22, 33, 44, 55, 66},
			},
			expectCallCounts: map[string]int{
				"model0": 3,
				"model1": 3,
			},
		})
	})

	t.Run("batched with no global", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{
				numOutputs:     1,
				hasGlobalModel: false,
				hasOutputName:  true,
			}),
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			routerInputs: []int{
				0, 1, 0, 2, 0, 1,
			},
			expectedOutputs: [][]float32{
				{11, 22, 33, 0.1, 55, 66},
			},
			expectCallCounts: map[string]int{
				"model0": 1,
				"model1": 1,
			},
		})
	})
}

func TestRouter_Predict_SignatureReordering(t *testing.T) {
	t.Run("reverse inputs", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{
				numOutputs:     1,
				hasGlobalModel: false,
				hasOutputName:  true,
			}),
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			routerInputs: []int{
				0, 1, 0, 1, 0, 1,
			},
			expectedOutputs: [][]float32{
				{11, 22, 33, 44, 55, 66},
			},
			expectCallCounts: map[string]int{
				"model0": 1,
				"model1": 1,
			},
			reverseInputs: true,
		})
	})

	t.Run("reverse outputs", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{
				numOutputs:     1,
				hasGlobalModel: false,
				hasOutputName:  true,
			}),
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			routerInputs: []int{
				0, 1, 0, 1, 0, 1,
			},
			expectedOutputs: [][]float32{
				{11, 22, 33, 44, 55, 66},
			},
			expectCallCounts: map[string]int{
				"model0": 1,
				"model1": 1,
			},
			reverseOutputs: true,
		})
	})

	t.Run("reversed fixed evaluator", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{
				numOutputs:       2,
				hasGlobalModel:   false,
				hasOutputName:    true,
				reverseFPOutputs: true,
			}),
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			routerInputs: []int{
				0, 1, 0, 2, 0, 1,
			},
			expectedOutputs: [][]float32{
				{11, 22, 33, 0.1, 55, 66},
				{111, 122, 133, 1.1, 155, 166},
			},
			expectCallCounts: map[string]int{
				"model0": 1,
				"model1": 1,
			},
		})
	})

	t.Run("reversed everything", func(t *testing.T) {
		predictTest(t, predictTestCase{
			routerConfig: makeConfig(configVariant{
				numOutputs:       2,
				hasGlobalModel:   false,
				hasOutputName:    true,
				reverseFPOutputs: true,
			}),
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			routerInputs: []int{
				0, 1, 0, 2, 0, 1,
			},
			reverseInputs:  true,
			reverseOutputs: true,
			expectedOutputs: [][]float32{
				{11, 22, 33, 0.1, 55, 66},
				{111, 122, 133, 1.1, 155, 166},
			},
			expectCallCounts: map[string]int{
				"model0": 1,
				"model1": 1,
			},
		})
	})
}

func TestRouter_Predict_Queuing(t *testing.T) {
	rtCfg := makeConfig(configVariant{
		numOutputs:     1,
		hasGlobalModel: false,
		hasOutputName:  true,
	})

	rtCfg.MaxQueueSize = 5
	rtCfg.Workers = 1

	_, router, mockEvaluators, _, params := prepareTestRouter(t,
		predictTestCase{
			routerConfig: rtCfg,
			stringInputs: [][]string{
				{"1", "2", "3", "4", "5", "6"},
				{"1", "2", "3", "4", "5", "6"},
			},
			// hack to work around how output signature is built
			expectedOutputs: [][]float32{
				{1},
			},
		})

	doneGroup := &sync.WaitGroup{}
	for _, evaluator := range mockEvaluators {
		doneGroup.Add(1)

		evaluator.waitFor = &sync.WaitGroup{}
		evaluator.doneGroup = doneGroup
		evaluator.waitFor.Add(1)
	}

	errCh := make(chan error, 10)

	var foundError uint32
	foundErrorLock := &sync.WaitGroup{}
	foundErrorLock.Add(1)

	ctx := context.Background()
	runPredictWG := &sync.WaitGroup{}
	for pi := range 10 {
		runPredictWG.Add(1)
		go func() {
			defer runPredictWG.Done()
			_, err := router.Predict(ctx, params)
			log.Printf("predict %d error: %v", pi, err)
			if err != nil {
				if atomic.CompareAndSwapUint32(&foundError, 0, 1) {
					foundErrorLock.Done()
				}

				errCh <- err
			}
		}()
	}

	unlockedCh := make(chan struct{}, 1)

	go func() {
		foundErrorLock.Wait()
		unlockedCh <- struct{}{}
	}()

	dl, ok := t.Deadline()
	boundCtx := ctx
	if ok {
		var cancel context.CancelFunc
		boundCtx, cancel = context.WithDeadline(ctx, dl)
		defer cancel()
	}

	select {
	case <-boundCtx.Done():
		t.Fatalf("test timed out")

	case <-unlockedCh:
		// positive case
	}

	for _, evaluator := range mockEvaluators {
		// unblock evaluators
		evaluator.waitFor.Done()
	}

	// wait for evaluators to finish
	doneGroup.Wait()
	runPredictWG.Wait()

	close(errCh)

	foundQueueSizeError := false
	for e := range errCh {
		if e != nil {
			if strings.Contains(e.Error(), queueSizeExceededError) {
				foundQueueSizeError = true
			} else {
				t.Fatalf("Predict error: %v", e)
			}
		}
	}

	if !foundQueueSizeError {
		t.Fatalf("queue size exceeded not found")
	}
}

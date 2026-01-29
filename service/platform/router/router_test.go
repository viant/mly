package router

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared/common"
)

// --- Router Predict scaffolds ---

type mockTritonServer struct {
	mu sync.Mutex

	readyState   map[string]bool
	modelLoadErr map[string]error
}

func (m *mockTritonServer) ModelLoad(ctx context.Context, modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.modelLoadErr[modelName]; err != nil {
		return err
	}

	if m.readyState == nil {
		m.readyState = make(map[string]bool)
	}

	m.readyState[modelName] = true

	return nil
}

type mockEvaluator struct {
	tritonServer *mockTritonServer

	modelName string
	signature func() *domain.Signature
}

func (m *mockEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	inputs := m.signature().Inputs
	if len(inputs) != len(params) {
		return nil, fmt.Errorf("mock error: expected %d inputs, got %d", len(inputs), len(params))
	}

	// params is expected to have a single non-router input in this test: [][]string with shape [1][1]
	var v string
	switch typed := params[0].(type) {
	case [][]string:
		v = typed[0][0]
	default:
		tval := reflect.TypeOf(params[0])
		panic("unexpected input type in mock Predict(): " + tval.String())
	}
	// simple function: length of string as float32
	out := [][]float32{{float32(len(v))}}
	return []interface{}{out}, nil
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

type mockUnloader struct {
	tritonServer *mockTritonServer
	unloadCh     chan string
}

func (m *mockUnloader) ModelUnload(ctx context.Context, tritonModelName string) error {
	if m.tritonServer != nil {
		m.tritonServer.mu.Lock()
		defer m.tritonServer.mu.Unlock()

		if m.tritonServer.readyState == nil {
			m.tritonServer.readyState = make(map[string]bool)
		}

		m.tritonServer.readyState[tritonModelName] = false
	}

	ch := m.unloadCh

	if ch != nil {
		ch <- tritonModelName
	}

	return nil
}

func waitForCalls(t *testing.T, ch <-chan string, count int) []string {
	t.Helper()
	var out []string
	for i := 0; i < count; i++ {
		select {
		case v := <-ch:
			out = append(out, v)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for call %d/%d", i+1, count)
		}
	}
	return out
}

func TestRouter_Predict(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		routerConfig *config.RouterConfig
		verifier     func(t *testing.T, results []interface{})
	}{
		{
			name: "with global model",
			routerConfig: &config.RouterConfig{
				InputName: "router_id",
				Global: config.GlobalModelConfig{
					Exists: true, // avoid fixed replacements path
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 1 {
					t.Fatalf("expected 1 output, got %d", len(results))
				}
				out, ok := results[0].([][]float32)
				if !ok {
					t.Fatalf("expected [][]float32, got %T", results[0])
				}
				want := [][]float32{{1}, {4}}
				if !reflect.DeepEqual(out, want) {
					t.Errorf("output mismatch: got %#v, want %#v", out, want)
				}
			},
		},
		{
			name: "without global model",
			routerConfig: &config.RouterConfig{
				InputName: "router_id",
				Global: config.GlobalModelConfig{
					PredictionReplacements: []config.PredictionReplacement{
						{
							Name:  "score",
							Type:  "float32",
							Value: 1.0,
						},
					},
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 1 {
					t.Fatalf("expected 1 output, got %d", len(results))
				}
				out, ok := results[0].([][]float32)
				if !ok {
					t.Fatalf("expected [][]float32, got %T", results[0])
				}
				want := [][]float32{{1}, {4}}
				if !reflect.DeepEqual(out, want) {
					t.Errorf("output mismatch: got %#v, want %#v", out, want)
				}
			},
		},
		{
			name: "with model output name",
			routerConfig: &config.RouterConfig{
				InputName: "router_id",
				Global: config.GlobalModelConfig{
					Exists: true,
				},
				Output: config.OutputConfig{
					FieldName: "model_output",
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 2 {
					t.Fatalf("expected 2 outputs, got %d, %v", len(results), results)
				}

				func() {
					out, ok := results[0].([][]float32)
					if !ok {
						t.Fatalf("expected [][]float32, got %T", results[0])
					}
					want := [][]float32{{1}, {4}}
					if !reflect.DeepEqual(out, want) {
						t.Errorf("output mismatch: got %#v, want %#v", out, want)
					}
				}()

				func() {
					out, ok := results[1].([][]string)
					if !ok {
						t.Fatalf("expected [][]string, got %T", results[1])
					}
					want := [][]string{{"model1"}, {"model2"}}
					if !reflect.DeepEqual(out, want) {
						t.Errorf("output mismatch: got %#v, want %#v", out, want)
					}
				}()
			},
		},
	}

	downstreamInput := domain.Input{
		Name:  "text",
		Index: 1,
		Type:  reflect.TypeOf(""),
	}

	downstreamSignature := &domain.Signature{
		Inputs: []domain.Input{downstreamInput},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.routerConfig.ConfigURL == "" {
				test.routerConfig.ConfigURL = "memory://router-config"
			}

			cfg := &config.Model{
				ID:       "router_test",
				Mode:     "router",
				Platform: "triton",
				Router:   test.routerConfig,
				Triton: &config.TritonConfig{
					ServerID: "test_server",
				},
			}

			cfg.Init(nil)
			cfg.Router.MaxQueueSize = 1000
			cfg.Router.Workers = 3

			router, err := newRouter(cfg, nil, map[string]UnloadService{
				"test_server": &triton.Service{},
			}, func(modelName string) (platform.PlatformEvaluator, error) {
				return &mockEvaluator{signature: func() *domain.Signature { return downstreamSignature }}, nil
			})

			if err != nil {
				t.Fatalf("NewRouter error: %v", err)
			}

			router.routingMap = map[int]string{
				1: "model1",
				2: "model2",
			}

			routerOutputs := []domain.Output{
				{Name: "score", Index: 0, DataType: "float32"},
			}

			if test.routerConfig.Output.FieldName != "" {
				routerOutputs = append(routerOutputs, domain.Output{Name: test.routerConfig.Output.FieldName, Index: 1, DataType: "string"})
			}

			routerInputName := cfg.Router.InputName
			routerInput := domain.Input{Name: routerInputName, Index: 0, Type: reflect.TypeOf(int64(0))}

			router.ioState = &IOState{
				inputs: map[string]*domain.Input{
					routerInputName:      &routerInput,
					downstreamInput.Name: &downstreamInput,
				},

				signature: &domain.Signature{
					Inputs: []domain.Input{
						routerInput,
						downstreamInput,
					},
					Outputs: routerOutputs,
				},
			}

			mockEval := &mockEvaluator{signature: func() *domain.Signature { return downstreamSignature }}
			router.routingTable = map[string]platform.PlatformEvaluator{
				"model1": mockEval,
				"model2": mockEval,
			}

			// batch of 2
			params := []interface{}{
				[][]int64{{1}, {2}},         // router id
				[][]string{{"a"}, {"abcd"}}, // backend input
			}

			results, err := router.Predict(ctx, params)
			if err != nil {
				t.Fatalf("Predict error: %v", err)
			}

			test.verifier(t, results)
		})
	}
}

// --- Batched Prediction Tests ---

// countingMockEvaluator is a configurable mock that handles batched inputs and tracks calls
type countingMockEvaluator struct {
	modelName    string
	signature    *domain.Signature
	predictCalls int
	mu           sync.Mutex
	err          error // if set, Predict returns this error
}

func newTestMockEvaluator(name string, sig *domain.Signature) *countingMockEvaluator {
	return &countingMockEvaluator{modelName: name, signature: sig}
}

func (m *countingMockEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	m.mu.Lock()
	m.predictCalls++
	m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	if len(params) == 0 {
		return nil, fmt.Errorf("no params provided")
	}

	// Determine batch size from first input
	var batchSize int
	switch typed := params[0].(type) {
	case [][]string:
		batchSize = len(typed)
	case [][]int32:
		batchSize = len(typed)
	case [][]int64:
		batchSize = len(typed)
	case [][]float32:
		batchSize = len(typed)
	case [][]float64:
		batchSize = len(typed)
	default:
		return nil, fmt.Errorf("unexpected input type: %T", params[0])
	}

	// Compute output: for each row, output the length of the first string input
	results := make([][]float32, batchSize)
	for i := 0; i < batchSize; i++ {
		var length int
		if typed, ok := params[0].([][]string); ok {
			length = len(typed[i][0])
		}
		results[i] = []float32{float32(length)}
	}

	return []interface{}{results}, nil
}

func (m *countingMockEvaluator) Signature() *domain.Signature     { return m.signature }
func (m *countingMockEvaluator) Dictionary() *common.Dictionary   { return nil }
func (m *countingMockEvaluator) Inputs() map[string]*domain.Input { return nil }
func (m *countingMockEvaluator) Stats(map[string]interface{})     {}
func (m *countingMockEvaluator) Close() error                     { return nil }
func (m *countingMockEvaluator) ReloadIfNeeded(ctx context.Context) error {
	return nil
}

func (m *countingMockEvaluator) getPredictCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.predictCalls
}

// predictTestSetup holds the common test setup for prediction tests
type predictTestSetup struct {
	router     *Router
	evaluators map[string]*countingMockEvaluator
	signature  *domain.Signature
}

// predictTestOptions configures the test setup
type predictTestOptions struct {
	forceBatchSize1 bool
	modelOutputName string
	modelNames      []string // defaults to ["model1", "model2"]
}

// setupPredictTest creates a router with mock evaluators for prediction testing
func setupPredictTest(t *testing.T, opts predictTestOptions) *predictTestSetup {
	t.Helper()

	if len(opts.modelNames) == 0 {
		opts.modelNames = []string{"model1", "model2"}
	}

	// Create shared signature
	downstreamSig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
		},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	// Create evaluators
	evaluators := make(map[string]*countingMockEvaluator)
	for _, name := range opts.modelNames {
		evaluators[name] = newTestMockEvaluator(name, downstreamSig)
	}

	// Create config
	cfg := &config.Model{
		ID:       "router_predict_test",
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL:       "memory://router-config",
			InputName:       "router_id",
			ForceBatchSize1: opts.forceBatchSize1,
			Global:          config.GlobalModelConfig{Exists: true},
		},
		Triton: &config.TritonConfig{ServerID: "test_server"},
	}

	if opts.modelOutputName != "" {
		cfg.Router.Output.FieldName = opts.modelOutputName
	}

	cfg.Init(nil)
	cfg.Router.MaxQueueSize = 1000
	cfg.Router.Workers = 10

	// Create router
	router, err := newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		if eval, ok := evaluators[modelName]; ok {
			return eval, nil
		}
		return evaluators[opts.modelNames[0]], nil
	})
	if err != nil {
		t.Fatalf("newRouter error: %v", err)
	}

	// Setup routing map (1->model1, 2->model2, etc.)
	router.routingMap = make(map[int]string)
	for i, name := range opts.modelNames {
		router.routingMap[i+1] = name
	}

	// Setup routing table
	router.routingTable = make(map[string]platform.PlatformEvaluator)
	for name, eval := range evaluators {
		router.routingTable[name] = eval
	}

	// Build signature outputs
	outputs := []domain.Output{{Name: "score", Index: 0, DataType: "float32"}}
	if opts.modelOutputName != "" {
		outputs = append(outputs, domain.Output{Name: opts.modelOutputName, Index: 1, DataType: "string"})
	}

	// Setup ioState
	routerInputName := cfg.Router.InputName
	routerInput := domain.Input{Name: routerInputName, Index: 1, Type: reflect.TypeOf(int64(0))}
	downstreamInput := domain.Input{Name: "text", Index: 0, Type: reflect.TypeOf("")}

	routerSig := &domain.Signature{
		Inputs:  []domain.Input{downstreamInput, routerInput},
		Outputs: outputs,
	}

	router.ioState = &IOState{
		inputs: map[string]*domain.Input{
			routerInputName:      &routerInput,
			downstreamInput.Name: &downstreamInput,
		},
		signature:         routerSig,
		routerInputOffset: 1,
	}

	return &predictTestSetup{
		router:     router,
		evaluators: evaluators,
		signature:  routerSig,
	}
}

func TestRouter_Predict_BatchingBehavior(t *testing.T) {
	tests := []struct {
		name            string
		forceBatchSize1 bool
		inputs          []string
		routingIDs      []int64
		wantScores      [][]float32
		wantCallCounts  map[string]int
	}{
		{
			name:            "batched_groups_by_model",
			forceBatchSize1: false,
			inputs:          []string{"a", "bb", "ccc", "dddd", "eeeee", "ffffff"},
			routingIDs:      []int64{1, 2, 1, 2, 1, 2}, // alternating model1/model2
			wantScores:      [][]float32{{1}, {2}, {3}, {4}, {5}, {6}},
			wantCallCounts:  map[string]int{"model1": 1, "model2": 1}, // each model called once
		},
		{
			name:            "batched_single_model",
			forceBatchSize1: false,
			inputs:          []string{"a", "bb", "ccc", "dddd"},
			routingIDs:      []int64{1, 1, 1, 1}, // all to model1
			wantScores:      [][]float32{{1}, {2}, {3}, {4}},
			wantCallCounts:  map[string]int{"model1": 1, "model2": 0},
		},
		{
			name:            "force_batch_size_1",
			forceBatchSize1: true,
			inputs:          []string{"a", "bb", "ccc", "dddd"},
			routingIDs:      []int64{1, 1, 1, 1}, // all to model1
			wantScores:      [][]float32{{1}, {2}, {3}, {4}},
			wantCallCounts:  map[string]int{"model1": 4, "model2": 0}, // 4 individual calls
		},
		{
			name:            "force_batch_size_1_multiple_models",
			forceBatchSize1: true,
			inputs:          []string{"a", "bb", "ccc", "dddd"},
			routingIDs:      []int64{1, 2, 1, 2}, // alternating
			wantScores:      [][]float32{{1}, {2}, {3}, {4}},
			wantCallCounts:  map[string]int{"model1": 2, "model2": 2}, // 2 calls each
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := setupPredictTest(t, predictTestOptions{
				forceBatchSize1: tt.forceBatchSize1,
			})

			// Build input params
			stringInputs := make([][]string, len(tt.inputs))
			for i, s := range tt.inputs {
				stringInputs[i] = []string{s}
			}
			routingInputs := make([][]int64, len(tt.routingIDs))
			for i, id := range tt.routingIDs {
				routingInputs[i] = []int64{id}
			}

			params := []interface{}{stringInputs, routingInputs}

			results, err := setup.router.Predict(context.Background(), params)
			if err != nil {
				t.Fatalf("Predict error: %v", err)
			}

			// Verify scores
			scores, ok := results[0].([][]float32)
			if !ok {
				t.Fatalf("expected [][]float32, got %T", results[0])
			}
			if !reflect.DeepEqual(scores, tt.wantScores) {
				t.Errorf("scores mismatch:\n  got:  %v\n  want: %v", scores, tt.wantScores)
			}

			// Verify call counts
			for modelName, wantCount := range tt.wantCallCounts {
				if eval, ok := setup.evaluators[modelName]; ok {
					gotCount := eval.getPredictCalls()
					if gotCount != wantCount {
						t.Errorf("%s call count: got %d, want %d", modelName, gotCount, wantCount)
					}
				}
			}
		})
	}
}

func TestRouter_Predict_WithModelOutput(t *testing.T) {
	setup := setupPredictTest(t, predictTestOptions{
		modelOutputName: "model_id",
	})

	params := []interface{}{
		[][]string{{"a"}, {"bb"}, {"ccc"}, {"dddd"}},
		[][]int64{{1}, {2}, {1}, {2}},
	}

	results, err := setup.router.Predict(context.Background(), params)
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(results))
	}

	// Verify scores
	scores, ok := results[0].([][]float32)
	if !ok {
		t.Fatalf("expected [][]float32, got %T", results[0])
	}
	wantScores := [][]float32{{1}, {2}, {3}, {4}}
	assert.Equal(t, wantScores, scores, "scores mismatch")

	// Verify model IDs
	modelIDs, ok := results[1].([][]string)
	if !ok {
		t.Fatalf("expected [][]string for model_id, got %T", results[1])
	}
	wantModelIDs := [][]string{{"model1"}, {"model2"}, {"model1"}, {"model2"}}
	assert.Equal(t, wantModelIDs, modelIDs, "model_id mismatch")
}

func TestRouter_Predict_ErrorPropagation(t *testing.T) {
	setup := setupPredictTest(t, predictTestOptions{
		modelNames: []string{"model1"},
	})

	// Configure evaluator to return error
	setup.evaluators["model1"].err = errors.New("model prediction failed")

	params := []interface{}{
		[][]string{{"a"}, {"bb"}},
		[][]int64{{1}, {1}},
	}

	_, err := setup.router.Predict(context.Background(), params)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !strings.Contains(err.Error(), "model prediction failed") {
		t.Errorf("expected error to contain 'model prediction failed', got: %v", err)
	}
}

// orderVerifyingEvaluator verifies inputs arrive in expected order and computes output
type orderVerifyingEvaluator struct {
	t             *testing.T
	modelName     string
	signature     *domain.Signature
	expectedOrder []string // expected input names in order
}

func (e *orderVerifyingEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	if len(params) != len(e.expectedOrder) {
		return nil, fmt.Errorf("expected %d inputs, got %d", len(e.expectedOrder), len(params))
	}

	// Verify we received inputs in the expected order by checking types match signature
	for i, param := range params {
		expectedName := e.expectedOrder[i]

		// Verify the type is a slice (basic sanity check)
		paramType := reflect.TypeOf(param)
		if paramType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("input %d (%s): expected slice, got %v", i, expectedName, paramType)
		}
	}

	// Compute output: concatenate first input values (assumes string inputs)
	var batchSize int
	switch typed := params[0].(type) {
	case [][]string:
		batchSize = len(typed)
	case [][]int64:
		batchSize = len(typed)
	default:
		return nil, fmt.Errorf("unexpected first input type: %T", params[0])
	}

	// Output: sum of string lengths from input_a + input_b
	results := make([][]float32, batchSize)
	for i := 0; i < batchSize; i++ {
		var sum int
		for _, param := range params {
			if typed, ok := param.([][]string); ok {
				sum += len(typed[i][0])
			}
		}
		results[i] = []float32{float32(sum)}
	}

	return []interface{}{results}, nil
}

func (e *orderVerifyingEvaluator) Signature() *domain.Signature             { return e.signature }
func (e *orderVerifyingEvaluator) Dictionary() *common.Dictionary           { return nil }
func (e *orderVerifyingEvaluator) Inputs() map[string]*domain.Input         { return nil }
func (e *orderVerifyingEvaluator) Stats(map[string]interface{})             {}
func (e *orderVerifyingEvaluator) Close() error                             { return nil }
func (e *orderVerifyingEvaluator) ReloadIfNeeded(ctx context.Context) error { return nil }

// TestRouter_Predict_DifferentInputOrdering verifies that models with different
// input orderings receive their inputs in the correct order
func TestRouter_Predict_DifferentInputOrdering(t *testing.T) {
	// Model1 expects: [input_a, input_b] (indices 0, 1)
	// Model2 expects: [input_b, input_a] (indices 0, 1) - REVERSED ORDER
	model1Sig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "input_a", Index: 0, Type: reflect.TypeOf("")},
			{Name: "input_b", Index: 1, Type: reflect.TypeOf("")},
		},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	model2Sig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "input_b", Index: 0, Type: reflect.TypeOf("")}, // REVERSED
			{Name: "input_a", Index: 1, Type: reflect.TypeOf("")}, // REVERSED
		},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	model1Eval := &orderVerifyingEvaluator{
		t:             t,
		modelName:     "model1",
		signature:     model1Sig,
		expectedOrder: []string{"input_a", "input_b"},
	}

	model2Eval := &orderVerifyingEvaluator{
		t:             t,
		modelName:     "model2",
		signature:     model2Sig,
		expectedOrder: []string{"input_b", "input_a"}, // expects reversed order
	}

	cfg := &config.Model{
		ID:       "router_ordering_test",
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			InputName: "router_id",
			Global:    config.GlobalModelConfig{Exists: true},
		},
		Triton: &config.TritonConfig{ServerID: "test_server"},
	}
	cfg.Init(nil)
	cfg.Router.MaxQueueSize = 1000
	cfg.Router.Workers = 10

	router, err := newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		if modelName == "model1" {
			return model1Eval, nil
		}
		return model2Eval, nil
	})
	if err != nil {
		t.Fatalf("newRouter error: %v", err)
	}

	router.routingMap = map[int]string{
		1: "model1",
		2: "model2",
	}
	router.routingTable = map[string]platform.PlatformEvaluator{
		"model1": model1Eval,
		"model2": model2Eval,
	}

	// Router's signature: [input_a, input_b, router_id]
	// This is the order the router receives inputs from the request
	routerSig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "input_a", Index: 0, Type: reflect.TypeOf("")},
			{Name: "input_b", Index: 1, Type: reflect.TypeOf("")},
			{Name: "router_id", Index: 2, Type: reflect.TypeOf(int64(0))},
		},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	router.ioState = &IOState{
		inputs: map[string]*domain.Input{
			"input_a":   &routerSig.Inputs[0],
			"input_b":   &routerSig.Inputs[1],
			"router_id": &routerSig.Inputs[2],
		},
		signature:         routerSig,
		routerInputOffset: 2, // router_id is at index 2
	}

	// Input data:
	// Row 0: input_a="aa", input_b="bbbb", router_id=1 (-> model1)
	// Row 1: input_a="ccc", input_b="dd", router_id=2 (-> model2)
	// Row 2: input_a="e", input_b="ffffff", router_id=1 (-> model1)
	// Row 3: input_a="gggg", input_b="h", router_id=2 (-> model2)
	params := []interface{}{
		[][]string{{"aa"}, {"ccc"}, {"e"}, {"gggg"}},    // input_a
		[][]string{{"bbbb"}, {"dd"}, {"ffffff"}, {"h"}}, // input_b
		[][]int64{{1}, {2}, {1}, {2}},                   // router_id
	}

	results, err := router.Predict(context.Background(), params)
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	// Verify results
	// Row 0: len("aa") + len("bbbb") = 2 + 4 = 6
	// Row 1: len("ccc") + len("dd") = 3 + 2 = 5
	// Row 2: len("e") + len("ffffff") = 1 + 6 = 7
	// Row 3: len("gggg") + len("h") = 4 + 1 = 5
	scores, ok := results[0].([][]float32)
	if !ok {
		t.Fatalf("expected [][]float32, got %T", results[0])
	}

	wantScores := [][]float32{{6}, {5}, {7}, {5}}
	if !reflect.DeepEqual(scores, wantScores) {
		t.Errorf("scores mismatch:\n  got:  %v\n  want: %v", scores, wantScores)
	}
}

// multiOutputEvaluator returns multiple outputs in the order specified by its signature
type multiOutputEvaluator struct {
	modelName string
	signature *domain.Signature
}

func (e *multiOutputEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	// Get batch size from first input
	var batchSize int
	switch typed := params[0].(type) {
	case [][]string:
		batchSize = len(typed)
	default:
		return nil, fmt.Errorf("unexpected input type: %T", params[0])
	}

	// Return outputs in the order defined by this evaluator's signature
	// For each row: score_a = len(input), score_b = len(input) * 2
	results := make([]interface{}, len(e.signature.Outputs))
	for outIdx, outDef := range e.signature.Outputs {
		outputData := make([][]float32, batchSize)
		for i := 0; i < batchSize; i++ {
			inputStr := params[0].([][]string)[i][0]
			var value float32
			switch outDef.Name {
			case "score_a":
				value = float32(len(inputStr))
			case "score_b":
				value = float32(len(inputStr) * 2)
			}
			outputData[i] = []float32{value}
		}
		results[outIdx] = outputData
	}

	return results, nil
}

func (e *multiOutputEvaluator) Signature() *domain.Signature             { return e.signature }
func (e *multiOutputEvaluator) Dictionary() *common.Dictionary           { return nil }
func (e *multiOutputEvaluator) Inputs() map[string]*domain.Input         { return nil }
func (e *multiOutputEvaluator) Stats(map[string]interface{})             {}
func (e *multiOutputEvaluator) Close() error                             { return nil }
func (e *multiOutputEvaluator) ReloadIfNeeded(ctx context.Context) error { return nil }

// TestRouter_Predict_DifferentOutputOrdering verifies that models with different
// output orderings have their outputs correctly reordered to match the router's signature
func TestRouter_Predict_DifferentOutputOrdering(t *testing.T) {
	// Model1 returns: [score_a, score_b] (indices 0, 1)
	// Model2 returns: [score_b, score_a] (indices 0, 1) - REVERSED ORDER
	model1Sig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
		},
		Outputs: []domain.Output{
			{Name: "score_a", Index: 0, DataType: "float32"},
			{Name: "score_b", Index: 1, DataType: "float32"},
		},
	}

	model2Sig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
		},
		Outputs: []domain.Output{
			{Name: "score_b", Index: 0, DataType: "float32"}, // REVERSED
			{Name: "score_a", Index: 1, DataType: "float32"}, // REVERSED
		},
	}

	model1Eval := &multiOutputEvaluator{modelName: "model1", signature: model1Sig}
	model2Eval := &multiOutputEvaluator{modelName: "model2", signature: model2Sig}

	cfg := &config.Model{
		ID:       "router_output_ordering_test",
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			InputName: "router_id",
			Global:    config.GlobalModelConfig{Exists: true},
		},
		Triton: &config.TritonConfig{ServerID: "test_server"},
	}
	cfg.Init(nil)
	cfg.Router.MaxQueueSize = 1000
	cfg.Router.Workers = 10

	router, err := newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		if modelName == "model1" {
			return model1Eval, nil
		}
		return model2Eval, nil
	})
	if err != nil {
		t.Fatalf("newRouter error: %v", err)
	}

	router.routingMap = map[int]string{
		1: "model1",
		2: "model2",
	}
	router.routingTable = map[string]platform.PlatformEvaluator{
		"model1": model1Eval,
		"model2": model2Eval,
	}

	// Router's signature: outputs are [score_a, score_b] (this is the canonical order)
	routerSig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			{Name: "router_id", Index: 1, Type: reflect.TypeOf(int64(0))},
		},
		Outputs: []domain.Output{
			{Name: "score_a", Index: 0, DataType: "float32"},
			{Name: "score_b", Index: 1, DataType: "float32"},
		},
	}

	router.ioState = &IOState{
		inputs: map[string]*domain.Input{
			"text":      &routerSig.Inputs[0],
			"router_id": &routerSig.Inputs[1],
		},
		signature:         routerSig,
		routerInputOffset: 1,
	}

	// Input data:
	// Row 0: text="aa", router_id=1 (-> model1) => score_a=2, score_b=4
	// Row 1: text="bbb", router_id=2 (-> model2) => score_a=3, score_b=6
	// Row 2: text="c", router_id=1 (-> model1) => score_a=1, score_b=2
	// Row 3: text="dddd", router_id=2 (-> model2) => score_a=4, score_b=8
	params := []interface{}{
		[][]string{{"aa"}, {"bbb"}, {"c"}, {"dddd"}}, // text
		[][]int64{{1}, {2}, {1}, {2}},                // router_id
	}

	results, err := router.Predict(context.Background(), params)
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(results))
	}

	// Verify score_a (output index 0 in router's signature)
	scoreA, ok := results[0].([][]float32)
	if !ok {
		t.Fatalf("expected [][]float32 for score_a, got %T", results[0])
	}
	wantScoreA := [][]float32{{2}, {3}, {1}, {4}}
	if !reflect.DeepEqual(scoreA, wantScoreA) {
		t.Errorf("score_a mismatch:\n  got:  %v\n  want: %v", scoreA, wantScoreA)
	}

	// Verify score_b (output index 1 in router's signature)
	scoreB, ok := results[1].([][]float32)
	if !ok {
		t.Fatalf("expected [][]float32 for score_b, got %T", results[1])
	}
	wantScoreB := [][]float32{{4}, {6}, {2}, {8}}
	if !reflect.DeepEqual(scoreB, wantScoreB) {
		t.Errorf("score_b mismatch:\n  got:  %v\n  want: %v", scoreB, wantScoreB)
	}
}

// TestRouter_Predict_FixedEvaluatorOutputOrdering verifies that the fixed evaluator
// correctly reorders its outputs to match the router's signature order, even when
// the PredictionReplacements are in a different order than the model outputs
func TestRouter_Predict_FixedEvaluatorOutputOrdering(t *testing.T) {
	// Model signature has outputs: [score_a, score_b]
	// But PredictionReplacements are configured in reverse order: [score_b, score_a]
	// The fixed evaluator should reorder to match the router signature

	modelSig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
		},
		Outputs: []domain.Output{
			{Name: "score_a", Index: 0, DataType: "float32"},
			{Name: "score_b", Index: 1, DataType: "float32"},
		},
	}

	modelEval := &multiOutputEvaluator{modelName: "model1", signature: modelSig}

	// Create config with PredictionReplacements in REVERSE order from model outputs
	cfg := &config.Model{
		ID:       "router_fixed_ordering_test",
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			InputName: "router_id",
			Global: config.GlobalModelConfig{
				Exists: false, // This enables fixed evaluator
				PredictionReplacements: []config.PredictionReplacement{
					{Name: "score_b", Type: "float32", Value: 99.0}, // REVERSED ORDER
					{Name: "score_a", Type: "float32", Value: 42.0}, // REVERSED ORDER
				},
			},
		},
		Triton: &config.TritonConfig{ServerID: "test_server"},
	}
	cfg.Init(nil)
	cfg.Router.MaxQueueSize = 1000
	cfg.Router.Workers = 10

	router, err := newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		return modelEval, nil
	})
	if err != nil {
		t.Fatalf("newRouter error: %v", err)
	}

	router.routingMap = map[int]string{
		1: "model1",
	}
	router.routingTable = map[string]platform.PlatformEvaluator{
		"model1": modelEval,
	}

	// Router signature: outputs are [score_a, score_b]
	routerSig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			{Name: "router_id", Index: 1, Type: reflect.TypeOf(int64(0))},
		},
		Outputs: []domain.Output{
			{Name: "score_a", Index: 0, DataType: "float32"},
			{Name: "score_b", Index: 1, DataType: "float32"},
		},
	}

	router.ioState = &IOState{
		inputs: map[string]*domain.Input{
			"text":      &routerSig.Inputs[0],
			"router_id": &routerSig.Inputs[1],
		},
		signature:         routerSig,
		routerInputOffset: 1,
	}

	// Input data:
	// Row 0: text="aa", router_id=1 (-> model1)
	// Row 1: text="bbb", router_id=999 (-> fixed evaluator, ID not in routingMap)
	params := []interface{}{
		[][]string{{"aa"}, {"bbb"}},
		[][]int64{{1}, {999}}, // 999 not in routingMap, uses fixed evaluator
	}

	results, err := router.Predict(context.Background(), params)
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(results))
	}

	// Verify score_a (output index 0 in router's signature)
	// Row 0: from model, len("aa") = 2
	// Row 1: from fixed evaluator, should be 42.0 (NOT 99.0)
	scoreA, ok := results[0].([][]float32)
	if !ok {
		t.Fatalf("expected [][]float32 for score_a, got %T", results[0])
	}
	wantScoreA := [][]float32{{2}, {42}}
	if !reflect.DeepEqual(scoreA, wantScoreA) {
		t.Errorf("score_a mismatch:\n  got:  %v\n  want: %v", scoreA, wantScoreA)
	}

	// Verify score_b (output index 1 in router's signature)
	// Row 0: from model, len("aa") * 2 = 4
	// Row 1: from fixed evaluator, should be 99.0 (NOT 42.0)
	scoreB, ok := results[1].([][]float32)
	if !ok {
		t.Fatalf("expected [][]float32 for score_b, got %T", results[1])
	}
	wantScoreB := [][]float32{{4}, {99}}
	if !reflect.DeepEqual(scoreB, wantScoreB) {
		t.Errorf("score_b mismatch:\n  got:  %v\n  want: %v", scoreB, wantScoreB)
	}
}

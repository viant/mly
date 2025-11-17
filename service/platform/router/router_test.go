package router

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/platform"
	tricli "github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	sharedrouter "github.com/viant/mly/shared/config/router"
)

// --- Router Predict scaffolds ---

type mockPredictOnly struct{}

func (m *mockPredictOnly) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	// params is expected to have a single non-router input in this test: [][]string with shape [1][1]
	var v string
	switch typed := params[0].(type) {
	case [][]string:
		v = typed[0][0]
	default:
		tval := reflect.TypeOf(params[0])
		panic("unexpected input type in mockPredictOnly: " + tval.String())
	}
	// simple function: length of string as float32
	out := [][]float32{{float32(len(v))}}
	return []interface{}{out}, nil
}

func (m *mockPredictOnly) Signature() *domain.Signature     { return nil }
func (m *mockPredictOnly) Dictionary() *common.Dictionary   { return nil }
func (m *mockPredictOnly) Inputs() map[string]*domain.Input { return nil }
func (m *mockPredictOnly) Stats(map[string]interface{})     {}
func (m *mockPredictOnly) Close() error                     { return nil }
func (m *mockPredictOnly) ReloadIfNeeded(ctx context.Context) error {
	return nil
}

type mockTritonClient struct {
	mu           sync.Mutex
	loadCalls    []string
	unloadCalls  []string
	unloadCh     chan string
	modelLoadErr map[string]error
}

func (m *mockTritonClient) ServerReady(ctx context.Context) error { return nil }
func (m *mockTritonClient) ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) ([]interface{}, error) {
	return nil, nil
}
func (m *mockTritonClient) ModelReady(ctx context.Context, modelName string) (bool, error) {
	return true, nil
}
func (m *mockTritonClient) ModelLoad(ctx context.Context, modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadCalls = append(m.loadCalls, modelName)
	if err := m.modelLoadErr[modelName]; err != nil {
		return err
	}
	return nil
}
func (m *mockTritonClient) ModelUnload(ctx context.Context, modelName string) error {
	m.mu.Lock()
	m.unloadCalls = append(m.unloadCalls, modelName)
	ch := m.unloadCh
	m.mu.Unlock()
	if ch != nil {
		ch <- modelName
	}
	return nil
}
func (m *mockTritonClient) Close() error { return nil }

func (m *mockTritonClient) snapshotLoadCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.loadCalls...)
}

func (m *mockTritonClient) snapshotUnloadCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.unloadCalls...)
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

func TestRouter_Predict_RoutesAndConcats(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		routerConfig *config.RouterConfig
		verifier     func(t *testing.T, results []interface{})
	}{
		{
			name: "with global model",
			routerConfig: &config.RouterConfig{
				ConfigURL: "memory://router-config",
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
				ConfigURL: "memory://router-config",
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
				ConfigURL: "memory://router-config",
				InputName: "router_id",
				Global: config.GlobalModelConfig{
					Exists: true, // avoid fixed replacements path
				},
				Output: config.OutputConfig{
					FieldName: "model_output",
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 2 {
					t.Fatalf("expected 1 output, got %d", len(results))
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

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Model{
				ID:       "router_test",
				Mode:     "router",
				Platform: "triton",
				MetaInput: shared.MetaInput{
					Inputs: []*shared.Field{
						// router input first (default offset 0)
						{Name: "router_id", Index: 0, DataType: "int64"},
						// single backend input
						{Name: "text", Index: 1, DataType: "string"},
					},
					Outputs: []*shared.Field{
						{Name: "score", Index: 0, DataType: "float32"},
					},
				},
				Router: test.routerConfig,
				Triton: &config.TritonConfig{
					ServerID: "test_server",
				},
			}

			router, err := NewRouter(cfg, nil, map[string]tricli.TritonClient{
				"test_server": &mockTritonClient{},
			})

			if err != nil {
				t.Fatalf("NewRouter error: %v", err)
			}

			router.routingMap = map[int]string{
				1: "model1",
				2: "model2",
			}
			mockEval := &mockPredictOnly{}
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

func TestRouter_applyRouterConfig_LoadsAndSwaps(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockTritonClient{
		unloadCh: make(chan string, 2),
	}

	oldConfig := &sharedrouter.RouterConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
			{EntityID: 2, ModelName: "modelB"},
		},
		GlobalModelName: "global-old",
	}

	router := &Router{
		tritonClient: mockClient,
		modelConfig: &config.Model{
			Triton: &config.TritonConfig{
				Timeout: 100,
			},
		},
		indexToName: map[int]string{
			0: "text",
		},
		routerConfig: oldConfig,
		routingMap: map[int]string{
			1: "modelA",
			2: "modelB",
		},
		routingTable: map[string]platform.PlatformEvaluator{
			"modelA": &mockPredictOnly{},
			"modelB": &mockPredictOnly{},
		},
		globalModel: &mockPredictOnly{},
	}

	newConfig := &sharedrouter.RouterConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelB"},
			{EntityID: 3, ModelName: "modelC"},
		},
		GlobalModelName: "global-new",
	}

	if err := router.applyRouterConfig(ctx, newConfig); err != nil {
		t.Fatalf("applyRouterConfig returned error: %v", err)
	}

	loadCalls := router.tritonClient.(*mockTritonClient).snapshotLoadCalls()
	if len(loadCalls) != 3 {
		t.Fatalf("expected 3 model loads, got %d (%v)", len(loadCalls), loadCalls)
	}

	expectedLoads := map[string]bool{
		"modelB":     false,
		"modelC":     false,
		"global-new": false,
	}
	for _, call := range loadCalls {
		if _, ok := expectedLoads[call]; ok {
			expectedLoads[call] = true
		}
	}
	for model, seen := range expectedLoads {
		if !seen {
			t.Fatalf("expected load for %s was not observed; calls=%v", model, loadCalls)
		}
	}

	waitForCalls(t, mockClient.unloadCh, 2)
	unloadCalls := mockClient.snapshotUnloadCalls()
	expectedUnloads := map[string]bool{
		"modelA":     false,
		"global-old": false,
	}
	for _, call := range unloadCalls {
		if _, ok := expectedUnloads[call]; ok {
			expectedUnloads[call] = true
		}
	}
	for model, seen := range expectedUnloads {
		if !seen {
			t.Fatalf("expected unload for %s was not observed; calls=%v", model, unloadCalls)
		}
	}

	if router.routerConfig != newConfig {
		t.Fatalf("routerConfig pointer not updated")
	}

	expectedRouting := map[int]string{
		1: "modelB",
		3: "modelC",
	}
	if !reflect.DeepEqual(router.routingMap, expectedRouting) {
		t.Fatalf("routingMap mismatch, got %#v", router.routingMap)
	}

	if router.globalModel == nil {
		t.Fatalf("globalModel was not set")
	}

	if _, ok := router.routingTable["modelB"]; !ok {
		t.Fatalf("routingTable missing modelB")
	}
	if _, ok := router.routingTable["modelC"]; !ok {
		t.Fatalf("routingTable missing modelC")
	}
	if _, ok := router.routingTable["modelA"]; ok {
		t.Fatalf("routingTable still contains modelA")
	}
}

func TestRouter_applyRouterConfig_LoadError(t *testing.T) {
	ctx := context.Background()
	loadErr := errors.New("load failure")
	mockClient := &mockTritonClient{
		modelLoadErr: map[string]error{
			"modelX": loadErr,
		},
	}

	oldConfig := &sharedrouter.RouterConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
		},
	}

	router := &Router{
		tritonClient: mockClient,
		modelConfig: &config.Model{
			Triton: &config.TritonConfig{
				Timeout: 50,
			},
		},
		indexToName:  map[int]string{},
		routerConfig: oldConfig,
		routingMap: map[int]string{
			1: "modelA",
		},
	}

	newConfig := &sharedrouter.RouterConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 2, ModelName: "modelX"},
		},
	}

	err := router.applyRouterConfig(ctx, newConfig)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "modelX") {
		t.Fatalf("expected error mentioning modelX, got %v", err)
	}

	if router.routerConfig != oldConfig {
		t.Fatalf("routerConfig should remain unchanged on error")
	}

	if !reflect.DeepEqual(router.routingMap, map[int]string{1: "modelA"}) {
		t.Fatalf("routingMap should remain unchanged on error")
	}

	if router.routingTable != nil {
		t.Fatalf("routingTable should not be replaced on error")
	}

	loadCalls := mockClient.snapshotLoadCalls()
	if len(loadCalls) != 1 || loadCalls[0] != "modelX" {
		t.Fatalf("expected single load attempt for modelX, got %v", loadCalls)
	}
}

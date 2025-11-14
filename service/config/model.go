package config

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/viant/afs/file"
	batchconfig "github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/shared"
	"github.com/viant/tapper/config"
)

// Model represents model config
type Model struct {
	ID    string
	Debug bool

	// Mode overrides the endpoint behavior from inference to routing.
	// This primarily is used to confirm that the mode should be a router.
	// Can be one of "inference" or "router".
	// Defaults to "inference".
	Mode string `json:",omitempty" yaml:",omitempty"`

	// Platform specifies where inference occurs.
	// Can be one of "tensorflow" or "triton".
	// Defaults to "tensorflow".
	Platform string `json:",omitempty" yaml:",omitempty"`

	// Location is the path the model will be copied to.
	Location string `json:",omitempty" yaml:",omitempty"`

	// Dir is used to build a Location if Location is not provided.
	// The built Location will use Dir directory after os.TempDir() and ID.
	Dir string

	// URL is the location of the model.
	// If Platform is "triton", this is the HTTP prefix of the Triton server, and is deprecated.
	// It is preferred to use service/endpoint.Config.TritonServers and Triton.ServerID instead.
	// This will create a new connection per instance URL is used for a Triton server.
	// Additionally, it is assumed that explicit model control is not enabled for that HTTP URL.
	URL string

	Batch *BatcherConfigFile `json:",omitempty" yaml:",omitempty"`

	// Tags is used when loading the Savedmodel.
	// Defaults to []string{"serve"}.
	Tags []string

	// UseDict enables caching and replacing OOV values as "[UNK]" in cache key.
	// If UseDict is nil, defaults to true.
	UseDict *bool `json:",omitempty" yaml:",omitempty"`

	// Deprecated: we usually extract the dictionary/vocabulary from TF graph
	DictURL string

	shared.MetaInput `json:",omitempty" yaml:",inline"`

	// Deprecated: we can infer output types from TF graph, and there may be more than one output
	OutputType string `json:",omitempty" yaml:",omitempty"`

	// Transformer is the name of the model output transformer.
	Transformer string `json:",omitempty" yaml:",omitempty"`

	// DataStore is the name of the datastore to use for caching.
	DataStore string `json:",omitempty" yaml:",omitempty"`

	// Router must be provided if Mode is "router".
	Router *RouterConfig `json:",omitempty" yaml:",omitempty"`

	// Stream is a github.com/viant/tapper configuration.
	// All requests are eligible to be logged.
	Stream *config.Stream `json:",omitempty" yaml:",omitempty"`

	// Triton configuration for Triton Inference Server models
	Triton *TritonConfig `json:",omitempty" yaml:",omitempty"`

	// Modified shows the state of the model files.
	Modified *Modified `json:",omitempty" yaml:",omitempty"`

	DictMeta DictionaryMeta

	// Test is used to test the model on startup.
	Test TestPayload `json:",omitempty" yaml:",omitempty"`
}

type TestPayload struct {
	Test        bool // if all blank, do a non-batch test
	Single      map[string]interface{}
	SingleBatch bool // only relevant with Single or blank
	Batch       map[string][]interface{}
}

// DictionaryMeta is used to confirm proper reloading of model components
type DictionaryMeta struct {
	Hash     int
	Reloaded time.Time
	Error    string
}

// UseDictionary returns true if dictionary can be used
func (m Model) UseDictionary() bool {
	return m.UseDict == nil || *m.UseDict
}

// Init initialises model config
func (m *Model) Init(globalBatchConfig *batchconfig.BatcherConfig) {
	if len(m.Tags) == 0 {
		m.Tags = []string{"serve"}
	}

	if m.Location == "" {
		m.Location = path.Join(os.TempDir(), m.ID+m.Dir)
	}

	_ = os.MkdirAll(m.Location, file.DefaultDirOsMode)

	m.Modified = &Modified{}

	m.MetaInput.Init()

	if m.Batch != nil {
		m.Batch.Init()
		v := m.Batch.BatcherConfig.Verbose
		if m.Debug && v == nil {
			v = &batchconfig.V{
				Output: true,
			}

			m.Batch.BatcherConfig.Verbose = v
		}

		if v != nil && v.ID == "" {
			v.ID = m.ID
		}
	} else if globalBatchConfig != nil {
		m.Batch = &BatcherConfigFile{
			BatcherConfig: *globalBatchConfig,
		}
	}

	if m.Router != nil {
		m.Router.Init()
	}
}

func (m *Model) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("model.ID was empty")
	}

	// Platform-specific validation
	platform := m.GetPlatform()
	switch platform {
	case "tensorflow":
		if m.URL == "" {
			return fmt.Errorf("tensorflow model %s requires URL", m.ID)
		}

		if m.Mode == "router" {
			return fmt.Errorf("tensorflow model %s is not supported in router mode", m.ID)
		}
	case "triton":
		if m.Triton == nil {
			return fmt.Errorf("triton model %s requires Triton configuration", m.ID)
		}

		m.Triton.Init()

		if err := m.Triton.Validate(m.Mode == "router", m.URL != ""); err != nil {
			return fmt.Errorf("triton model %s config invalid: %w", m.ID, err)
		}
	default:
		return fmt.Errorf("unsupported platform '%s' for model %s (supported: tensorflow, triton)", platform, m.ID)
	}

	if m.Mode == "router" {
		if m.Router == nil {
			return fmt.Errorf("router model %s requires Router configuration", m.ID)
		}

		if err := m.Router.Validate(); err != nil {
			return fmt.Errorf("router model %s config invalid: %w", m.ID, err)
		}
	}

	return nil
}

// TritonConfig represents Triton Inference Server specific configuration.
type TritonConfig struct {
	// Model name in Triton.
	// Optional if Model.Mode is "router".
	ModelName string `json:",omitempty" yaml:",omitempty"`

	// ServerID is the ID of the Triton server.
	ServerID string `json:",omitempty" yaml:",omitempty"`

	// RepositoryExplicit should be true if the Model Repository is in EXPLICIT mode.
	// In the case of URL-based Triton configuration, the default is to assume POLL mode.
	// See https://docs.nvidia.com/deeplearning/triton-inference-server/user-guide/docs/user_guide/model_management.html
	RepositoryExplicit bool `json:",omitempty" yaml:",omitempty"`

	// Maximum request timeout in milliseconds.
	// Defaults to 100 milliseconds.
	Timeout int `json:",omitempty" yaml:",omitempty"`
}

func (t *TritonConfig) Init() {
	if t.Timeout == 0 {
		t.Timeout = 100
	}
}

func (t *TritonConfig) Validate(isRouter bool, urlPresent bool) error {
	if !isRouter && t.ModelName == "" {
		return fmt.Errorf("triton ModelName is required")
	}

	if t.ServerID == "" && !urlPresent {
		return fmt.Errorf("triton ServerID or Model.URL is required")
	}

	return nil
}

// GetPlatform returns the platform with default to "tensorflow" for backward compatibility
func (m *Model) GetPlatform() string {
	if m.Platform == "" {
		m.Platform = "tensorflow"
	}

	return m.Platform
}

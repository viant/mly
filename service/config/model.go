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

	// Location is the path the model will be copied to.
	Location string `json:",omitempty" yaml:",omitempty"`

	// Dir is used to build a Location if Location is not provided.
	// The build Location will use Dir directory after os.TempDir() and ID.
	Dir string

	URL string

	Batch *BatcherConfigFile `json:",omitempty" yaml:",omitempty"`

	// Tags is used when loading the Savedmodel.
	// Defaults to []string{"serve"}.
	Tags []string

	// UseDict enables caching and replacing OOV values as "[UNK]" in cache key.
	// If UseDict is nil, defaults to true.
	UseDict *bool `json:",omitempty" yaml:",omitempty"`

	DictURL string // Deprecated: we usually extract the dictionary/vocabulary from TF graph

	shared.MetaInput `json:",omitempty" yaml:",inline"`

	// Deprecated: we can infer output types from TF graph, and there may be more than one output
	OutputType string `json:",omitempty" yaml:",omitempty"`

	Transformer string `json:",omitempty" yaml:",omitempty"`

	// caching
	DataStore string `json:",omitempty" yaml:",omitempty"`

	// Stream is a github.com/viant/tapper configuration.
	// All requests are eligible to be logged.
	Stream *config.Stream `json:",omitempty" yaml:",omitempty"`

	// Modified shows the state of the model files.
	Modified *Modified `json:",omitempty" yaml:",omitempty"`

	DictMeta DictionaryMeta

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
}

// Validate validates model config
func (m *Model) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("model.ID was empty")
	}

	if m.URL == "" {
		return fmt.Errorf("model.URL was empty")
	}

	return nil
}

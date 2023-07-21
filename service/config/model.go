package config

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/viant/afs/file"
	"github.com/viant/mly/shared"
	"github.com/viant/tapper/config"
)

// Model represents model config
type Model struct {
	ID    string
	Debug bool

	Location string `json:",omitempty" yaml:",omitempty"`
	Dir      string
	URL      string

	Tags []string

	// IO and caching
	UseDict          *bool  `json:",omitempty" yaml:",omitempty"`
	DictURL          string // Deprecated: we usually extract the dictionary/vocabulary from TF graph
	shared.MetaInput `json:",omitempty" yaml:",inline"`
	OutputType       string `json:",omitempty" yaml:",omitempty"` // Deprecated: we can infer output types from TF graph
	Transformer      string `json:",omitempty" yaml:",omitempty"`

	// caching
	DataStore string `json:",omitempty" yaml:",omitempty"`

	// logging
	Stream *config.Stream `json:",omitempty" yaml:",omitempty"`

	// for health and monitoring
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
func (m *Model) Init() {
	if len(m.Tags) == 0 {
		m.Tags = []string{"serve"}
	}
	if m.Location == "" {
		m.Location = path.Join(os.TempDir(), m.ID+m.Dir)
	}
	_ = os.MkdirAll(m.Location, file.DefaultDirOsMode)
	m.Modified = &Modified{}
	m.MetaInput.Init()
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

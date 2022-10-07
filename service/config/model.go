package config

import (
	"fmt"
	"github.com/viant/afs/file"
	"github.com/viant/mly/shared"
	"github.com/viant/tapper/config"
	"os"
	"path"
)

//Model represents model config
type Model struct {
	ID               string
	URL              string
	Debug            bool
	Location         string `json:",omitempty" yaml:",omitempty"`
	Tags             []string
	OutputType       string `json:",omitempty" yaml:",omitempty"`
	UseDict          *bool  `json:",omitempty" yaml:",omitempty"`
	DictURL          string
	Transformer      string         `json:",omitempty" yaml:",omitempty"`
	DataStore        string         `json:",omitempty" yaml:",omitempty"`
	Modified         *Modified      `json:",omitempty" yaml:",omitempty"`
	Stream           *config.Stream `json:",omitempty" yaml:",omitempty"`
	shared.MetaInput `json:",omitempty" yaml:",inline"`
}

//UseDictionary returns true if dictionary can be used
func (m Model) UseDictionary() bool {
	return m.UseDict == nil || *m.UseDict
}

//Init initialises model config
func (m *Model) Init() {
	if len(m.Tags) == 0 {
		m.Tags = []string{"serve"}
	}
	if m.Location == "" {
		m.Location = path.Join(os.TempDir(), m.ID)
	}
	_ = os.MkdirAll(m.Location, file.DefaultDirOsMode)
	m.Modified = &Modified{}
	m.MetaInput.Init()
}

//Validate validates model config
func (m *Model) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("model.ID was empty")
	}
	if m.URL == "" {
		return fmt.Errorf("model.URL was empty")
	}
	return nil
}

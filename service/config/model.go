package config

import (
	"fmt"
	"github.com/viant/afs/file"
	"os"
	"path"
)

type Model struct {
	ID          string
	URL         string
	Location    string
	Tags        []string
	OutputType  string
	UseDict     *bool
	Transformer string
	DataStore   string
	KeyFields   []string
}

func (m Model) UseDictionary() bool {
	return m.UseDict == nil || *m.UseDict
}

func (m *Model) Init() {
	if len(m.Tags) == 0 {
		m.Tags = []string{"serve"}
	}
	if m.Location == "" {
		m.Location = path.Join(os.TempDir(), m.ID)
	}
	_ = os.MkdirAll(m.Location, file.DefaultDirOsMode)

}

func (m *Model) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("model.ID was empty")
	}
	if m.URL == "" {
		return fmt.Errorf("model.URL was empty")
	}
	if m.OutputType == "" {
		return fmt.Errorf("model.OutputType was empty")
	}
	return nil
}

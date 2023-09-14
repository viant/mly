package endpoint

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/viant/afs"
	"github.com/viant/mly/service/config"
	econfig "github.com/viant/mly/service/endpoint/config"
	batchconfig "github.com/viant/mly/service/tfmodel/batcher/config"
	sconfig "github.com/viant/mly/shared/config"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
)

const (
	configURI = "/v1/api/config/"
)

// Config represents an endpoint config
type Config struct {
	config.ModelList      `json:",omitempty" yaml:",inline"`
	sconfig.DatastoreList `json:",omitempty" yaml:",inline"`

	// GlobalBatching provides a default batching configuration if
	// models do not provide their own.
	// If GlobalBatching is provided but a model should not be batching,
	// set the TODO to 0.
	GlobalBatching *batchconfig.BatcherConfig `json:",omitempty" yaml:",omitempty"`

	Endpoint econfig.Endpoint

	EnableMemProf bool
	EnableCPUProf bool

	AllowedSubnet []string `json:",omitempty" yaml:",omitempty"`
}

// Init initialise config
func (c *Config) Init() {
	if c.GlobalBatching != nil {
		c.GlobalBatching.Init()
	}

	c.ModelList.Init(c.GlobalBatching)
	c.DatastoreList.Init()
	c.Endpoint.Init()
}

// Validate validates config
func (c *Config) Validate() error {
	if err := c.ModelList.Validate(); err != nil {
		return err
	}
	if err := c.DatastoreList.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *Config) LoadFromURL(ctx context.Context, URL string, target interface{}) error {
	fs := afs.New()
	reader, err := fs.OpenURL(ctx, URL)
	if err != nil {
		return errors.Wrapf(err, "failed to get config: %v", URL)
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return errors.Wrapf(err, "failed to load config: %v", URL)
	}
	transient := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &transient); err != nil {
		return err
	}
	aMap := map[string]interface{}{}
	yaml.Unmarshal(data, &aMap)
	err = toolbox.DefaultConverter.AssignConverted(target, aMap)
	if err != nil {
		return errors.Wrapf(err, "failed to convert config: %v", URL)
	}
	return nil
}

// NewConfigFromURL creates a new config from URL
func NewConfigFromURL(ctx context.Context, URL string) (*Config, error) {
	cfg := &Config{}
	if err := cfg.LoadFromURL(ctx, URL, cfg); err != nil {
		return nil, err
	}
	cfg.Init()
	return cfg, cfg.Validate()
}

type configHandler struct {
	*Config
}

func (h *configHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	JSON, _ := json.Marshal(h.Config)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}

func NewConfigHandler(config *Config) http.Handler {
	return &configHandler{Config: config}
}

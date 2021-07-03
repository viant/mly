package endpoint

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/viant/afs"
	"github.com/viant/mly/service/config"
	econfig "github.com/viant/mly/service/endpoint/config"
	"github.com/viant/mly/shared/common"
	sconfig "github.com/viant/mly/shared/config"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"io/ioutil"

	"net/http"
)

const (
	configURI = "/v1/api/config/"
)

//Config represents an endpoint config
type Config struct {
	config.ModelList
	sconfig.DatastoreList
	Endpoint      econfig.Endpoint
	AllowedSubnet []string
}

//Init initialise config
func (c *Config) Init() {
	c.ModelList.Init()
	c.DatastoreList.Init()
	c.Endpoint.Init()
}

//Validate validates config
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



type configHandler struct {
	*Config
}

func (h *configHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !common.IsAuthorized(request, h.Config.AllowedSubnet) {
		writer.WriteHeader(http.StatusForbidden)
		return
	}
	JSON, _ := json.Marshal(h.Config)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}

//NewConfigHandler creates a config handler
func NewConfigHandler(config *Config) http.Handler {
	return &configHandler{Config: config}
}


//NewConfigFromURL creates a new config from URL
func NewConfigFromURL(ctx context.Context, URL string) (*Config, error) {
	cfg := &Config{}
	if err :=cfg.LoadFromURL(ctx, URL, cfg); err != nil {
		return nil, err
	}
	cfg.Init()
	return cfg, cfg.Validate()
}



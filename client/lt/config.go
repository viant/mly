package lt

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/afs"
	"github.com/viant/cloudless/data/processor"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

//Field represents
type Field struct {
	Name string
	Type string
}

//Config represents load testing config
type Config struct {
	processor.Config
	Model           string
	Hostname        string
	Port            int
	CacheSize       int
	Fields          []*Field
	NewResponseData func() interface{}
}

func (c *Config) Init(ctx context.Context, fs afs.Service) error {
	c.Config.InitWithNoLimit()
	if c.NewResponseData == nil {
		c.NewResponseData = func() interface{} {
			return map[string]interface{}{}
		}
	}
	return c.Config.Init(ctx, fs)
}

func (c *Config) Validate() error {
	if len(c.Model) == 0 {
		return fmt.Errorf("hostname was empty")
	}
	if c.Port == 0 {
		return fmt.Errorf("port was not set")
	}
	if len(c.Hostname) == 0 {
		return fmt.Errorf("hostname was empty")
	}
	if len(c.Fields) == 0 {
		return fmt.Errorf("fields was empty")
	}
	return c.Config.Validate()
}

//NewConfigFromURL creates a config from URL
func NewConfigFromURL(ctx context.Context, URL string) (*Config, error) {
	fs := afs.New()
	reader, err := fs.OpenURL(ctx, URL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get config: %v", URL)
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load config: %v", URL)
	}
	transient := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &transient); err != nil {
		return nil, err
	}
	aMap := map[string]interface{}{}
	_ = yaml.Unmarshal(data, &aMap)
	cfg := &Config{}
	err = toolbox.DefaultConverter.AssignConverted(cfg, aMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert config: %v", URL)
	}
	if err = cfg.Init(ctx, afs.New()); err != nil {
		return nil, err
	}
	return cfg, cfg.Validate()
}

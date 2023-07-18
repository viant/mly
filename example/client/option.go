package client

import (
	"encoding/json"
	"fmt"

	"github.com/viant/mly/shared/client"
)

type Options struct {
	Host       string `short:"h" long:"host" description:"endpoint host"`
	Port       int    `short:"p" long:"port" description:"endpoint port"`
	Debug      bool   `long:"debug"`
	Model      string `short:"m" long:"model" description:"model"`
	Storable   string `short:"s" long:"storable"`
	PayloadStr string `short:"a" long:"payload"`
	Format     string `short:"f" long:"format" choice:"mly" choice:"json"`
	TimeoutUs  int    `short:"t" long:"timeout"`
	Metrics    bool   `long:"metrics"`
}

type C uint8

const (
	None C = iota
	Single
	Batch
)

func (o *Options) Init() {
	if o.Host == "" {
		o.Host = "localhost"
	}

	if o.Port == 0 {
		o.Port = 8086
	}
}

func (o *Options) Payload() (*CliPayload, error) {
	pl := new(CliPayload)
	c := None
	var err error
	switch o.Format {
	case "json":
		data := make(map[string]interface{})
		err = json.Unmarshal([]byte(o.PayloadStr), &data)
		for k, v := range data {
			switch vt := v.(type) {
			case []int64, []int32, []float32, []float64, []string:
				if c != None && c == Single {
					return nil, fmt.Errorf("inconsistent single/batch (at %s, was %d)", k, c)
				}
				c = Batch

				tt := vt.([]interface{})

				if len(tt) > pl.Batch {
					pl.Batch = len(tt)
				}
			case int64, int32, float32, float64, string:
				if c != None && c == Batch {
					return nil, fmt.Errorf("inconsistent single/batch (at %s, was %d)", k, c)
				}
				c = Single
			}
		}

		pl.Data = data
	default:
		err = Parse(o.PayloadStr, pl)
	}

	if err != nil {
		return nil, err
	}

	return pl, err
}

func (o *Options) Validate() error {
	if o.Model == "" {
		return fmt.Errorf("model was empty")
	}

	return nil
}

func (o *Options) Hosts() []*client.Host {
	return []*client.Host{{Name: o.Host, Port: o.Port}}
}

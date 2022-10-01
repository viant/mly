package client

import (
	"fmt"
	"github.com/viant/mly/shared/client"
)

type Options struct {
	Sa []string `short:"a" long:"sa" description:"sls model input" `
	Sl []string `short:"s" long:"sl" description:"sls/vec model input"`
	X  []string `short:"x" long:"x_input" description:"auxiliary input input"`

	Tv    []string `short:"t" long:"tv" description:"vec model input" `
	Model string   `short:"m" long:"model" description:"model" `
	Host  string   `short:"h" long:"host" description:"endpoint host" `
	Port  int      `short:"p" long:"port" description:"endpoint port" `
}

func (o *Options) Init() {
	if o.Host == "" {
		o.Host = "localhost"
	}
	if o.Port == 0 {
		o.Port = 8086
	}
	if len(o.Tv) > 0 && o.Model == "" {
		o.Model = "vec"
	}
	if len(o.Sa) > 0 && o.Model == "" {
		o.Model = "sls"
	}
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

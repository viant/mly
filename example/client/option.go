package client

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/viant/mly/shared/client"
)

type Options struct {
	Host    string `long:"host" description:"endpoint host"`
	Port    int    `short:"p" long:"port" description:"endpoint port"`
	Address string `long:"address" description:"address overrides host and port"`

	Debug     bool `long:"debug"`
	TimeoutUs int  `short:"t" long:"timeout"`

	Model    string `short:"m" long:"model" description:"model"`
	Storable string `short:"s" long:"storable"`

	CacheMB     int  `long:"cache"`
	NoHashCheck bool `long:"nohash"`

	// The total number of requests sent will be payloads * concurrent * repeats.

	Workers    int `long:"workers"`
	Concurrent int `long:"concurrent"`
	// In the case Workers > 1, Repeats is the total number of times each payload
	// is run, not the number of times each workers sends the paylod.
	Repeats int `long:"repeats" description:"times to repeat all payloads"`

	PayloadStr   []string `short:"a" long:"payload"`
	PayloadPause int      `long:"pause" description:"pause seconds between payloads"`
	PayloadDelay int      `long:"delay" description:"pause seconds from first payload"`

	SkipError bool `long:"skiperrs"`

	NoOutput     bool `long:"noout"`
	Metrics      bool `long:"metrics"`
	ErrorHistory bool `long:"errhist"`

	// Report forces NoOutput and SkipError true, Metrics and ErrorHistory false.
	// Will generate a final JSON object as its only output to stdout.
	// stderr may have other output if Debug is true or there are other errors.
	Report bool `long:"report"`
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

	if o.Repeats <= 0 {
		o.Repeats = 1
	}

	if o.Concurrent <= 0 {
		o.Concurrent = 1
	}

	if o.Workers <= 0 {
		o.Workers = 1
	}

	if o.Report {
		o.SkipError = true
		o.NoOutput = true

		o.Metrics = false
		o.ErrorHistory = false
	}
}

func (o *Options) Payloads() ([]*CliPayload, error) {
	pls := make([]*CliPayload, len(o.PayloadStr))

	for i, payloadStr := range o.PayloadStr {
		pl := new(CliPayload)
		err := Parse(payloadStr, pl)
		if err != nil {
			return nil, err
		}

		pls[i] = pl
	}

	return pls, nil
}

func (o *Options) Validate() error {
	if o.Model == "" {
		return fmt.Errorf("model was empty")
	}

	return nil
}

func (o *Options) Hosts() []*client.Host {
	if o.Address != "" {
		elems := strings.Split(o.Address, ",")
		hosts := make([]*client.Host, len(elems))
		for i, addr := range elems {
			components := strings.Split(addr, ":")

			var domain string
			var port int
			if len(components) == 1 {
				// no port separator, assume domain only
				domain = addr
			} else if len(components) == 2 {
				domain = components[0]
				var err error
				port, err = strconv.Atoi(components[1])
				if err != nil {
					panic(err)
				}
			} else {
				panic(fmt.Sprintf("unknown address: %s", addr))
			}

			hosts[i] = client.NewHost(domain, port)
		}

		return hosts
	}

	return []*client.Host{client.NewHost(o.Host, o.Port)}
}

package tools

import "fmt"

type Options struct {
	Mode      string `short:"m" long:"mode" choice:"discover" choice:"run" description:"mode"`
	Operation string `short:"o" long:"opt" choice:"dictHash" choice:"signature" choice:"layers" choice:"config"`
	SourceURL string `short:"s" long:"src" description:"source location, required by all opt where mode=discover"`
	DestURL   string `short:"d" long:"dest" description:"dest location, required if opt=layers"`
	ConfigURL string `short:"c" long:"config" description:"required if mode=run"`
}

func (o Options) Validate() error {
	if o.Mode == "run" {
		if o.ConfigURL == "" {
			return fmt.Errorf("configurl was empty")
		}

		return nil
	}
	if o.SourceURL == "" {
		return fmt.Errorf("source location was empty")
	}
	if o.Operation == "layers" && o.DestURL == "" {
		return fmt.Errorf("destination location was empty")
	}
	return nil
}

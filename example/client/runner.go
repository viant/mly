package client

import (
	"context"
	"fmt"
	"github.com/viant/mly/example/sls"
	"github.com/viant/mly/example/vec"
	"github.com/viant/mly/shared/client"
)

func RunWithOptions(options *Options) error {
	options.Init()
	if err := options.Validate(); err != nil {
		return err
	}
	srv, err := client.New(options.Model, options.Hosts())
	if err != nil {
		return err
	}

	message := srv.NewMessage()
	defer message.Release()
	response := &client.Response{}

	if len(options.Sa) > 0 {
		message.StringKey("sa", options.Sa[0])
		message.StringKey("sl", options.Sl[0])
		message.StringKey("x", options.X[0])
		response.Data = &sls.Record{}
	} else {
		if len(options.Tv) == 0 {
			return fmt.Errorf("no input data")
		}
		//multi input mode (batch mode)
		message.StringsKey("sl", options.Sl)
		message.StringsKey("tv", options.Tv)
		records := vec.Records{}
		response.Data = &records

	}

	err = srv.Run(context.Background(), message, response)
	if err != nil {
		return err
	}
	return err
}

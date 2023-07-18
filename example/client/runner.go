package client

import (
	"context"
	"fmt"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/service/endpoint/checker"
	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/shared/common/storable"
	"github.com/viant/toolbox"
)

func RunWithOptions(options *Options) error {
	options.Init()
	if err := options.Validate(); err != nil {
		return err
	}

	if options.Model == "" {
		return fmt.Errorf("could not determine model")
	}

	pl, err := options.Payload()
	if err != nil {
		return err
	}

	if options.Debug {
		fmt.Printf("payload:%v\n", pl)
	}

	gm := gmetric.New()

	cli, err := client.New(options.Model, options.Hosts(), client.WithDebug(options.Debug), client.WithGmetrics(gm))
	if err != nil {
		return err
	}

	message := cli.NewMessage()
	defer message.Release()

	pl.SetBatch(message)
	pl.Iterator(func(k string, value interface{}) error {
		return pl.Bind(k, value, message)
	})

	response := &client.Response{}

	storableSrv := storable.Singleton()
	maker, err := storableSrv.Lookup(options.Storable)
	if err != nil {
		if options.Debug {
			fmt.Printf("could not find Storable:\"%s\", building dynamically\n", options.Storable)
		}

		maker = checker.Generated(cli.Config.Datastore.MetaInput.Outputs, pl.Batch, false)
	}

	response.Data = maker()

	ctx := context.Background()
	cancel := func() {}
	if options.TimeoutUs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.TimeoutUs)*time.Microsecond)
	}

	err = cli.Run(ctx, message, response)

	if err != nil && !options.Metrics {
		return err
	}

	cancel()

	if options.Metrics {
		ctrs := gm.OperationCounters()
		toolbox.Dump(ctrs)
		return nil
	}

	toolbox.Dump(response)

	return err
}

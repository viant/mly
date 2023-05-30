package client

import (
	"context"
	"fmt"

	"github.com/viant/mly/example/string_lookup_int"
	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/shared/common"
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

	fmt.Println(pl)

	cli, err := client.New(options.Model, options.Hosts(), client.WithDebug(options.Debug))
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
	storableSrv.Register("sli", func() common.Storable {
		return new(string_lookup_int.Record)
	})

	maker, err := storableSrv.Lookup(options.Storable)
	if err != nil {
		fmt.Printf("could not find Storable:(%s), building dynamically\n", options.Storable)
		maker = Generated(cli.Config.Datastore.MetaInput.Outputs, pl.Batch)
	}

	response.Data = maker()

	err = cli.Run(context.Background(), message, response)
	if err != nil {
		return err
	}

	toolbox.Dump(response)

	return err
}

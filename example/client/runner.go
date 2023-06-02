package client

import (
	"context"
	"fmt"

	slfmodel "github.com/viant/mly/example/transformer/slf/model"
	"github.com/viant/mly/service/endpoint/checker"
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

	if options.Debug {
		fmt.Printf("payload:%v\n", pl)
	}

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

	// in actual client code, any type information should be available within the context of the caller, so using a dynamic
	// type instantiation service is not needed
	storableSrv.Register(slfmodel.Namespace, func() common.Storable {
		return new(slfmodel.Segmented)
	})

	storableSrv.Register("slft_batch", func() common.Storable {
		sa := make([]*slfmodel.Segmented, 0)
		segmenteds := slfmodel.Segmenteds(sa)
		return &segmenteds
	})

	maker, err := storableSrv.Lookup(options.Storable)
	if err != nil {
		if options.Debug {
			fmt.Printf("could not find Storable:\"%s\", building dynamically\n", options.Storable)
		}

		maker = checker.Generated(cli.Config.Datastore.MetaInput.Outputs, pl.Batch, false)
	}

	response.Data = maker()

	err = cli.Run(context.Background(), message, response)
	if err != nil {
		return err
	}

	toolbox.Dump(response)

	return err
}

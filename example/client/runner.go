package client

import (
	"context"
	"fmt"
	"sync"
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

	pls, err := options.Payloads()
	if err != nil {
		return err
	}

	if options.Debug {
		fmt.Printf("payloads:%v\n", pls)
	}

	gm := gmetric.New()

	opts := []client.Option{
		client.WithDebug(options.Debug),
		client.WithGmetrics(gm),
	}

	if options.CacheMB > 0 {
		opts = append(opts, client.WithCacheSize(options.CacheMB))
	}

	if !options.NoHashCheck {
		opts = append(opts, client.WithHashValidation(true))
	}

	cli, err := client.New(options.Model, options.Hosts(), opts...)
	if err != nil {
		return err
	}

	if options.PayloadDelay > 0 {
		time.Sleep(time.Duration(options.PayloadDelay) * time.Second)
	}

	pPause := time.Duration(options.PayloadPause)
	lp := len(pls)

	concurrency := options.Concurrent

	for i, upl := range pls {
		pl := upl
		payloadedRunner := func() error {
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
				cancel()
				return err
			}

			cancel()

			if options.Metrics {
				return nil
			}

			toolbox.Dump(response)
			return nil
		}

		errs := make([]error, concurrency)

		wg := new(sync.WaitGroup)
		wg.Add(concurrency)

		for wi := 0; wi < concurrency; wi++ {
			go func(wi int) {
				defer wg.Done()

				time.Sleep(1)

				err = payloadedRunner()

				if err != nil {
					errs[wi] = err
				}
			}(wi)
		}

		wg.Wait()

		for wi, err := range errs {
			if err != nil {
				return fmt.Errorf("%d:%w", wi, err)
			}
		}

		if i < lp-1 && pPause > 0 {
			time.Sleep(pPause * time.Second)
		}
	}

	if options.Metrics {
		ctrs := gm.OperationCounters()
		toolbox.Dump(ctrs)
	}

	cli.Close()

	if options.ErrorHistory {
		tops := cli.ErrorHistory.TopK()
		for _, t := range tops {
			fmt.Printf("%d %s\n", t.Count, string(t.Data))
		}

	}

	return err
}

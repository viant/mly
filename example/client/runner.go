package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/service/endpoint/checker"
	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
	"github.com/viant/toolbox"
)

// Use CustomMakerRegistry with --maker to use a specific entity for Response.Data.
var CustomMakerRegistry *customMakerRegistry = new(customMakerRegistry)

func RunWithOptions(runOpts *Options) error {
	runOpts.Init()
	if err := runOpts.Validate(); err != nil {
		return err
	}

	if runOpts.Model == "" {
		return fmt.Errorf("could not determine model")
	}

	payloads, err := runOpts.Payloads()
	if err != nil {
		return err
	}

	if runOpts.Debug {
		fmt.Printf("payloads:%v\n", payloads)
	}

	gm := gmetric.New()

	cliOpts := []client.Option{
		client.WithDebug(runOpts.Debug),
		client.WithGmetrics(gm),
	}

	if runOpts.CacheMB > 0 {
		cliOpts = append(cliOpts, client.WithCacheSize(runOpts.CacheMB))
	}

	if !runOpts.NoHashCheck {
		cliOpts = append(cliOpts, client.WithHashValidation(true))
	}

	cli, err := client.New(runOpts.Model, runOpts.Hosts(), cliOpts...)
	if err != nil {
		return err
	}

	if runOpts.PayloadDelay > 0 {
		time.Sleep(time.Duration(runOpts.PayloadDelay) * time.Second)
	}

	storableSrv := storable.Singleton()

	var maker func(int) func() common.Storable
	storableMaker, err := storableSrv.Lookup(runOpts.Storable)
	if err != nil {
		if runOpts.Debug {
			log.Printf("could not find Storable:\"%s\", building dynamically", runOpts.Storable)
		}

		maker = func(batch int) func() common.Storable {
			return checker.Generated(cli.Config.Datastore.MetaInput.Outputs, batch, false)
		}

		err = nil
	} else {
		maker = func(int) func() common.Storable {
			return storableMaker
		}
	}

	// wrap return value to interface{}
	dataSetter := func(i int) func() interface{} {
		return func() interface{} {
			return maker(i)()
		}
	}

	if runOpts.CustomMaker != "" {
		custom, ok := CustomMakerRegistry.registry[runOpts.CustomMaker]
		if !ok {
			if runOpts.Debug {
				fmt.Printf("no such custom maker:\"%s\"\n", runOpts.CustomMaker)
			}
		} else {
			dataSetter = func(int) func() interface{} { return custom }
		}
	}

	lenPayloads := len(payloads)

	fullyCompleted := new(sync.WaitGroup)
	fullyCompleted.Add(lenPayloads * runOpts.Repeats)

	fchan := make(chan runContext, 1)
	closed := make(chan struct{}, 1)

	numWorkers := runOpts.Workers

	var echan chan error
	if !runOpts.SkipError {
		echan = make(chan error, numWorkers)
	}

	concurrency := runOpts.Concurrent
	for w := 0; w < numWorkers; w++ {
		go worker(w, echan, fchan, closed, concurrency, fullyCompleted)
	}

	report := Report{
		Start: time.Now(),
		Runs:  make([]RepeatSet, runOpts.Repeats),
	}

	pause := time.Duration(runOpts.PayloadPause)
	for r := 0; r < runOpts.Repeats; r++ {
		report.Runs[r] = RepeatSet{r, make([]WorkerPayload, lenPayloads)}
		rs := &report.Runs[r]

		for i, pload := range payloads {
			rs.WPayloads[i] = WorkerPayload{Payload: pload}
			rd := &rs.WPayloads[i]
			payloadedRunner := makePayloadRunner(cli, pload, runOpts, dataSetter)

			fchan <- runContext{
				WP: rd,
				Fn: payloadedRunner,
			}

			if echan != nil {
				select {
				case err = <-echan:
					return err
				default:
				}
			}

			if i < lenPayloads-1 && pause > 0 {
				time.Sleep(pause * time.Second)
			}
		}
	}

	if echan != nil {
		waitErr := make(chan struct{}, 0)
		go func() {
			fullyCompleted.Wait()
			waitErr <- struct{}{}
		}()
		select {
		case err = <-echan:
			return err
		case <-waitErr:
		}
	} else {
		fullyCompleted.Wait()
	}

	for w := 0; w < numWorkers; w++ {
		closed <- struct{}{}
	}

	cli.Close()

	report.End = time.Now()

	opcs := gm.OperationCounters()
	report.Metrics = opcs

	if runOpts.Metrics {
		toolbox.Dump(opcs)
	}

	if runOpts.ErrorHistory {
		tops := cli.ErrorHistory.TopK()
		for _, t := range tops {
			fmt.Printf("%d %s\n", t.Count, string(t.Data))
		}
	}

	if runOpts.Report {
		toolbox.Dump(report)
	}

	return err
}

type runContext struct {
	WP *WorkerPayload
	Fn func() (*client.Response, error)
}

func worker(worker int, echan chan error, fchan chan runContext, closed chan struct{},
	concurrency int, allDone *sync.WaitGroup) {

	for {
		select {
		case rc := <-fchan:
			wg := new(sync.WaitGroup)
			wg.Add(concurrency)

			wp := rc.WP
			wp.Worker = worker
			wp.Runs = make([]WPRun, concurrency)

			payloadedRunner := rc.Fn
			for wi := 0; wi < concurrency; wi++ {
				go func(wi int) {
					resp, err := payloadedRunner()
					if echan != nil && err != nil {
						echan <- err
					}

					wp.Runs[wi] = WPRun{resp, err}
					wg.Done()
				}(wi)
			}
			wg.Wait()

			allDone.Done()
		case <-closed:
			break
		}
	}
}

func makePayloadRunner(cli *client.Service, pl *CliPayload, runOpts *Options,
	builder func(int) func() interface{}) func() (*client.Response, error) {

	maker := builder(pl.Batch)

	return func() (*client.Response, error) {
		message := cli.NewMessage()

		pl.SetBatch(message)
		pl.Iterator(func(k string, value interface{}) error {
			return pl.Bind(k, value, message)
		})

		response := &client.Response{}
		response.Data = maker()

		ctx := context.Background()
		cancel := func() {}
		if runOpts.TimeoutUs > 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(runOpts.TimeoutUs)*time.Microsecond)
		}

		err := cli.Run(ctx, message, response)

		cancel()

		if err != nil {
			return nil, err
		}

		if !runOpts.NoOutput {
			toolbox.Dump(response)
		}

		return response, nil
	}
}

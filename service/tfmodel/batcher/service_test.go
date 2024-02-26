package batcher

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/gmetric"
	serrs "github.com/viant/mly/service/errors"
	"github.com/viant/mly/service/tfmodel/batcher/adjust"
	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/service/tfmodel/evaluator/test"
	"github.com/viant/mly/shared/common/permute"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/toolbox"
)

type mockEvaluator struct {
	Entered chan struct{}
	Exited  chan struct{}
}

func (m *mockEvaluator) Evaluate(ctx context.Context, params []interface{}) ([]interface{}, error) {
	m.Entered <- struct{}{}
	fmt.Println("mock entered")
	m.Exited <- struct{}{}
	fmt.Println("mock exited")
	return []interface{}{}, nil
}

func (m mockEvaluator) Close() error {
	return nil
}

func createBSMeta() ServiceMeta {
	s := gmetric.New()
	m := &stat.GMeter{
		Service:        s,
		Unit:           time.Microsecond,
		RecentDuration: 5 * time.Second,
		NumRecent:      12,
	}

	return ServiceMeta{
		queueMetric:      m.Op("test", "queue", ""),
		dispatcherMetric: m.MOp("test", "dispatcher", "", NewDispatcherP()),
		dispatcherLoop:   m.Op("test", "dispatcherLoop", ""),
		blockQDelay:      m.Op("test", "blockQ", ""),
		inputQDelay:      m.Op("test", "inputQ", ""),
	}
}

func TestServiceShedding(t *testing.T) {
	mockEval := &mockEvaluator{
		Entered: make(chan struct{}, 0),
		Exited:  make(chan struct{}, 0),
	}

	mockIn := mockEval.Entered
	mockOut := mockEval.Exited

	bc := config.BatcherConfig{
		MaxBatchSize:            1,
		MaxQueuedBatches:        1,
		MaxEvaluatorConcurrency: 1,
	}

	batchSrv := NewBatcher(mockEval, 1, bc, createBSMeta())
	batchSrv.Verbose = &config.V{
		ID:     "test-shedding",
		Input:  true,
		Output: true,
	}

	qs := batchSrv.newQueueService()
	ds := batchSrv.newDispatcher()

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}, {"b"}})

	ctx := context.Background()

	// this will read from inputQ
	go ds.dispatch()
	// this will write to inputQ and wait for blockQ
	b1, err := batchSrv.queue(ctx, feeds)
	// this means ds.dispatch() is done
	require.Nil(t, err)

	// once ds.dispatch() is done, batchQ is full
	qs.dispatch()

	<-mockIn
	// this means evaluator should have started, so shedding should be clear

	go ds.dispatch()
	b2, err := batchSrv.queue(ctx, feeds)
	require.Nil(t, err)

	// this will block for free evaluator
	// and activate shedding
	go qs.dispatch()

	select {
	case <-mockIn:
		t.Error("should not have run evaluator")
	default:
	}

	shedBatch, err := batchSrv.queue(ctx, feeds)
	// this fails due to shedding being on
	require.NotNil(t, err)
	require.Nil(t, shedBatch)

	<-mockOut
	// Evaluator finished, should unblocks qs
	<-b1.channel

	// this will block until the go qs.dispatch() from earlier finishes
	fmt.Println("b2 <-mockIn")
	<-mockIn
	// this means evaluator should have started, so shedding should be clear
	// b2 should be in eval mode

	// this should make qs wait for free
	go qs.dispatch()

	go ds.dispatch()
	b3, err := batchSrv.queue(ctx, feeds)
	require.Nil(t, err)
	// b3 should've been dispatched

	<-mockOut
	// b2 should finish
	fmt.Println("b2 wait")
	<-b2.channel

	fmt.Println("b3 done")
	<-mockIn
	<-mockOut
	fmt.Println("b3 wait")
	go qs.dispatch()
	<-b3.channel
}

func TestServiceBatchMax(t *testing.T) {
	signature, evaluator, met := test.TLoadEvaluator(t, "example/model/string_lookups_int_model")

	batchSrv := NewBatcher(evaluator, len(signature.Inputs), config.BatcherConfig{
		MaxBatchSize: 100,
		BatchWait:    time.Millisecond * 1,
	}, createBSMeta())

	batchSrv.Verbose = &config.V{
		ID:     "test",
		Input:  true,
		Output: true,
	}
	batchSrv.Start()

	fmt.Printf("%+v\n", batchSrv)

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}, {"b"}})
	feeds = append(feeds, [][]string{{"c"}, {"d"}})
	singleValue, err := evaluator.Evaluate(context.Background(), feeds)
	assert.Nil(t, err)
	fmt.Printf("%v\n", singleValue)

	feeds3 := make([]interface{}, 0)
	feeds3 = append(feeds3, [][]string{{"e"}, {"f"}, {"g"}})
	feeds3 = append(feeds3, [][]string{{"h"}, {"i"}, {"j"}})
	sv3, err := evaluator.Evaluate(context.Background(), feeds3)
	assert.Nil(t, err)
	fmt.Printf("%v\n", sv3)

	preQ := time.Now()

	wg := new(sync.WaitGroup)

	ctx := context.Background()

	var errors int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		wg.Add(1)

		go func() {
			sb, err := batchSrv.queue(ctx, feeds)
			assert.Nil(t, err)

			wait := time.Now()
			select {
			case r := <-sb.channel:
				if false {
					fmt.Printf("preQ:%s wait:%s\n", time.Since(preQ), time.Since(wait))
				}
				assert.Equal(t, singleValue, r)
			case err = <-sb.ec:
				fmt.Printf("%s\n", err)
				atomic.AddInt32(&errors, 1)
			}

			wg.Done()
		}()

		go func() {
			sb, err := batchSrv.queue(ctx, feeds3)
			assert.Nil(t, err)

			wait := time.Now()
			select {
			case r := <-sb.channel:
				if false {
					fmt.Printf("preQ:%s wait:%s\n", time.Since(preQ), time.Since(wait))
				}
				assert.Equal(t, sv3, r)
			case err = <-sb.ec:
				fmt.Printf("%s\n", err)
				atomic.AddInt32(&errors, 1)
			}

			wg.Done()
		}()
	}

	assert.Equal(t, int32(0), errors, "got errors")

	wg.Wait()
	toolbox.Dump(met)

	batchSrv.Close()
}

func TestClose(t *testing.T) {
	// TODO add test to make sure there are no leaks or early closes if there is
	// any queues not empty but Close() is called.
}

func BenchmarkServiceParallel(b *testing.B) {
	bnil := func(err error) {
		if err != nil {
			b.Error(err)
		}
	}

	signature, evaluator, _ := test.LoadEvaluator("example/model/string_lookups_int_model", bnil, bnil)

	bcfg := config.BatcherConfig{
		MaxBatchSize: 100,
		BatchWait:    time.Millisecond * 1,
	}

	batcher := NewBatcher(evaluator, len(signature.Inputs), bcfg, createBSMeta())
	batcher.Start()

	feeds2 := make([]interface{}, 0)
	feeds2 = append(feeds2, [][]string{{"a"}, {"b"}})
	feeds2 = append(feeds2, [][]string{{"c"}, {"d"}})

	evaluator.Evaluate(context.Background(), feeds2)

	preQ := time.Now()
	ctx := context.Background()

	wg := new(sync.WaitGroup)

	var errors int32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg.Add(1)

			go func() {
				sb, err := batcher.queue(ctx, feeds2)
				wait := time.Now()
				select {
				case <-sb.channel:
					if false {
						fmt.Printf("preQ:%s wait:%s\n", time.Since(preQ), time.Since(wait))
					}
				case err = <-sb.ec:
					fmt.Printf("%s\n", err)
					atomic.AddInt32(&errors, 1)
				}

				wg.Done()
			}()
		}
		wg.Wait()
	})

	if errors > 0 {
		b.Error("had errors")
	}

	batcher.Close()
}

func BenchmarkServiceVariantParallel(ob *testing.B) {
	mbsVariants := []int{5, 10, 20}
	bwVariants := []time.Duration{0, time.Microsecond * 1000}
	mecVariants := []int{0, 5}
	mqbVariants := []int{0}

	adjIVars := []time.Duration{time.Microsecond * 200}
	adjMaxVars := []uint32{0, 10}

	iter := permute.NewPermuter(
		[][]int{mbsVariants, mqbVariants, mecVariants},
		[][]time.Duration{bwVariants, adjIVars},
		[][]uint32{adjMaxVars},
	)

	c, variant := iter.Next()
	for c {
		mbs := variant.Ints[0]
		mqb := variant.Ints[1]
		mec := variant.Ints[2]

		bw := variant.Durs[0]
		adjI := variant.Durs[1]

		adjMax := variant.Uint32s[0]

		runID := fmt.Sprintf("MBS-%d-MQB-%d-BW-%s-MEC-%d-AdjI-%s-AdjMax-%d", mbs, mqb, bw, mec, adjI, adjMax)

		ob.Run(runID, func(b *testing.B) {
			var adj *adjust.AdjustConfig
			if adjMax > 0 {
				adj = &adjust.AdjustConfig{
					Increment: adjI,
					Max:       adjMax,
				}
			}

			bcfg := config.BatcherConfig{
				MaxBatchSize:            mbs,
				BatchWait:               bw,
				MaxQueuedBatches:        mqb,
				MaxEvaluatorConcurrency: mec,
				TimeoutAdjustments:      adj,
				//Verbose:                 &config.V{ID: runID},
			}

			bcfg.Init()

			bnil := func(err error) {
				if err != nil {
					b.Error(err)
				}
			}

			signature, evaluator, evMeta := test.LoadEvaluator("example/model/string_lookups_int_model", bnil, bnil)

			var negOneUI64 uint64 = 0
			maxEval := &negOneUI64

			cntr := evMeta.TFMetric().Counters[1].Counter.Custom
			switch actual := cntr.(type) {
			case *stat.Occupancy:
				maxEval = &(*stat.Occupancy)(actual).Max
			default:
			}

			bsMeta := createBSMeta()
			batcher := NewBatcher(evaluator, len(signature.Inputs), bcfg, bsMeta)
			batcher.Start()

			feeds2 := make([]interface{}, 0)
			feeds2 = append(feeds2, [][]string{{"a"}, {"b"}})
			feeds2 = append(feeds2, [][]string{{"c"}, {"d"}})

			evaluator.Evaluate(context.Background(), feeds2)
			ctx := context.Background()

			agg := make(chan []time.Duration, 100)
			aggDone := make(chan struct{}, 0)

			adi := 0
			allDurs := make([]time.Duration, b.N)

			go func() {
				c := true
				for c {
					select {
					case ds := <-agg:
						for _, d := range ds {
							allDurs[adi] = d
							adi++
						}
					case <-aggDone:
						c = false
					}
				}

				c = true
				for c {
					select {
					case ds := <-agg:
						for _, d := range ds {
							allDurs[adi] = d
							adi++
						}
					default:
						c = false
					}
				}

				aggDone <- struct{}{}
			}()

			var numErrs, overloads uint64
			b.RunParallel(func(pb *testing.PB) {
				times := make([]time.Duration, 0)

				for pb.Next() {
					start := time.Now()
					sb, err := batcher.queue(ctx, feeds2)
					if err != nil {
						if errors.Is(err, serrs.OverloadedError) {
							atomic.AddUint64(&overloads, 1)
						} else {
							atomic.AddUint64(&numErrs, 1)
						}
						return
					}

					select {
					case <-sb.channel:
					case err = <-sb.ec:
						atomic.AddUint64(&numErrs, 1)
					}

					times = append(times, time.Now().Sub(start))
				}

				agg <- times
			})

			aggDone <- struct{}{}
			<-aggDone

			sort.Slice(allDurs, func(i, j int) bool { return allDurs[i] < allDurs[j] })

			b.ReportMetric(float64(*maxEval), "max_eval")

			b.ReportMetric(float64(allDurs[0]), "dur_min")

			ldurs := len(allDurs)

			b.ReportMetric(float64(allDurs[ldurs-1]), "dur_max")
			b.ReportMetric(float64(allDurs[ldurs/2]), "dur_median")

			b.ReportMetric(float64(allDurs[int(float64(ldurs)*0.05)]), "dur_p05")
			b.ReportMetric(float64(allDurs[int(float64(ldurs)*0.95)]), "dur_p95")

			b.ReportMetric(float64(numErrs), "errors")
			b.ReportMetric(float64(overloads), "overloads")

			batcher.Close()
		})

		c, variant = iter.Next()
	}
}

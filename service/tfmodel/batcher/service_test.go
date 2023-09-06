package batcher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/service/tfmodel/evaluator/test"
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
	return ServiceMeta{
		queueMetric:      s.OperationCounter("test", "queue", "", time.Microsecond, time.Minute, 2),
		dispatcherMetric: s.MultiOperationCounter("test", "dispatcher", "", time.Microsecond, time.Minute, 2, NewDispatcherP()),
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

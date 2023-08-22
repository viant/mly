package tfmodel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/service/tfmodel/batcher"
	"github.com/viant/toolbox"
)

func TestBatcherBatchMax(t *testing.T) {
	signature, evaluator, met := tLoadEvaluator(t, "example/model/string_lookups_int_model")
	batchSrv := NewBatcher(evaluator, len(signature.Inputs), batcher.BatcherConfig{
		MaxBatchCounts: 3,
		MaxBatchSize:   100,
		MaxBatchWait:   time.Millisecond * 1,
	})

	batchSrv.Verbose = &batcher.V{"test", true}
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

	var errors int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		wg.Add(1)

		go func() {
			sb, err := batchSrv.Queue(feeds)
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
			sb, err := batchSrv.Queue(feeds3)
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

func BenchmarkBatcherParallel(b *testing.B) {
	bnil := func(err error) {
		if err != nil {
			b.Error(err)
		}
	}

	signature, evaluator, _ := loadEvaluator("example/model/string_lookups_int_model", bnil, bnil)

	bcfg := batcher.BatcherConfig{
		MaxBatchSize:   100,
		MaxBatchCounts: 80,
		MaxBatchWait:   time.Millisecond * 1,
	}

	batcher := NewBatcher(evaluator, len(signature.Inputs), bcfg)

	feeds2 := make([]interface{}, 0)
	feeds2 = append(feeds2, [][]string{{"a"}, {"b"}})
	feeds2 = append(feeds2, [][]string{{"c"}, {"d"}})

	evaluator.Evaluate(context.Background(), feeds2)

	preQ := time.Now()

	wg := new(sync.WaitGroup)

	var errors int32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg.Add(1)

			go func() {
				sb, err := batcher.Queue(feeds2)
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

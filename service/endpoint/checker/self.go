package checker

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/client"
	"github.com/viant/toolbox"
)

func SelfTest(host []*client.Host, timeout time.Duration, modelID string, usesTransformer bool, inputs_ []*shared.Field, tp config.TestPayload, outputs []*shared.Field, debug bool) error {
	cli, err := client.New(modelID, host, client.WithDebug(true))
	if err != nil {
		return fmt.Errorf("%s:%w", modelID, err)
	}

	inputs := cli.Config.Datastore.MetaInput.Inputs

	// generate payload

	var testData map[string]interface{}
	var batchSize int
	if len(tp.Batch) > 0 {
		for k, v := range tp.Batch {
			testData[k] = v
			batchSize = len(v)
		}
	} else {
		if len(tp.Single) > 0 {
			testData = tp.Single

			for _, field := range inputs {
				n := field.Name
				sv, ok := tp.Single[n]
				switch field.DataType {
				case "int", "int32", "int64":
					if !ok {
						testData[n] = rand.Int31()
					} else {
						switch tsv := sv.(type) {
						case string:
							testData[n], err = strconv.Atoi(tsv)
							if err != nil {
								return err
							}
						}
					}
				case "float", "float32", "float64":
					testData[n] = rand.Float32()
				default:
					if !ok {
						testData[n] = fmt.Sprintf("test-%d", rand.Int31())
					} else {
						testData[n] = toolbox.AsString(sv)
					}
				}
			}
		} else {
			testData = make(map[string]interface{})
			for _, field := range inputs {
				n := field.Name
				switch field.DataType {
				case "int", "int32", "int64":
					testData[n] = rand.Int31()
				case "float", "float32", "float64":
					testData[n] = rand.Float32()
				default:
					testData[n] = fmt.Sprintf("test-%d", rand.Int31())
				}
			}
		}

		if tp.SingleBatch {
			for _, field := range inputs {
				fn := field.Name
				tv := testData[fn]
				switch field.DataType {
				case "int", "int32", "int64":
					var v int
					switch atv := tv.(type) {
					case int:
						v = atv
					case int32:
					case int64:
						v = int(atv)
					default:
						return fmt.Errorf("test data malformed: %s expected int-like, found %T", fn, tv)
					}

					b := [1]int{v}
					testData[fn] = b[:]
				case "float", "float32", "float64":
					var v float32
					switch atv := tv.(type) {
					case float32:
						v = atv
					case float64:
						v = float32(atv)
					default:
						return fmt.Errorf("test data malformed: %s expected float32-like, found %T", fn, tv)
					}

					b := [1]float32{v}
					testData[fn] = b[:]
				default:
					switch atv := tv.(type) {
					case string:
						b := [1]string{atv}
						testData[fn] = b[:]
					default:
						return fmt.Errorf("test data malformed: %s expected string-like, found %T", fn, tv)
					}
				}
			}

			batchSize = 1
		}
	}

	if debug {
		log.Printf("[%s test] batchSize:%d %+v", modelID, batchSize, testData)
	}

	msg := cli.NewMessage()
	defer msg.Release()

	if batchSize > 0 {
		msg.SetBatchSize(batchSize)
	}

	for k, vs := range testData {
		switch at := vs.(type) {
		case []float32:
			msg.FloatsKey(k, at)
		case []float64:
			rat := make([]float32, len(at))
			for i, v := range at {
				rat[i] = float32(v)
			}
			msg.FloatsKey(k, rat)
		case float32:
			msg.FloatKey(k, at)
		case float64:
			msg.FloatKey(k, float32(at))

		case []int:
			msg.IntsKey(k, at)
		case []int32:
			rat := make([]int, len(at))
			for i, v := range at {
				rat[i] = int(v)
			}
			msg.IntsKey(k, rat)
		case []int64:
			rat := make([]int, len(at))
			for i, v := range at {
				rat[i] = int(v)
			}
			msg.IntsKey(k, rat)

		case int:
			msg.IntKey(k, at)
		case int32:
			msg.IntKey(k, int(at))
		case int64:
			msg.IntKey(k, int(at))

		case []string:
			msg.StringsKey(k, at)
		case string:
			msg.StringKey(k, at)

		default:
			return fmt.Errorf("%s:could not form payload %T (%+v)", modelID, at, at)
		}
	}

	resp := new(client.Response)
	// see if there is a transform
	// if there is, trigger the transform with mock data?
	resp.Data = Generated(outputs, batchSize, usesTransformer)()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// send response
	err = cli.Run(ctx, msg, resp)

	if err != nil {
		return fmt.Errorf("%s:Run():%v", modelID, err)
	}

	return nil
}

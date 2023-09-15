package stream

import (
	"bytes"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/tapper/config"
	"github.com/viant/tapper/io"
	tlog "github.com/viant/tapper/log"
	"github.com/viant/tapper/msg"
	"github.com/viant/tapper/msg/json"
)

type dictProvider func() *common.Dictionary
type outputsProvider func() []domain.Output

// Service is used to log request inputs to model outputs without an output
// transformer, in JSON format.
//
// The input values will be directly inlined into the resulting JSON.
// The outputs will be provided as properties in the resulting JSON, with
// the keys as the output Tensor names.
//
// If the dimensions of the output from the model are [1, numOutputs, 1] (single
// request), the value in the JSON object will be a scalar.
// If the dimensions of the output from the model are [batchSize, numOutputs, 1],
// (batch request), the value in the JSON object will be a list of scalars of
// length batchSize.
// If the dimensions of the output from the model are [1, numOutputs, outDims],
// (single request), the value of the JSON object will be a list of scalars of
// length outDims.
// If the dimensions of the output from the model are [batchSize, numOutputs, outDims],
// (batch request), the value of the JSON object will be a list of objects of length
// batchSize, where each object has a property with key "output" and value a
// list of scalars of length outDims.
type Service struct {
	modelID string

	config      *config.Stream
	logger      *tlog.Logger
	msgProvider *msg.Provider

	dictProvider    dictProvider
	outputsProvider outputsProvider

	logMetric          *gmetric.Operation
	logCounterNoData   *gmetric.Counter
	logCounterNoSample *gmetric.Counter
	logCounterNoEnd    *gmetric.Counter
	logCounterComplete *gmetric.Counter
}

func NewService(modelID string, streamCfg *config.Stream, afsv afs.Service, dp dictProvider, op outputsProvider, m *gmetric.Service) (*Service, error) {
	uuid := getStreamID()
	logger, err := tlog.New(streamCfg, uuid, afs.New())
	if err != nil {
		return nil, err
	}

	location := reflect.TypeOf(Service{}).PkgPath()

	s := &Service{
		modelID:         modelID,
		config:          streamCfg,
		logger:          logger,
		msgProvider:     msg.NewProvider(2048, 32, json.New),
		dictProvider:    dp,
		outputsProvider: op,

		logMetric:          m.MultiOperationCounter(location, modelID+"LogPerf", modelID+" tlog", time.Microsecond, time.Minute, 2, stat.ErrorOnly()),
		logCounterNoData:   m.Counter(location, modelID+"LogPerf_NoData", modelID+" tlog no data"),
		logCounterNoSample: m.Counter(location, modelID+"LogPerf_NoSample", modelID+" tlog skipped logging"),
		logCounterNoEnd:    m.Counter(location, modelID+"LogPerf_NoEnd", modelID+" tlog missing closing bracket"),
		logCounterComplete: m.Counter(location, modelID+"LogPerf_Done", modelID+" tlog completed"),
	}

	return s, nil
}

func (s *Service) Log(data []byte, output interface{}, timeTaken time.Duration) {
	if len(data) == 0 {
		s.logCounterNoData.Increment()
		return
	}

	if !s.config.CanSample() {
		s.logCounterNoSample.Increment()
		return
	}

	hasBatchSize := bytes.Contains(data, []byte("batch_size"))
	begin := bytes.IndexByte(data, '{')
	end := bytes.LastIndexByte(data, '}')
	if end == -1 {
		s.logCounterNoEnd.Increment()
		return
	}

	onDone := s.logMetric.Begin(time.Now())
	stats := stat.NewValues()
	defer func() { onDone(time.Now(), stats.Values()...) }()

	// procedurally build the JSON string
	tmsg := s.msgProvider.NewMessage()
	defer tmsg.Free()

	// include original json from request body
	// remove all newlines as they break JSONL
	singleLineBody := bytes.ReplaceAll(data[begin+1:end], []byte("\n"), []byte(" "))
	singleLineBody = bytes.ReplaceAll(singleLineBody, []byte("\r"), []byte(" "))
	tmsg.Put(singleLineBody)

	// add some metadata
	tmsg.PutByte(',')

	tmsg.PutInt("eval_duration", int(timeTaken.Microseconds()))
	if bytes.Index(data, []byte("timestamp")) == -1 {
		tmsg.PutString("timestamp", time.Now().In(time.UTC).Format("2006-01-02 15:04:05.000-07"))
	}

	dict := s.dictProvider()

	if dict != nil {
		tmsg.PutInt("dict_hash", int(dict.Hash))
	}

	outputs := s.outputsProvider()
	writeObject(tmsg, hasBatchSize, output, outputs)

	if err := s.logger.Log(tmsg); err != nil {
		stats.Append(err)
		log.Printf("[%s log] failed to log: %v\n", s.modelID, err)
	}

	s.logCounterComplete.Increment()
}

func writeObject(tmsg msg.Message, hasBatchSize bool, output interface{}, outputs []domain.Output) {
	if value, ok := output.([]interface{}); ok {
		for outputIdx, v := range value {
			outputName := outputs[outputIdx].Name

			switch actual := v.(type) {
			case [][]string:
				switch len(actual) {
				case 0:
				case 1:
					outVec := actual[0]
					if hasBatchSize || len(outVec) > 1 {
						tmsg.PutStrings(outputName, outVec)
					} else {
						tmsg.PutString(outputName, outVec[0])
					}
				default:
					lAct := len(actual[0])
					multiOutDims := lAct > 1
					if multiOutDims {
						c := make([]io.Encoder, lAct)
						for i, v := range actual {
							c[i] = KVStrings(v)
						}
						tmsg.PutObjects(outputName, c)
					} else {
						var stringSlice = make([]string, len(actual))
						for i, vec := range actual {
							stringSlice[i] = vec[0]
						}

						tmsg.PutStrings(outputName, stringSlice)
					}
				}
			case [][]int64:
				switch len(actual) {
				case 0:
				case 1:
					outVec := actual[0]
					lenOutVec := len(outVec)
					if hasBatchSize || lenOutVec > 1 {
						var c []int
						if lenOutVec > 1 {
							c = make([]int, lenOutVec)
							for i, v := range outVec {
								c[i] = int(v)
							}
						} else {
							c = []int{int(outVec[0])}
						}

						tmsg.PutInts(outputName, c)
					} else {
						tmsg.PutInt(outputName, int(actual[0][0]))
					}
				default:
					lAct := len(actual[0])
					multiOutDims := lAct > 1
					if multiOutDims {
						c := make([]io.Encoder, lAct)
						for i, v := range actual {
							t := make([]int, lAct)
							for ii, vv := range v {
								t[ii] = int(vv)
							}
							c[i] = KVInts(t)
						}
						tmsg.PutObjects(outputName, c)
					} else {
						var ints = make([]int, len(actual))
						for i, vec := range actual {
							ints[i] = int(vec[0])
						}
						tmsg.PutInts(outputName, ints)
					}
				}
			case [][]float32:
				switch len(actual) {
				case 0:
				case 1:
					outVec := actual[0]
					lenOutVec := len(outVec)
					if hasBatchSize || lenOutVec > 1 {
						var c []float64
						if lenOutVec > 1 {
							c = make([]float64, lenOutVec)
							for i, v := range outVec {
								c[i] = float64(v)
							}
						} else {
							c = []float64{float64(outVec[0])}
						}

						tmsg.PutFloats(outputName, c)
					} else {
						tmsg.PutFloat(outputName, float64(actual[0][0]))
					}
				default:
					lAct := len(actual[0])
					multiOutDims := lAct > 1
					if multiOutDims {
						c := make([]io.Encoder, lAct)
						for i, v := range actual {
							t := make([]float64, lAct)
							for ii, vv := range v {
								t[ii] = float64(vv)
							}
							c[i] = KVFloat64s(t)
						}
						tmsg.PutObjects(outputName, c)
					} else {
						var ints = make([]float64, len(actual))
						for i, vec := range actual {
							ints[i] = float64(vec[0])
						}
						tmsg.PutFloats(outputName, ints)
					}
				}
			}
		}
	}
}

func getStreamID() string {
	ID := ""
	if UUID, err := uuid.NewUUID(); err == nil {
		ID = UUID.String()
	}
	if hostname, err := os.Hostname(); err == nil {
		if host, err := common.GetHostIPv4(hostname); err == nil {
			ID = host
		}
	}
	return ID
}

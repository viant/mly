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
	tlog "github.com/viant/tapper/log"
	"github.com/viant/tapper/msg"
	"github.com/viant/tapper/msg/json"
)

type dictProvider func() *common.Dictionary
type outputsProvider func() []domain.Output

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
	if value, ok := output.([]interface{}); ok {
		for outputIdx, v := range value {
			outputName := outputs[outputIdx].Name

			switch actual := v.(type) {
			case [][]string:
				if len(actual) > 0 {
					switch len(actual) {
					case 0:
					case 1:
						if hasBatchSize {
							tmsg.PutStrings(outputName, []string{actual[0][0]})
						} else {
							tmsg.PutString(outputName, actual[0][0])
						}
					default:
						var stringSlice = make([]string, len(actual))
						for i, vec := range actual {
							stringSlice[i] = vec[0]
						}
						tmsg.PutStrings(outputName, stringSlice)
					}
				}
			case [][]int64:
				if len(actual) > 0 {
					switch len(actual) {
					case 0:
					case 1:
						if hasBatchSize {
							tmsg.PutInts(outputName, []int{int(actual[0][0])})
						} else {
							tmsg.PutInt(outputName, int(actual[0][0]))
						}
					default:
						var ints = make([]int, len(actual))
						for i, vec := range actual {
							ints[i] = int(vec[0])
						}
						tmsg.PutInts(outputName, ints)
					}
				}
			case [][]float32:
				if len(actual) > 0 {
					switch len(actual) {
					case 0:
					case 1:
						if hasBatchSize {
							tmsg.PutFloats(outputName, []float64{float64(actual[0][0])})
						} else {
							tmsg.PutFloat(outputName, float64(actual[0][0]))
						}
					default:
						var floats = make([]float64, len(actual))
						for i, vec := range actual {
							floats[i] = float64(vec[0])
						}
						tmsg.PutFloats(outputName, floats)
					}
				}
			}
		}
	}

	if err := s.logger.Log(tmsg); err != nil {
		stats.Append(err)
		log.Printf("[%s log] failed to log: %v\n", s.modelID, err)
	}

	s.logCounterComplete.Increment()
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

package stream

import (
	"bytes"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/viant/afs"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
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
}

func NewService(modelID string, streamCfg *config.Stream, afsv afs.Service, dp dictProvider, op outputsProvider) (*Service, error) {
	uuid := getStreamID()
	logger, err := tlog.New(streamCfg, uuid, afs.New())
	if err != nil {
		return nil, err
	}

	s := &Service{
		modelID:         modelID,
		config:          streamCfg,
		logger:          logger,
		msgProvider:     msg.NewProvider(2048, 32, json.New),
		dictProvider:    dp,
		outputsProvider: op,
	}

	return s, nil
}

func (s *Service) Log(data []byte, output interface{}, timeTaken time.Duration) {
	if len(data) == 0 {
		return
	}

	if !s.config.CanSample() {
		return
	}

	hasBatchSize := bytes.Contains(data, []byte("batch_size"))
	begin := bytes.IndexByte(data, '{')
	end := bytes.LastIndexByte(data, '}')
	if end == -1 {
		return
	}

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
		log.Printf("[%s log] failed to log: %v\n", s.modelID, err)
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

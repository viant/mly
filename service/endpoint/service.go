package endpoint

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/endpoint/checker"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/datastore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

//Service represents http bridge
type Service struct {
	metrics  *gmetric.Service
	config   *Config
	server   *http.Server
	fs       afs.Service
	handlers map[string]*service.Handler
	health   *healthHandler
}

// Metric returns operation metrics
func (s *Service) Metric() *gmetric.Service {
	return s.metrics
}

func (s *Service) mustEmbedUnimplementedEvaluatorServer() {
	panic("implement me")
}

// Deprecated
func (s *Service) ListenAndServe() error {
	ln, err := s.Listen()
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

func (s *Service) Listen() (net.Listener, error) {
	return net.Listen("tcp", s.server.Addr)
}

func (s *Service) Serve(l net.Listener) error {
	log.Printf("starting mly service endpoint: %v\n", s.config.Endpoint.Port)
	return s.server.Serve(l)
}

//ListenAndServeTLS start https endpoint on secure port
func (s *Service) ListenAndServeTLS(certFile, keyFile string) error {
	log.Printf("starting mly service endpoint: %v\n", s.config.Endpoint.Port)
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

//Shutdown stops server
func (s *Service) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Service) initModelHandler(datastores map[string]*datastore.Service, pool *buffer.Pool, mux *http.ServeMux) error {
	var err error
	waitGroup := sync.WaitGroup{}
	numModels := len(s.config.ModelList.Models)
	waitGroup.Add(numModels)
	var lock sync.Mutex
	log.Printf("init %d models...\n", numModels)
	start := time.Now()

	for i := range s.config.ModelList.Models {
		go func(model *config.Model) {
			defer waitGroup.Done()
			log.Printf("model %s init...\n", model.ID)

			if e := s.initModel(datastores, pool, mux, model, &lock); e != nil {
				log.Printf("model %s error:%s\n", model.ID, e)
				lock.Lock()
				err = e
				lock.Unlock()
			}

			log.Printf("model %s done\n", model.ID)
		}(s.config.ModelList.Models[i])
	}
	waitGroup.Wait()

	log.Printf("all model services ready, %s", time.Since(start))
	return err
}

func (s *Service) initModel(datastores map[string]*datastore.Service, pool *buffer.Pool, mux *http.ServeMux, model *config.Model, lock *sync.Mutex) error {
	srv, err := service.New(context.Background(), s.fs, model, s.metrics, datastores)
	if err != nil {
		return fmt.Errorf("failed to create service for model:%v, err:%w", model.ID, err)
	}
	handler := service.NewHandler(srv, pool, s.config.Endpoint.WriteTimeout-time.Millisecond)

	lock.Lock()
	defer lock.Unlock()

	s.health.RegisterHealthPoint(model.ID, &srv.ReloadOK)

	s.handlers[model.ID] = handler
	mux.Handle(fmt.Sprintf(common.ModelURI, model.ID), handler)

	metaHandler := newMetaHandler(srv, &s.config.DatastoreList)
	mux.Handle(fmt.Sprintf(common.MetaURI, model.ID), metaHandler)

	return nil
}

func (s *Service) SelfTest() error {
	waitGroup := sync.WaitGroup{}
	numModels := len(s.config.ModelList.Models)
	waitGroup.Add(numModels)

	host := [1]*client.Host{&client.Host{
		Name: "localhost",
		Port: s.config.Endpoint.Port,
	}}

	hosts := host[:]

	timeout := time.Duration(s.config.Endpoint.ReadTimeoutMs) * time.Millisecond

	errs := make([]error, 0)
	errHandler := make(chan error, 2)

	go func() {
		for {
			e := <-errHandler

			if e == nil {
				return
			}

			errs = append(errs, e)
		}
	}()

	for _, m := range s.config.ModelList.Models {
		go func(modelID string, inputs []*shared.Field, tp config.TestPayload, outputs []*shared.Field, debug bool) {
			defer waitGroup.Done()
			// for backwards compatibility, skip tests if not specified
			if !tp.Test && !tp.SingleBatch && len(tp.Single) == 0 && len(tp.Batch) == 0 {
				log.Printf("!!! skip test %s !!!", modelID)
				return
			}

			start := time.Now()
			err := selfTest(hosts, timeout, modelID, inputs, tp, outputs, debug)

			if err != nil {
				errHandler <- err
				return
			}

			log.Printf("tested %s %s", modelID, time.Now().Sub(start))
		}(m.ID, m.Inputs, m.Test, m.Outputs, m.Debug)
	}

	waitGroup.Wait()

	// should stop goroutine
	errHandler <- nil
	defer close(errHandler)

	var err error
	if len(errs) > 0 {
		errstrs := make([]string, len(errs))
		for i, ierr := range errs {
			errstrs[i] = ierr.Error()
		}

		errstr := strings.Join(errstrs, ";")
		err = fmt.Errorf("%s", errstr)
	}

	return err
}

func selfTest(host []*client.Host, timeout time.Duration, modelID string, inputs_ []*shared.Field, tp config.TestPayload, outputs []*shared.Field, debug bool) error {
	cli, err := client.New(modelID, host)
	if err != nil {
		return fmt.Errorf("%s:%w", modelID, err)
	}

	inputs := cli.Config.Datastore.MetaInput.Inputs

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
		} else {
			testData = make(map[string]interface{})
			for _, field := range inputs {
				n := field.Name
				switch field.DataType {
				case "int", "int32", "int64":
					testData[n] = 0
				case "float", "float32", "float64":
					testData[n] = 0.0
				default:
					testData[n] = ""
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp := new(client.Response)
	// see if there is a transform
	// if there is, trigger the transform with mock data?
	resp.Data = checker.Generated(outputs, batchSize)()

	err = cli.Run(ctx, msg, resp)
	if err != nil {
		return fmt.Errorf("%s:Run():%w", modelID, err)
	}

	return nil
}

//ShutdownOnInterrupt
func (s *Service) shutdownOnInterrupt() {
	closed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		// We received an interrupt signal, shut down.
		if err := s.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP Service Shutdown: %v", err)
		}
		close(closed)
	}()
}

//New creates a bridge Service
func New(config *Config) (*Service, error) {
	mux := http.NewServeMux()

	mux.Handle(configURI, NewConfigHandler(config))

	if config.EnableMemProf {
		log.Print("!!! enabling memory profiling endpoint !!!")
		mux.Handle(memProfURI, NewProfHandler(config))
	}

	healthHandler := NewHealthHandler(config)
	mux.Handle(healthURI, healthHandler)

	metrics := gmetric.New()
	metricHandler := gmetric.NewHandler(common.MetricURI, metrics)
	mux.Handle(common.MetricURI, metricHandler)

	result := &Service{
		config:   config,
		metrics:  metrics,
		fs:       afs.New(),
		handlers: make(map[string]*service.Handler),
		health:   healthHandler,
	}

	datastores, err := datastore.NewStoresV2(&config.DatastoreList, metrics, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastores: %w", err)
	}

	// TODO remove
	for _, ds := range datastores {
		ds.ServerDeprecatedFuncAnnouncement()
	}

	pool := buffer.New(config.Endpoint.PoolMaxSize, config.Endpoint.BufferSize)
	err = result.initModelHandler(datastores, pool, mux)
	if err != nil {
		return nil, err
	}

	result.server = &http.Server{
		Addr:           ":" + strconv.Itoa(config.Endpoint.Port),
		Handler:        h2c.NewHandler(mux, &http2.Server{}),
		ReadTimeout:    time.Millisecond * time.Duration(config.Endpoint.ReadTimeoutMs),
		WriteTimeout:   time.Millisecond * time.Duration(config.Endpoint.WriteTimeoutMs),
		MaxHeaderBytes: config.Endpoint.MaxHeaderBytes,
	}

	result.shutdownOnInterrupt()
	return result, nil
}

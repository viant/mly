package endpoint

import (
	"context"
	"fmt"
	"sync"

	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/service/config"
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

//Metric returns operation metrics
func (s *Service) Metric() *gmetric.Service {
	return s.metrics
}

func (s *Service) mustEmbedUnimplementedEvaluatorServer() {
	panic("implement me")
}

//ListenAndServe start http endpoint
func (s *Service) ListenAndServe() error {
	log.Printf("starting mly service endpoint: %v\n", s.config.Endpoint.Port)
	return s.server.ListenAndServe()
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

	datastores, err := datastore.NewStores(&config.DatastoreList, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastores: %w", err)
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

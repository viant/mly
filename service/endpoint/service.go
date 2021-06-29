package endpoint

import (
	"context"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/common"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/buffer"
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
	metrics *gmetric.Service
	config  *Config
	server  *http.Server
	fs      afs.Service
}

//Metric returns operation metrics
func (s *Service) Metric() *gmetric.Service {
	return s.metrics
}

//ListenAndServe start http endpoint
func (s *Service) ListenAndServe() error {
	fmt.Printf("starting mly service endpoint: %v\n", s.config.Endpoint.Port)
	return s.server.ListenAndServe()
}

//ListenAndServeTLS start https endpoint on secure port
func (s *Service) ListenAndServeTLS(certFile, keyFile string) error {
	fmt.Printf("starting mly service endpoint: %v\n", s.config.Endpoint.Port)
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

//Shutdown stops server
func (s *Service) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}



func (s *Service) initModelHandler(datastores map[string]*datastore.Service, pool *buffer.Pool, mux *http.ServeMux) error {
	for i, model := range s.config.ModelList.Models {
		srv, err := service.New(context.TODO(), s.fs, s.config.ModelList.Models[i], s.metrics, datastores)
		if err != nil {
			return fmt.Errorf("failed to create service for modelService: %v, due to %w", model.ID, err)
		}
		handler := service.NewHandler(srv, pool, s.config.Endpoint.WriteTimeout-time.Millisecond)
		mux.Handle(fmt.Sprintf(common.ModelURI, model.ID), handler)
		metaHandler := newMetaHandler(srv, &s.config.DatastoreList)
		mux.Handle(fmt.Sprintf(common.MetaURI, model.ID), metaHandler)
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
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP Service Shutdown: %v", err)
		}
		close(closed)
	}()
}



//New creates a bridge Service
func New(config *Config) (*Service, error) {
	metrics := gmetric.New()
	mux := http.NewServeMux()
	mux.Handle(configURI, NewConfigHandler(config))
	metricHandler := gmetric.NewHandler(common.MetricURI, metrics)
	mux.Handle(common.MetricURI, metricHandler)

	result := &Service{config: config, metrics: metrics, fs: afs.New()}
	pool := buffer.New(config.Endpoint.PoolMaxSize, config.Endpoint.BufferSize)

	datastores, err := datastore.NewStores(&config.DatastoreList, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastores: %w", err)
	}

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

package endpoint

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/viant/gmetric"
	srvConfig "github.com/viant/mly/service/config"
	"github.com/viant/mly/service/endpoint/checker"
	"github.com/viant/mly/service/endpoint/health"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/datastore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"time"
)

const healthURI = "/v1/api/health"

//Service represents http bridge
type Service struct {
	server *http.Server
	config *Config
}

// Deprecated - use Listen and Serve separately
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
	log.Printf("starting mly service endpoint: %v\n", s.server.Addr)
	return s.server.Serve(l)
}

// ListenAndServeTLS start https endpoint on secure port
func (s *Service) ListenAndServeTLS(certFile, keyFile string) error {
	log.Printf("starting mly service endpoint: %v\n", s.server.Addr)
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown stops http.Server
func (s *Service) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Runs a client side call for each model once.
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
		go func(modelID string, transformer string, inputs []*shared.Field, tp srvConfig.TestPayload, outputs []*shared.Field, debug bool) {
			defer waitGroup.Done()
			// for backwards compatibility, skip tests if not specified
			if !tp.Test && !tp.SingleBatch && len(tp.Single) == 0 && len(tp.Batch) == 0 {
				log.Printf("!!! skip test %s !!!", modelID)
				return
			}

			start := time.Now()
			err := checker.SelfTest(hosts, timeout, modelID, transformer != "", inputs, tp, outputs, debug)
			if err != nil {
				errHandler <- err
				return
			}

			log.Printf("tested %s %s", modelID, time.Now().Sub(start))
		}(m.ID, m.Transformer, m.Inputs, m.Test, m.Outputs, m.Debug)
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

// Registers Shutdown() on interrupt.
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

func New(cfg *Config) (*Service, error) {
	mux := http.NewServeMux()

	am := NewAuthMux(mux, cfg)

	am.Handle(configURI, NewConfigHandler(cfg))

	if cfg.EnableMemProf {
		log.Print("!!! enabling memory profiling endpoint !!!")
		am.Handle(memProfURI, NewProfHandler())
	}

	if cfg.EnableCPUProf {
		log.Print("!!! enabling cpu profiling endpoints !!!")
		am.HandleFunc(cpuProfIndexURI, pprof.Index)
		am.HandleFunc(cpuProfCmdlineURI, pprof.Cmdline)
		am.HandleFunc(cpuProfProfileURI, pprof.Profile)
		am.HandleFunc(cpuProfSymbolURI, pprof.Symbol)
		am.HandleFunc(cpuProfTraceURI, pprof.Trace)
	}

	healthHandler := health.NewHealthHandler()
	am.Handle(healthURI, healthHandler)

	metrics := gmetric.New()
	metricHandler := gmetric.NewHandler(common.MetricURI, metrics)
	mux.Handle(common.MetricURI, metricHandler)

	datastores, err := datastore.NewStoresV2(&cfg.DatastoreList, metrics, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastores: %w", err)
	}

	// TODO remove
	for _, ds := range datastores {
		ds.ServerDeprecatedFuncAnnouncement()
	}

	err = Build(mux, cfg, datastores, healthHandler, metrics)
	if err != nil {
		return nil, err
	}

	result := &Service{
		config: cfg,
		server: &http.Server{
			Addr:           ":" + strconv.Itoa(cfg.Endpoint.Port),
			Handler:        h2c.NewHandler(mux, &http2.Server{}),
			ReadTimeout:    time.Millisecond * time.Duration(cfg.Endpoint.ReadTimeoutMs),
			WriteTimeout:   time.Millisecond * time.Duration(cfg.Endpoint.WriteTimeoutMs),
			MaxHeaderBytes: cfg.Endpoint.MaxHeaderBytes,
		},
	}

	result.shutdownOnInterrupt()

	return result, nil
}

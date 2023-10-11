package endpoint

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/service/config"
	serviceConfig "github.com/viant/mly/service/config"
	"github.com/viant/mly/service/endpoint/meta"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/datastore"
	"golang.org/x/sync/semaphore"
)

const (
	us  float64 = 1000
	ms          = 1000000
	sec         = 1000000000
)

var defIdleBuckets []float64 = []float64{
	// rarer and hard-to-measure elapsed
	10, 100,

	// most models and differences occur here
	1 * us, 2500, 5 * us, 7500,
	10 * us, 25 * us, 50 * us, 75 * us,
	100 * us, 250 * us, 500 * us, 750 * us,
	1 * ms, 2.5 * ms, 5 * ms, 7.5 * ms,

	// these are effectively larger than model latency
	10 * ms, 50 * ms, 100 * ms, 500 * ms,
	1 * sec,
}

// TODO Refactor out
type Hook interface {
	Hook(*config.Model, *service.Service)
}

func Build(mux *http.ServeMux, config *Config, datastores map[string]*datastore.Service,
	hooks []Hook, metrics *gmetric.Service, promReg *prometheus.Registry) error {

	cfge := config.Endpoint
	pool := buffer.New(cfge.PoolMaxSize, cfge.BufferSize)

	fs := afs.New()
	handlerTimeout := cfge.WriteTimeout - time.Millisecond

	sema := semaphore.NewWeighted(cfge.MaxEvaluatorConcurrency)

	waitGroup := sync.WaitGroup{}
	numModels := len(config.ModelList.Models)
	waitGroup.Add(numModels)

	buckets := defIdleBuckets
	if mc := config.Metrics; mc != nil {
		if pms := mc.Prometheus; pms != nil {
			if mib := pms.ModelIdletimeBuckets; mib != nil && len(mib) > 0 {
				buckets = mib
			}
		}
	}

	// records in time.Duration (nanosecond count)
	obsv := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "mly",
		Subsystem: "model",
		Name:      "idletime",

		Help: "measured time between requests",

		Buckets: buckets,
	}, []string{"model"})

	var err error
	err = promReg.Register(obsv)
	if err != nil {
		return err
	}

	log.Printf("init %d models...\n", numModels)

	var lock sync.Mutex
	start := time.Now()
	for _, m := range config.ModelList.Models {
		go func(model *serviceConfig.Model) {
			defer waitGroup.Done()

			mstart := time.Now()

			log.Printf("[%s] model loading", model.ID)
			e := func() error {
				tfService := tfmodel.NewService(model, fs, metrics, sema, cfge.MaxEvaluatorWait)
				modelSrv, err := service.New(context.Background(), model, tfService, fs, metrics, datastores)
				if err != nil {
					return fmt.Errorf("failed to create service for model:%v, err:%w", model.ID, err)
				}

				handler := service.NewHandler(modelSrv, pool, handlerTimeout, metrics, obsv)

				lock.Lock()
				defer lock.Unlock()

				for _, hook := range hooks {
					hook.Hook(model, modelSrv)
				}

				mux.Handle(fmt.Sprintf(common.ModelURI, model.ID), handler)

				metaHandler := meta.NewMetaHandler(modelSrv, &config.DatastoreList, metrics)
				mux.Handle(fmt.Sprintf(common.MetaURI, model.ID), metaHandler)

				return nil
			}()

			if e != nil {
				log.Printf("[%s init] error:%s", model.ID, e)

				lock.Lock()
				err = e
				lock.Unlock()
			}

			log.Printf("[%s] model loaded (%s)", model.ID, time.Now().Sub(mstart))
		}(m)
	}

	waitGroup.Wait()

	log.Printf("all model services loaded in %s", time.Since(start))
	return err
}

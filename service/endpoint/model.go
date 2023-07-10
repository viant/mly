package endpoint

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/buffer"
	serviceConfig "github.com/viant/mly/service/config"
	"github.com/viant/mly/service/endpoint/health"
	"github.com/viant/mly/service/endpoint/meta"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/datastore"
	"github.com/viant/mly/shared/semaph"
)

func Build(mux *http.ServeMux, config *Config, datastores map[string]*datastore.Service, healthHandler *health.HealthHandler, metrics *gmetric.Service) error {
	pool := buffer.New(config.Endpoint.PoolMaxSize, config.Endpoint.BufferSize)
	fs := afs.New()
	handlerTimeout := config.Endpoint.WriteTimeout - time.Millisecond

	sema := semaph.NewSemaph(config.Endpoint.MaxEvaluatorConcurrency)

	waitGroup := sync.WaitGroup{}
	numModels := len(config.ModelList.Models)
	waitGroup.Add(numModels)

	log.Printf("init %d models...\n", numModels)

	var err error
	var lock sync.Mutex
	start := time.Now()
	for _, m := range config.ModelList.Models {
		go func(model *serviceConfig.Model) {
			defer waitGroup.Done()

			mstart := time.Now()

			log.Printf("[%s] model loading", model.ID)
			e := func() error {
				modelSrv, err := service.New(context.Background(), fs, model, metrics, sema, datastores)
				if err != nil {
					return fmt.Errorf("failed to create service for model:%v, err:%w", model.ID, err)
				}

				handler := service.NewHandler(modelSrv, pool, handlerTimeout)

				lock.Lock()
				defer lock.Unlock()

				healthHandler.RegisterHealthPoint(model.ID, &modelSrv.ReloadOK)

				mux.Handle(fmt.Sprintf(common.ModelURI, model.ID), handler)

				metaHandler := meta.NewMetaHandler(modelSrv, &config.DatastoreList)
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

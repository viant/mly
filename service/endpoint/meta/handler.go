package meta

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/service"
	"github.com/viant/mly/shared/client/config"
	sconfig "github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/config/datastore"
	"github.com/viant/mly/shared/stat"
)

type metaHandler struct {
	datastore    *config.Remote
	modelService *service.Service

	handlerMetrics *gmetric.Operation
	dictMetrics    *gmetric.Operation
	cfgMetrics     *gmetric.Operation
}

func (h *metaHandler) handleConfigRequest(writer http.ResponseWriter) error {
	data, err := json.Marshal(h.datastore)
	if err != nil {
		return err
	}
	writer.Header().Set("Content-type", "application/json")
	_, err = io.Copy(writer, bytes.NewReader(data))
	return err
}

func (h *metaHandler) handleDictionaryRequest(writer http.ResponseWriter) error {
	dictionary := h.modelService.Dictionary()
	data, err := json.Marshal(dictionary)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	gzWriter := gzip.NewWriter(buf)
	_, err = io.Copy(gzWriter, bytes.NewReader(data))
	if err == nil {
		if err = gzWriter.Flush(); err == nil {
			err = gzWriter.Close()
		}
	}
	if err != nil {
		return err
	}
	writer.Header().Set("Content-Encoding", "gzip")
	writer.Header().Set("Content-type", "application/json")
	_, err = io.Copy(writer, buf)
	return err
}

func (h *metaHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()

	handlerDone := h.handlerMetrics.Begin(startTime)
	values := stat.NewValues()
	defer func() {
		handlerDone(time.Now(), values.Values()...)
	}()

	var err error
	if strings.HasSuffix(request.RequestURI, "/dictionary") {
		specDone := h.dictMetrics.Begin(startTime)
		specValues := stat.NewValues()
		defer func() {
			specDone(time.Now(), specValues.Values()...)
		}()
		err = h.handleDictionaryRequest(writer)
		specValues.Append(err)
		values.Append(Dictionary)
	} else if strings.HasSuffix(request.RequestURI, "/config") {
		specDone := h.cfgMetrics.Begin(startTime)
		specValues := stat.NewValues()
		defer func() {
			specDone(time.Now(), specValues.Values()...)
		}()
		err = h.handleConfigRequest(writer)
		specValues.Append(err)
		values.Append(Config)
	} else {
		values.Append(NotFound)
		http.NotFound(writer, request)
	}

	if err != nil {
		values.Append(err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func NewMetaHandler(srv *service.Service, datastoreList *sconfig.DatastoreList, gmetrics *gmetric.Service) *metaHandler {
	cfg := srv.Config()
	location := reflect.TypeOf(metaHandler{}).PkgPath()
	handler := &metaHandler{
		modelService: srv,
		datastore:    assembleConfig(datastoreList, cfg.DataStore),

		handlerMetrics: gmetrics.MultiOperationCounter(location, cfg.ID+"MetaHandler", cfg.ID+" meta handler performance", time.Microsecond, time.Minute, 1, NewServiceVP()),
		dictMetrics:    gmetrics.MultiOperationCounter(location, cfg.ID+"DictMeta", cfg.ID+" dictionary service performance", time.Microsecond, time.Minute, 1, NewProvider()),
		cfgMetrics:     gmetrics.MultiOperationCounter(location, cfg.ID+"CfgMeta", cfg.ID+" configuration service performance", time.Microsecond, time.Minute, 1, NewProvider()),
	}

	handler.datastore.MetaInput = cfg.MetaInput
	return handler
}

func assembleConfig(datastoreList *sconfig.DatastoreList, name string) *config.Remote {
	connections := map[string]*datastore.Connection{}
	datastores := map[string]*sconfig.Datastore{}
	if len(datastoreList.Connections) > 0 {
		for i, item := range datastoreList.Connections {
			connections[item.ID] = datastoreList.Connections[i]
		}
	}
	if len(datastoreList.Datastores) > 0 {
		for i, item := range datastoreList.Datastores {
			datastores[item.ID] = datastoreList.Datastores[i]
		}
	}
	result := &config.Remote{}
	ds, ok := datastores[name]
	if ok {
		result.Datastore = *ds
		if ds.Reference != nil {
			if conn, ok := connections[ds.Reference.Connection]; ok {
				result.Connections = append(result.Connections, conn)
			}
			if ds.L2 != nil {
				if conn, ok := connections[ds.L2.Connection]; ok {
					result.Connections = append(result.Connections, conn)
				}
			}
		}
	}
	return result
}

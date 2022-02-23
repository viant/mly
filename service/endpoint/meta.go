package endpoint

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/viant/mly/service"
	"github.com/viant/mly/shared/client/config"
	sconfig "github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/config/datastore"
	"io"
	"net/http"
	"strings"
)

type metaHandler struct {
	datastore    *config.Datastore
	modelService *service.Service
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
	var err error
	if strings.HasSuffix(request.RequestURI, "/dictionary") {
		err = h.handleDictionaryRequest(writer)
	} else if strings.HasSuffix(request.RequestURI, "/config") {
		err = h.handleConfigRequest(writer)
	} else {
		http.NotFound(writer, request)
	}
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func newMetaHandler(srv *service.Service, datastoreList *sconfig.DatastoreList) *metaHandler {
	modelConfig := srv.Config()
	handler := &metaHandler{
		modelService: srv,
		datastore:    assembleConfig(datastoreList, modelConfig.DataStore),
	}
	handler.datastore.KeyFields = modelConfig.KeyFields
	handler.datastore.WildcardKeys = modelConfig.WildcardFields
	return handler

}

func assembleConfig(datastoreList *sconfig.DatastoreList, name string) *config.Datastore {
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
	result := &config.Datastore{}
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

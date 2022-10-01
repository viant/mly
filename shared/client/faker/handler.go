package faker

import (
	"context"
	"fmt"
	"github.com/viant/afs"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

const (
	beginFragment = "/v1/api/model/"
	endFragment   = "/eval"
)

type Handler struct {
	baseURL string
	debug   bool
	fs      afs.Service
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	URI := r.RequestURI
	var data []byte
	var err error
	if r.Body != nil {
		data, err = ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	machURL := h.matchedURL(URI)
	output, err := h.fs.DownloadWithURL(context.TODO(), machURL)
	if h.debug {
		fmt.Printf("matched URL: %v %s\n", machURL, data)
		fmt.Printf("output: %s\n", output)
	}
	if err == nil {
		w.Header().Set("Content-Type", "applicaiton/json")
		w.Write(output)
		return
	}

	w.WriteHeader(http.StatusOK)

}

func (h *Handler) matchedURL(URI string) string {
	match := ""
	if index := strings.Index(URI, beginFragment); index != -1 {
		match = URI[index+len(beginFragment):]
	}
	if index := strings.Index(match, endFragment); index != -1 {
		match = match[:index]
	}
	return path.Join(h.baseURL, match, "response.json")
}

package faker

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/viant/afs"
)

const (
	beginFragment = "/v1/api/model/"
	endFragment   = "/eval"
)

type Handler struct {
	baseURL string
	debug   bool
	fs      afs.Service

	next func([]byte, http.ResponseWriter)
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

	testDataURL := h.fetchTestdata(URI)
	output, err := h.fs.DownloadWithURL(context.TODO(), testDataURL)

	if h.debug {
		fmt.Printf("matched URL: %v %s\n", testDataURL, data)
		fmt.Printf("output: %s\n", output)
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (h *Handler) Then(next func([]byte, http.ResponseWriter)) bool {
	old := h.next
	h.next = next
	return old != nil
}

func (h *Handler) fetchTestdata(URI string) string {
	match := ""
	if index := strings.Index(URI, beginFragment); index != -1 {
		match = URI[index+len(beginFragment):]
	}
	if index := strings.Index(match, endFragment); index != -1 {
		match = match[:index]
	}
	return path.Join(h.baseURL, match, "response.json")
}

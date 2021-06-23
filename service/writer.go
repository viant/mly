package service

import (
	"io"
	"net/http"
)

type flushWriter struct {
	w io.Writer
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	// Flush - send the buffered written data to the client
	if f, ok := fw.w.(http.Flusher); ok {
		f.Flush()
	}
	return
}

func echoCapitalHandler(w http.ResponseWriter, r *http.Request) {
	// First flash response headers
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	// Copy from the request body to the response writer and flush
	// (send to client)
	io.Copy(flushWriter{w: w}, r.Body)
}

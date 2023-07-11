package smasher

import (
	"bufio"
	"net"
	"net/http"
)

type SocketPass struct {
	sockets chan *http.Response
	w       *bufio.Writer
	conn    net.Conn
}

func (sp *SocketPass) Write(p []byte) (n int, err error) {
	n, err = sp.conn.Write(p)
	// TODO net/http seems to collect number of bytes written for determining if error in the future
	return
}

// implement net/http.RoundTripper
func (sp *SocketPass) RoundTrip(q *http.Request) (*http.Response, error) {
	err := q.Write(sp.w)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func New(t *http.Transport) *SocketPass {
	sp := new(SocketPass)

	// kind of like net/http.(*Transport).writeBufferSize()
	var bsize int
	if t.WriteBufferSize > 0 {
		bsize = t.WriteBufferSize
	} else {
		bsize = 4 << 10
	}

	sp.w = bufio.NewWriterSize(sp, bsize)
	return sp
}

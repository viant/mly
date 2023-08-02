package tracker

import (
	"io"
)

type Tracker interface {
	// Adds an element to track
	AddBytes([]byte)

	// Shows top K elements, unsorted
	TopK() []Item

	io.Closer
}

type Item struct {
	Data  []byte
	Count uint64
}

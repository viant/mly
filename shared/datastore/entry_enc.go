package datastore

import (
	"github.com/viant/bintly"
)

//EncodeBinary encodes data from binary stream
func (e *Entry) EncodeBinary(stream *bintly.Writer) error {
	stream.MString(e.Key)
	stream.Bool(e.NotFound)
	stream.Time(e.Expiry)
	stream.Int(e.Hash)
	if e.Data == nil {
		stream.MAlloc(0)
		return nil
	}
	stream.MAlloc(1)
	if coder, ok := e.Data.(bintly.Encoder); ok {
		return stream.Coder(coder)
	}
	return stream.Any(e.Data)
}

//DecodeBinary decodes data to binary stream
func (e *Entry) DecodeBinary(stream *bintly.Reader) error {
	stream.MString(&e.Key)
	stream.Bool(&e.NotFound)
	stream.Time(&e.Expiry)
	stream.Int(&e.Hash)
	size := stream.MAlloc()
	if size == 0 {
		return nil
	}
	if coder, ok := e.Data.(bintly.Decoder); ok {
		return stream.Coder(coder)
	}
	return stream.Any(e.Data)
}

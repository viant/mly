package datastore


import (
	"github.com/francoispqt/gojay"
	"github.com/viant/bintly"
)

//EncodeBinary encodes data from binary stream
func (e *Entry) EncodeBinary(stream *bintly.Writer) error {
	stream.MString(e.Key)
	stream.Bool(e.NotFound)
	stream.Time(e.Expiry)
	if e.Data == nil {
		stream.MAlloc(0)
		return nil
	}
	stream.MAlloc(1)
	if bintlyCoder, ok := e.Data.(bintly.Encoder); ok {
		return stream.Coder(bintlyCoder)
	}
	if gojayCoder, ok := e.Data.(gojay.MarshalerJSONObject); ok {
		data, err := gojay.Marshal(gojayCoder)
		if err != nil {
			return err
		}
		stream.Uint8s(data)
	}
	return nil
}

//DecodeBinary decodes data to binary stream
func (e *Entry) DecodeBinary(stream *bintly.Reader) error {
	stream.MString(&e.Key)
	stream.Bool(&e.NotFound)
	stream.Time(&e.Expiry)
	size := stream.MAlloc()
	if size == 0 {
		return nil
	}
	if bintlyCoder, ok := e.Data.(bintly.Decoder); ok {
		return stream.Coder(bintlyCoder)
	}
	if gojayCoder, ok := e.Data.(gojay.UnmarshalerJSONObject); ok {
		var data []byte
		stream.Uint8s(&data)
		return gojay.Unmarshal(data, gojayCoder)
	}
	return nil
}


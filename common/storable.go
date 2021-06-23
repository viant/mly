package common

import "github.com/francoispqt/gojay"

type Storable interface {
	Iterator() Iterator
	Set(iter Iterator) error
	MarshalJSONObject(enc *gojay.Encoder)
	IsNil() bool
	SetHash(hash int)
	Hash() int
}

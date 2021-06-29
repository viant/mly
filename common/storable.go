package common

type Storable interface {
	Iterator() Iterator
	Set(iter Iterator) error
}

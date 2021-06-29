package common

//Storable represents storable interface
type Storable interface {
	Iterator() Iterator
	Set(iter Iterator) error
}

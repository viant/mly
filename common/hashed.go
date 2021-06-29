package common

const (
	//HashBin represents hash bin
	HashBin = "dictHash"
)

//Hashed represent dictionary hash holder
type Hashed interface {
	Hash() int
	SetHash(hash int)
}

//SetHash sets hash
func SetHash(dest interface{}, hash int) {
	if hasher, ok := dest.(Hashed); ok {
		hasher.SetHash(hash)
	}
}

//Hash returns hash or zero
func Hash(source interface{}) int {
	if hasher, ok := source.(Hashed); ok {
		return hasher.Hash()
	}
	return 0
}

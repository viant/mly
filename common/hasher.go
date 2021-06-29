package common

const (
	HashBin = "dictHash"
)

type Hasher interface {
	Hash() int
	SetHash(hash int)
}

//SetHash sets hash
func SetHash(dest interface{}, hash int) {
	if hasher, ok := dest.(Hasher); ok {
		hasher.SetHash(hash)
	}
}

//Hash returns hash or zero
func Hash(source interface{}) int {
	if hasher, ok := source.(Hasher); ok {
		return hasher.Hash()
	}
	return 0
}

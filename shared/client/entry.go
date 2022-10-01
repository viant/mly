package client

type entry struct {
	ints     map[int]bool
	float32s map[float32]bool
	strings  map[string]bool
}

func (e *entry) hasString(val string) bool {
	if len(e.strings) == 0 {
		return false
	}
	_, ok := e.strings[val]
	return ok
}

func (e *entry) hasInt(val int) bool {
	if len(e.ints) == 0 {
		return false
	}
	_, ok := e.ints[val]
	return ok
}

func (e *entry) hasFloat32(val float32) bool {
	if len(e.float32s) == 0 {
		return false
	}
	_, ok := e.float32s[val]
	return ok
}

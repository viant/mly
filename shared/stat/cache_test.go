package stat

import "testing"

func TestCache(t *testing.T) {
	TestMapping(t, NewCache)
}

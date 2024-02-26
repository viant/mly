package stat

import "testing"

func TestStore(t *testing.T) {
	TestMapping(t, NewClient)
}

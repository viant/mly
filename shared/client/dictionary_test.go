package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNil(t *testing.T) {
	var d *Dictionary
	assert.Nil(t, d)
	d.reduceFloat("test", 1.0)
	d.lookupString("test", "test")
}

package buffer

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	useCases := []struct {
		description string
		bufferSize  int
		reader      io.Reader
		hasError    bool
		expect      string
	}{
		{
			description: "medium buffer, small data",
			reader:      strings.NewReader("Lorem ipsum"),
			bufferSize:  1024,
			expect:      "Lorem ipsum",
		},
		{
			description: "large buffer, medium data",
			reader:      strings.NewReader(strings.Repeat("Lorem ipsum", 1024)),
			bufferSize:  1024 * 1024,
			expect:      strings.Repeat("Lorem ipsum", 1024),
		},
		{
			description: "large buffer, large data",
			reader:      strings.NewReader(strings.Repeat("Lorem ipsum", 1024*1024)),
			bufferSize:  1024 * 1024 * 26,
			expect:      strings.Repeat("Lorem ipsum", 1024*1024),
		},
		{
			description: "too small buffer",
			reader:      strings.NewReader(strings.Repeat("Lorem ipsum", 1024*1024)),
			bufferSize:  1024,
			hasError:    true,
			expect:      strings.Repeat("Lorem ipsum", 1024*1024),
		},
	}

	for _, useCase := range useCases {
		pool := New(10, useCase.bufferSize)
		data, size, err := Read(pool, useCase.reader)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		if !assert.Equal(t, size, len(useCase.expect), useCase.description) {
			continue
		}
		assert.Equal(t, string(data[:size]), useCase.expect, useCase.description)
		pool.Put(data)

	}
}

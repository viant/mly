package stat

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

// TestHandler_Mapping verifies the routing invariants shared by all providers:
// every non-error key maps to a dedicated (non -1) bucket, and a generic error
// maps to bucket 0 (ErrorKey).
func TestHandler_Mapping(t *testing.T) {
	stat.TestMapping(t, NewHandler)
}

// TestHandler_CustomProvider guards the contract whose violation caused a
// production nil-pointer panic: ResponseMarshalError and ResponseCommittedError
// satisfy counter.CustomCounter, so the provider that emits them must be a
// counter.CustomProvider. The compile-time guards in handler.go encode this;
// this test documents and exercises the runtime expectation.
func TestHandler_CustomProvider(t *testing.T) {
	p := NewHandler()

	cp, ok := p.(counter.CustomProvider)
	assert.True(t, ok, "NewHandler() must implement counter.CustomProvider")
	if !ok {
		return
	}
	assert.NotNil(t, cp.NewCounter(), "NewCounter() must return a non-nil CustomCounter")
}

// TestHandler_NoPanicOnCustomCounterFlush reproduces the production panic path:
// a CustomCounter-typed stat marker flushed through a gmetric Operation built
// with NewHandler(). Before NewHandler() implemented CustomProvider,
// counter.NewOperation left Value.Custom nil and
// counter.(*MultiCounter).incrementValueBy nil-dereferenced when invoking
// Aggregate (counter/multi.go).
func TestHandler_NoPanicOnCustomCounterFlush(t *testing.T) {
	op := counter.NewOperation(time.Microsecond, NewHandler())

	markers := []interface{}{
		ResponseCommittedError{Error: errors.New("write response body: connection reset by peer")},
		ResponseMarshalError{Error: errors.New("marshal response: unsupported type")},
	}

	assert.NotPanics(t, func() {
		onDone := op.Begin(time.Now())
		onDone(time.Now(), markers...)
	})

	// Markers must land in their dedicated buckets (Map: 3 and 4), not the
	// generic ErrorKey bucket (index 0).
	assert.EqualValues(t, 1, op.Counters[3].Count, "ResponseMarshalError -> bucket 3")
	assert.EqualValues(t, 1, op.Counters[4].Count, "ResponseCommittedError -> bucket 4")
	assert.EqualValues(t, 0, op.Counters[0].Count, "ErrorKey bucket must stay empty")
}

// TestHandler_NoPanicOnNilErrorMarker guards the marker String() methods,
// which are reachable on the sampling path (TopK.Aggregate). A zero-value
// marker (nil Error) must not nil-dereference when flushed.
func TestHandler_NoPanicOnNilErrorMarker(t *testing.T) {
	op := counter.NewOperation(time.Microsecond, NewHandler())

	markers := []interface{}{
		ResponseCommittedError{},
		ResponseMarshalError{},
	}

	assert.NotPanics(t, func() {
		onDone := op.Begin(time.Now())
		onDone(time.Now(), markers...)
	})

	assert.EqualValues(t, 1, op.Counters[3].Count, "ResponseMarshalError -> bucket 3")
	assert.EqualValues(t, 1, op.Counters[4].Count, "ResponseCommittedError -> bucket 4")
}

// TestHandler_GenericErrorRoutesToErrorBucket verifies a plain error still
// routes to bucket 0 and does not take the CustomCounter Aggregate path
// (errors are not CustomCounters), so this remains safe regardless of the
// Custom allocation.
func TestHandler_GenericErrorRoutesToErrorBucket(t *testing.T) {
	op := counter.NewOperation(time.Microsecond, NewHandler())

	assert.NotPanics(t, func() {
		onDone := op.Begin(time.Now())
		onDone(time.Now(), errors.New(stat.ErrorKey))
	})

	assert.EqualValues(t, 1, op.Counters[0].Count, "generic error -> bucket 0")
}

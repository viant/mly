package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	// ResponseMarshalErrorKey counts requests where the prediction succeeded
	// but gojay.Marshal of the Response struct failed. The HTTP response was
	// NOT committed: ServeHTTP recovers by emitting an explicit 500.
	ResponseMarshalErrorKey = "responseMarshalError"

	// ResponseCommittedErrorKey counts requests where status + headers were
	// already flushed to the client (200 OK) when the body Write failed.
	// This is the server-side counterpart to the bidder-observed
	// "200 OK + empty/truncated body → invalid_json" failure mode.
	// A non-zero rate here indicates either:
	//   - clients are closing the connection mid-response (most common
	//     under load when client deadline < server response time), or
	//   - HTTP/1.1 keepalive desync producing broken pipes on reuse.
	// Distinct from ErrorKey so it can be alerted independently.
	ResponseCommittedErrorKey = "responseCommittedError"
)

// ResponseMarshalError is a stat marker for the gmetric provider. The
// embedded error is retained for top-K error sampling; the struct itself
// is intentionally NOT an `error` so the type-switch in Map can route it
// to its own bucket without colliding with the generic error case.
type ResponseMarshalError struct{ Error error }

// String implements fmt.Stringer (used by gmetric top-K error sampling).
func (r ResponseMarshalError) String() string { return r.Error.Error() }

// Aggregate implements github.com/viant/gmetric/counter.CustomCounter.
func (r ResponseMarshalError) Aggregate(interface{}) {}

// ResponseCommittedError is the analogous stat marker for post-commit
// write failures. See ResponseCommittedErrorKey for the operational
// significance.
type ResponseCommittedError struct{ Error error }

func (r ResponseCommittedError) String() string         { return r.Error.Error() }
func (r ResponseCommittedError) Aggregate(interface{}) {}

// handler is the gmetric counter.Provider for service.Handler.ServeHTTP.
// It is a strict superset of shared/stat.NewCtxErrOnly(): the first three
// keys (ErrorKey, Canceled, DeadlineExceeded) preserve their indices so
// any existing Prometheus dashboards/alerts on those buckets continue to
// emit at the same labels. Two new keys are appended for the explicit
// response-write failure classes introduced by the explicit-commit
// refactor of writeResponse.
type handler struct{}

// Keys returns the stat key labels in stable index order. Order matters:
// gmetric's counter buckets are addressed by index, and changing the
// order of the first three would silently re-label existing series.
func (h handler) Keys() []string {
	return []string{
		stat.ErrorKey,                // 0
		stat.Canceled,                // 1
		stat.DeadlineExceeded,        // 2
		ResponseMarshalErrorKey,      // 3
		ResponseCommittedErrorKey,    // 4
	}
}

// Map routes a value to its key index. Concrete struct cases come BEFORE
// the generic `error` case to ensure typed stat markers route to their
// dedicated buckets even if a future change makes them satisfy `error`.
func (h handler) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	if _, ok := value.(ResponseMarshalError); ok {
		return 3
	}
	if _, ok := value.(ResponseCommittedError); ok {
		return 4
	}

	switch v := value.(type) {
	case error:
		return 0
	case string:
		switch v {
		case stat.Canceled:
			return 1
		case stat.DeadlineExceeded:
			return 2
		case ResponseMarshalErrorKey:
			return 3
		case ResponseCommittedErrorKey:
			return 4
		}
	}

	return -1
}

// NewHandler returns the counter.Provider used by Handler.ServeHTTP's
// per-request httpContextMetrics.
func NewHandler() counter.Provider {
	return handler{}
}

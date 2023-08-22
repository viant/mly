package config

import "time"

// Modified represents modified folder
type Modified struct {
	Min time.Time
	Max time.Time
}

func (r *Modified) Span() time.Duration {
	return r.Max.Sub(r.Min)
}

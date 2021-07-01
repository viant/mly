package config

import "time"

//Modified represents modified folder
type Modified struct {
	Min time.Time
	Max time.Time
}

func (r *Modified) Span() time.Duration {
	return r.Min.Sub(r.Min)
}


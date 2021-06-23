package datastore

import (
	"strings"
	"time"
)

type Timeout struct {
	Unit       string
	Connection int
	Socket     int
	Total    int
}

func (t *Timeout) DurationUnit() time.Duration {
	if t.Unit == "" {
		t.Unit = "ms"
	}
	switch strings.ToLower(t.Unit) {
	case "ms":
		return time.Millisecond
	case "sec":
		return time.Second
	case "min":
		return time.Minute
	default:
		return time.Millisecond
	}
}

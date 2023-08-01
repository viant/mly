package datastore

import "time"

const (
	defaultTimeToLiveMs = 900000 // 15 minutes
	defaultRetryTimeMs  = 5000   // 5 seconds
)

//Reference datastore reference
type Reference struct {
	Connection   string
	Namespace    string
	Dataset      string
	TimeToLiveMs int `json:",omitempty" yaml:",omitempty"`
	timeToLive   time.Duration
	RetryTimeMs  int `json:",omitempty" yaml:",omitempty"`
	retryTime    time.Duration
	ReadOnly     bool `json:",omitempty" yaml:",omitempty"`
}

func (d *Reference) TimeToLive() time.Duration {
	if d.timeToLive > 0 {
		return d.timeToLive
	}
	d.timeToLive = time.Duration(d.TimeToLiveMs) * time.Millisecond
	return d.timeToLive
}

func (d *Reference) RetryTime() time.Duration {
	if d.retryTime > 0 {
		return d.retryTime
	}
	d.retryTime = time.Duration(d.RetryTimeMs) * time.Millisecond
	return d.retryTime
}

func (d *Reference) Init() {
	if d.TimeToLiveMs == 0 {
		d.TimeToLiveMs = defaultTimeToLiveMs
	}
	if d.RetryTimeMs == 0 {
		d.RetryTimeMs = defaultRetryTimeMs
	}
}

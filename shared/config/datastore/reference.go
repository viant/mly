package datastore

import "time"

type Reference struct {
	Connection   string
	Namespace    string
	Dataset      string
	TimeToLiveMs int
	timeToLive   time.Duration
	RetryTimeMs  int
	retryTime    time.Duration
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

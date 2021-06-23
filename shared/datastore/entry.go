package datastore

import "time"

type Entry struct {
	Key      string
	Data     EntryData
	NotFound bool
	Expiry   time.Time
}

//EntryData
type EntryData interface{}


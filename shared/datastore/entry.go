package datastore

import "time"

//Entry represents data entry
type Entry struct {
	Key      string
	Data     EntryData
	Hash     int
	NotFound bool
	Expiry   time.Time
}

//EntryData
type EntryData interface{}

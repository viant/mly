package client

type Pool struct {
	// key is shared/config/datastore/Connection.ID
	clients map[string]*Service
}

var DefaultAerospikePool = &Pool{}

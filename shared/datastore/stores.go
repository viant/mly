package datastore

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore/client"
	"github.com/viant/mly/shared/stat"
)

// Deprecated: use NewStoresV2 or NewStoresV3 instead.
func NewStores(cfg *config.DatastoreList, gmetrics *gmetric.Service) (map[string]*Service, error) {
	return NewStoresV2(cfg, gmetrics, false)
}

// NewStoresV2 verbose enables logging of connections.
// Deprecated: use NewStoresV3.
func NewStoresV2(cfg *config.DatastoreList, gmetrics *gmetric.Service, verbose bool) (map[string]*Service, error) {
	return NewStoresV3(cfg, gmetrics, verbose, nil)
}

// NewStoresV3 creates new stores.
// verbose enables logging of connections.
// connections is a map of connections to be used for datastores.
// Note that connections shares via ID, not by Hostnames.
func NewStoresV3(cfg *config.DatastoreList, gmetrics *gmetric.Service, verbose bool, connections map[string]*client.Service) (map[string]*Service, error) {
	var result = make(map[string]*Service)
	location := reflect.TypeOf(Service{}).PkgPath()

	if connections == nil {
		connections = make(map[string]*client.Service)
	}

	if len(cfg.Connections) > 0 {
		for _, connection := range cfg.Connections {
			connID := connection.ID
			if _, ok := connections[connID]; ok {
				if verbose {
					log.Printf("connection %s already exists", connID)
				}

				continue
			}

			aero, err := client.New(connection)
			if err != nil {
				return nil, fmt.Errorf("failed to create client for %v, due to %w", connID, err)
			}

			connections[connID] = aero

			if verbose {
				log.Printf("connection %s ok", connID)
			}
		}
	}

	for _, db := range cfg.Datastores {
		l1Client, l2Client, err := getClient(db, connections)
		if err != nil {
			return nil, err
		}

		dbID := db.ID

		var rctr, wctr *gmetric.Operation
		if dbID != "" {
			rctr = gmetrics.MultiOperationCounter(location, dbID, dbID+" read cache performance", time.Microsecond, time.Minute, 2, stat.NewCache())
			wctr = gmetrics.MultiOperationCounter(location, dbID+"CacheW", dbID+" write cache performance", time.Microsecond, time.Minute, 2, stat.NewWrite())
		}

		dbService, err := NewWithCache(db, l1Client, l2Client, rctr, wctr)
		if err != nil {
			return nil, fmt.Errorf("failed to create datastore: %v, due to %w", dbID, err)
		}

		result[dbID] = dbService
		if verbose {
			log.Printf("datastore %s l1:%v l2:%v ok", dbID, l1Client, l2Client)
		}
	}

	return result, nil
}

func getClient(db *config.Datastore, connections map[string]*client.Service) (*client.Service, *client.Service, error) {
	if db == nil || db.Reference == nil {
		return nil, nil, nil
	}

	var l1Client, l2Client *client.Service
	if db.Reference.Connection != "" {
		var ok bool
		if l1Client, ok = connections[db.Reference.Connection]; !ok {
			return nil, nil, fmt.Errorf("failed to lookup datastore connection %v, for %v", db.Reference.Connection, db.ID)
		}

		if db.Debug {
			log.Printf("datastore %s l1 connection ok", db.ID)
		}

		if db.L2 != nil {
			if l2Client, ok = connections[db.L2.Connection]; !ok {
				return nil, nil, fmt.Errorf("failed to lookup datastore connection %v, for %v", db.L2.Connection, db.ID)
			}
		}
	}

	return l1Client, l2Client, nil
}

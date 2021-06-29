package datastore

import (
	"fmt"
	"github.com/viant/gmetric"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore/client"
	"github.com/viant/mly/shared/stat"
	"reflect"
	"time"
)

//NewStores creates new stores
func NewStores(cfg *config.DatastoreList, gmetrics *gmetric.Service) (map[string]*Service, error) {
	var result = make(map[string]*Service)
	location := reflect.TypeOf(Service{}).PkgPath()
	var connections = map[string]client.Service{}
	if len(cfg.Connections) > 0 {
		for i, connection := range cfg.Connections {
			aero, err := client.New(cfg.Connections[i])
			if err != nil {
				return nil, fmt.Errorf("failed to create client for %v, due to %w", connection.ID, err)
			}
			connections[connection.ID] = aero
		}
	}
	for i, db := range cfg.Datastores {
		l1Client, l2Client, err := getClient(db, connections)
		if err != nil {
			return nil, err
		}
		var counter *gmetric.Operation
		if db.Cache != nil {
			counter = gmetrics.MultiOperationCounter(location, db.ID, db.ID+" performance", time.Microsecond, time.Minute, 2, stat.NewCache())
		} else {
			counter = gmetrics.MultiOperationCounter(location, db.ID, db.ID+" performance", time.Microsecond, time.Minute, 2, stat.NewStore())
		}
		dbService, err := NewWithCache(cfg.Datastores[i], l1Client, l2Client, counter)
		if err != nil {
			return nil, fmt.Errorf("failed to create datastore: %v, due to %w", db.ID, err)
		}
		result[db.ID] = dbService
	}
	return result, nil
}

func getClient(db *config.Datastore, connections map[string]client.Service) (client.Service, client.Service, error) {
	var l1Client, l2Client client.Service
	var ok bool
	if db.Reference.Connection != "" {
		if l1Client, ok = connections[db.Reference.Connection]; !ok {
			return nil, nil, fmt.Errorf("faild to lookup datastore connection %v, for %v", db.Reference.Connection, db.ID)
		}
		if db.L2 != nil {
			if l2Client, ok = connections[db.L2.Connection]; !ok {
				return nil, nil, fmt.Errorf("faild to lookup datastore connection %v, for %v", db.L2.Connection, db.ID)
			}
		}
	}
	return l1Client, l2Client, nil
}

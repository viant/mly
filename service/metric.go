package service

import (
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/service/stat"
)

type ServiceOperation *gmetric.Operation

func NewServiceOperation(metrics *gmetric.Service, location, id string) ServiceOperation {
	return metrics.MultiOperationCounter(location,
		id+"Perf", id+" service performance", time.Microsecond,
		time.Minute, 2, stat.NewProvider())
}

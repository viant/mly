package lt

import "github.com/viant/cloudless/data/processor"

//Reporter represents load testing reported
type Reporter struct {
	processor.BaseReporter

	ServiceTimeMcs int64
	ServiceCount   int32

	ResponseTimeMcs int64
	ResponseCount   int32

	CacheTimeMcs int64
	CacheCount   int32

	AvgServiceTimeMcs  int64
	AvgResponseTimeMcs int64
	AvgCacheTimeMcs    int64
}

func NewReporter() processor.Reporter {
	return &Reporter{
		BaseReporter: processor.BaseReporter{
			Response: &processor.Response{Status: processor.StatusOk},
		},
	}
}

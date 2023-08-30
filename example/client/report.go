package client

import (
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/shared/client"
)

type (
	WPRun struct {
		Response *client.Response
		Err      error
	}

	WorkerPayload struct {
		Payload *CliPayload
		Worker  int

		Runs []WPRun
	}

	RepeatSet struct {
		Num       int
		WPayloads []WorkerPayload
	}

	Report struct {
		Start time.Time
		End   time.Time

		Runs    []RepeatSet
		Metrics []gmetric.Operation
	}
)

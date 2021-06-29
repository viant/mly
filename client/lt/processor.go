package lt

import (
	"context"
	"fmt"
	"github.com/viant/cloudless/data/processor"
	"github.com/viant/mly/client"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type clientKey string

var keyValue = clientKey("ml")

//Processor represents load testing processor
type Processor struct {
	*Config
}

//Pre runs preprocessing logic
func (p *Processor) Pre(ctx context.Context, reporter processor.Reporter) (context.Context, error) {
	aClient, err := client.New(p.Model, []*client.Host{client.NewHost(p.Hostname, p.Port)}, client.NewCacheSize(p.CacheSize))
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, keyValue, aClient), nil
}

func (p *Processor) Process(ctx context.Context, data []byte, reporter processor.Reporter) error {
	aReporter := reporter.(*Reporter)
	values := strings.Split(string(data), ",")
	if len(values) != len(p.Config.Fields) {
		return processor.NewDataCorruption(fmt.Sprintf("invalid message, expected %v, but had:%v", len(p.Config.Fields), len(values)))
	}
	aClient := ctx.Value(keyValue).(*client.Service)
	started := time.Now()
	msg := aClient.NewMessage()
	for i, field := range p.Config.Fields {
		switch field.Type {
		case "int":
			value, _ := strconv.Atoi(values[i])
			msg.IntKey(field.Name, value)
		case "boolean":
			value, _ := strconv.ParseBool(values[i])
			msg.BoolKey(field.Name, value)
		case "float32":
			value, _ := strconv.ParseFloat(values[i], 32)
			msg.FloatKey(field.Name, float32(value))
		default:
			msg.StringKey(field.Name, values[i])
		}
	}
	response := client.NewResponse(p.Config.NewResponseData())
	err := aClient.Run(ctx, msg, response)

	if err == nil {
		elapsed := time.Now().Sub(started)
		atomic.AddInt64(&aReporter.ResponseTimeMcs, elapsed.Microseconds())
		atomic.AddInt32(&aReporter.ResponseCount, 1)
		if response.Status == "cache" {
			atomic.AddInt32(&aReporter.CacheCount, 1)
			atomic.AddInt64(&aReporter.CacheTimeMcs, elapsed.Microseconds())
		} else {
			atomic.AddInt32(&aReporter.ServiceCount, 1)
			atomic.AddInt64(&aReporter.ServiceTimeMcs, elapsed.Microseconds())
		}
	}

	return err
}

func (p *Processor) Post(ctx context.Context, reporter processor.Reporter) error {
	aReporter := reporter.(*Reporter)
	aReporter.AvgResponseTimeMcs = int64(float32(aReporter.ResponseTimeMcs) / float32(aReporter.ResponseCount))
	aReporter.AvgServiceTimeMcs = int64(float32(aReporter.ServiceTimeMcs) / float32(aReporter.ServiceCount))
	aReporter.AvgCacheTimeMcs = int64(float32(aReporter.CacheTimeMcs) / float32(aReporter.CacheCount))
	aClient := ctx.Value(keyValue).(*client.Service)
	return aClient.Close()
}

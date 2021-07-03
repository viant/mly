package lt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/cloudless/data/processor"
	"io"
	"log"
	"time"
)

//Run runs a load tests
func Run(options *Options) {
	ctx := context.Background()
	cfg, err := NewConfigFromURL(ctx, options.ConfigURL)
	if err != nil {
		log.Fatalln(err)
	}
	proc := &Processor{
		Config: cfg,
	}
	fs := afs.New()
	_service := processor.New(&cfg.Config, afs.New(), proc, NewReporter)
	data, err := fs.DownloadWithURL(ctx, options.DataFile)
	if err != nil {
		log.Fatalln(err)
	}
	for i := 0; ; {
		reporter := _service.Do(ctx, &processor.Request{
			ReadCloser: io.NopCloser(bytes.NewReader(data)),
			SourceURL:  options.DataFile,
			StartTime:  time.Now(),
		})
		output, _ := json.MarshalIndent(reporter, "", "  ")
		fmt.Println(output)
		i++
		if i > options.Repeat {
			break
		}
	}
}

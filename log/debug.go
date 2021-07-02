package log

import (
	"fmt"
	"github.com/viant/toolbox"
	"os"
)

const DebugKey = "DEBUG"

var debugEnabled = toolbox.AsBoolean(os.Getenv(DebugKey))

func Debug(message string, args ...interface{}) {
	if !debugEnabled {
		return
	}
	fmt.Printf(message, args...)
}

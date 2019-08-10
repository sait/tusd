package cli

import (
	"log"
	"os"

	"github.com/sait/tusd"
)

var stdout = log.New(os.Stdout, "[tusd] ", log.Ldate|log.Ltime)
var stderr = log.New(os.Stderr, "[tusd] ", log.Ldate|log.Ltime)

func logEv(logOutput *log.Logger, eventName string, details ...string) {
	tusd.LogEvent(logOutput, eventName, details...)
}

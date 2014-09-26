package libwebsocketd

import (
	"fmt"
	"os"
)

// no real tests here so far, just a helper func for others

func logger_helper(logfunc func(args ...interface{})) *LogScope {
	log := new(LogScope)
	log.LogFunc = func(_ *LogScope, _ LogLevel, level string, cat string, f string, attr ...interface{}) {
		if v := os.Getenv("LOGALL"); v != "" {
			fmt.Printf("LOG-%s [%s] %s\n", level, cat, fmt.Sprintf(f, attr...))
		} else {
			logfunc(fmt.Sprintf("LOG-%s [%s]", level, cat), fmt.Sprintf(f, attr...))
		}
	}

	return log
}

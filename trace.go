package ktmt

import (
	"io"
	"log"
)

type (
	Logger interface {
		Println(v ...interface{})
		Printf(format string, v ...interface{})
	}

	NOOPLogger struct{}
)

func (NOOPLogger) Println(v ...interface{})               {}
func (NOOPLogger) Printf(format string, v ...interface{}) {}

var (
	ERROR    Logger = NOOPLogger{}
	CRITICAL Logger = NOOPLogger{}
	WARN     Logger = NOOPLogger{}
	INFO     Logger = NOOPLogger{}
	DEBUG    Logger = NOOPLogger{}
)

func SetLogger(out io.Writer) {
	ERROR = log.New(out, "[ERROR]", log.LstdFlags)
	CRITICAL = log.New(out, "[CRITICAL]", log.LstdFlags)
	WARN = log.New(out, "[WARN]", log.LstdFlags)
	INFO = log.New(out, "[INFO]", log.LstdFlags)
	DEBUG = log.New(out, "[DEBUG]", log.LstdFlags)
}

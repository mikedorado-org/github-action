package hydro

import (
	"fmt"

	"github.com/github/github-telemetry-go/kvp"
	"github.com/github/github-telemetry-go/log"
	"github.com/github/hydro-client-go/v6/pkg/hydro"
)

type Logger struct {
	print func(string, ...kvp.Field)
	panic func(string, ...kvp.Field)
	fatal func(string, ...kvp.Field)
}

// Use the global otel logger to configure a hydro logger
func NewHydroLogger(name string) hydro.Logger {
	logger := log.Named("hydro." + name)
	s := Logger{
		print: logger.Info,
		panic: logger.Error,
		fatal: logger.Fatal,
	}
	return &s
}

func (l *Logger) Print(v ...interface{}) {
	l.print(fmt.Sprint(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.print(fmt.Sprintf(format, v...))
}

func (l *Logger) Println(v ...interface{}) {
	l.print(fmt.Sprintln(v...))
}

func (l *Logger) Panic(v ...interface{}) {
	l.panic(fmt.Sprint(v...))
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	l.panic(fmt.Sprintf(format, v...))
}

func (l *Logger) Panicln(v ...interface{}) {
	l.panic(fmt.Sprintln(v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	l.fatal(fmt.Sprint(v...))
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.fatal(fmt.Sprintf(format, v...))
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.fatal(fmt.Sprintln(v...))
}

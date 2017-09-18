package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	DEBUG = iota
	INFO
	WARNING
	ERROR
)

type Logger struct {
	level int
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	err   *log.Logger
}

var levels = map[string]int{
	"DEBUG":   DEBUG,
	"INFO":    INFO,
	"WARNING": WARNING,
	"ERROR":   ERROR,
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.debug.Printf(format, v...)
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		l.info.Printf(format, v...)
	}
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WARNING {
		l.warn.Printf(format, v...)
	}
}

func (l *Logger) Err(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.err.Printf(format, v...)
	}
}

type invalidLogLevelError string

func (e invalidLogLevelError) Error() string {
	return fmt.Sprintf("%s: invalid log level", e)
}

func NewLogger(level string) (*Logger, error) {
	levelnum, ok := levels[strings.ToUpper(level)]
	if !ok {
		return nil, invalidLogLevelError(level)
	}

	return &Logger{
		level: levelnum,
		debug: log.New(os.Stdout, "DEBUG: ", 0),
		info:  log.New(os.Stdout, "INFO: ", 0),
		warn:  log.New(os.Stdout, "WARNING: ", 0),
		err:   log.New(os.Stderr, "ERROR: ", 0),
	}, nil
}

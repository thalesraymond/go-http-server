package handler

import "log"

type Logger interface {
	Error(msg string, err error)
	Info(msg string)
	Debug(msg string)
	Warn(msg string)
}

type defaultLogger struct{ l *log.Logger }

func (l *defaultLogger) Error(msg string, err error) {
	l.l.Printf("ERROR %s: %v", msg, err)
}

func (l *defaultLogger) Info(msg string) {
	l.l.Printf("INFO %s", msg)
}

func (l *defaultLogger) Debug(msg string) {
	l.l.Printf("DEBUG %s", msg)
}

func (l *defaultLogger) Warn(msg string) {
	l.l.Printf("WARN %s", msg)
}

func NewLogger() Logger {
	return &defaultLogger{l: log.Default()}
}

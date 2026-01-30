package main

import (
	"log"
	"os"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

type Logger struct {
	info    *log.Logger
	error   *log.Logger
	warning *log.Logger
	debug   *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		info:    log.New(os.Stdout, colorGreen+"[INFO]"+colorReset+" ", log.LstdFlags),
		error:   log.New(os.Stderr, colorRed+"[ERROR]"+colorReset+" ", log.LstdFlags),
		warning: log.New(os.Stdout, colorYellow+"[WARN]"+colorReset+" ", log.LstdFlags),
		debug:   log.New(os.Stdout, colorCyan+"[DEBUG]"+colorReset+" ", log.LstdFlags),
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.error.Printf(format, v...)
}

func (l *Logger) Warning(format string, v ...interface{}) {
	l.warning.Printf(format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.debug.Printf(format, v...)
}

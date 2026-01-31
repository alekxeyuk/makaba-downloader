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
	trace   *log.Logger
	fatal   *log.Logger
	success *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		info:    log.New(os.Stdout, colorGreen+"[INFO]"+" ", log.LstdFlags),
		error:   log.New(os.Stderr, colorRed+"[ERROR]"+" ", log.LstdFlags),
		warning: log.New(os.Stdout, colorYellow+"[WARN]"+" ", log.LstdFlags),
		debug:   log.New(os.Stdout, colorCyan+"[DEBUG]"+" ", log.LstdFlags),
		trace:   log.New(os.Stdout, colorBlue+"[TRACE]"+" ", log.LstdFlags),
		fatal:   log.New(os.Stderr, colorPurple+"[FATAL]"+" ", log.LstdFlags),
		success: log.New(os.Stdout, colorWhite+"[SUCCESS]"+" ", log.LstdFlags),
	}
}

func (l *Logger) Info(format string, v ...any) {
	l.info.Printf(format+colorReset, v...)
}

func (l *Logger) Error(format string, v ...any) {
	l.error.Printf(format+colorReset, v...)
}

func (l *Logger) Warning(format string, v ...any) {
	l.warning.Printf(format+colorReset, v...)
}

func (l *Logger) Debug(format string, v ...any) {
	l.debug.Printf(format+colorReset, v...)
}

func (l *Logger) Trace(format string, v ...any) {
	l.trace.Printf(format+colorReset, v...)
}

func (l *Logger) Fatal(format string, v ...any) {
	l.fatal.Printf(format+colorReset, v...)
	os.Exit(1)
}

func (l *Logger) Success(format string, v ...any) {
	l.success.Printf(format+colorReset, v...)
}

// Global logger instance for use throughout the package
var Log = NewLogger()

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
		info:    log.New(os.Stdout, colorGreen+"[INFO]"+colorReset+" ", log.LstdFlags),
		error:   log.New(os.Stderr, colorRed+"[ERROR]"+colorReset+" ", log.LstdFlags),
		warning: log.New(os.Stdout, colorYellow+"[WARN]"+colorReset+" ", log.LstdFlags),
		debug:   log.New(os.Stdout, colorCyan+"[DEBUG]"+colorReset+" ", log.LstdFlags),
		trace:   log.New(os.Stdout, colorBlue+"[TRACE]"+colorReset+" ", log.LstdFlags),
		fatal:   log.New(os.Stderr, colorPurple+"[FATAL]"+colorReset+" ", log.LstdFlags),
		success: log.New(os.Stdout, colorWhite+"[SUCCESS]"+colorReset+" ", log.LstdFlags),
	}
}

func (l *Logger) Info(format string, v ...any) {
	l.info.Printf(format, v...)
}

func (l *Logger) Error(format string, v ...any) {
	l.error.Printf(format, v...)
}

func (l *Logger) Warning(format string, v ...any) {
	l.warning.Printf(format, v...)
}

func (l *Logger) Debug(format string, v ...any) {
	l.debug.Printf(format, v...)
}

func (l *Logger) Trace(format string, v ...any) {
	l.trace.Printf(format, v...)
}

func (l *Logger) Fatal(format string, v ...any) {
	l.fatal.Printf(format, v...)
	os.Exit(1)
}

func (l *Logger) Success(format string, v ...any) {
	l.success.Printf(format, v...)
}

// Global logger instance for use throughout the package
var Log = NewLogger()

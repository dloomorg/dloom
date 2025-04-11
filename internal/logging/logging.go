package logging

import (
	"fmt"
	"os"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
)

type Logger struct {
	UseColors bool
}

func (l *Logger) LogError(format string, args ...interface{}) {
	var message string
	if l.UseColors {
		message = Red + "[ERROR]: " + format + Reset
	} else {
		message = "[ERROR]: " + format
	}

	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

func (l *Logger) LogWarning(format string, args ...interface{}) {
	var message string
	if l.UseColors {
		message = Yellow + "[WARNING]: " + format + Reset
	} else {
		message = "[WARNING]: " + format
	}

	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

func (l *Logger) LogTrace(format string, args ...interface{}) {
	var message string
	if l.UseColors {
		message = Cyan + "[TRACE]: " + format + Reset
	} else {
		message = "[TRACE]: " + format
	}

	fmt.Printf(message+"\n", args...)
}

func (l *Logger) LogInfo(format string, args ...interface{}) {
	var message string
	if l.UseColors {
		message = Green + "[INFO]: " + format + Reset
	} else {
		message = "[INFO]: " + format
	}

	fmt.Printf(message+"\n", args...)
}

func (l *Logger) LogInfoNoReturn(format string, args ...interface{}) {
	var message string
	if l.UseColors {
		message = Green + "[INFO]: " + format + Reset
	} else {
		message = "[INFO]: " + format
	}

	fmt.Printf(message, args...)
}

func (l *Logger) LogDryRun(format string, args ...interface{}) {
	var message string
	if l.UseColors {
		message = Blue + "[DRY RUN]: " + format + Reset
	} else {
		message = "[DRY RUN]: " + format
	}

	fmt.Printf(message+"\n", args...)
}

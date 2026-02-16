package logger

import (
	"log"
	"os"
	"sync/atomic"
	"time"
)

type Level int32

const (
	Debug Level = iota
	Info
	Warn
	Error
)

var current atomic.Int32

// Init sets initial level and basic format.
func Init(level string) {
	SetLevel(level)
	log.SetOutput(os.Stdout)
	log.SetFlags(0) // we print our own timestamp
}

// SetLevel changes the level at runtime.
func SetLevel(level string) {
	switch level {
	case "debug":
		current.Store(int32(Debug))
	case "warn":
		current.Store(int32(Warn))
	case "error":
		current.Store(int32(Error))
	default:
		current.Store(int32(Info))
	}
}

// GetLevel returns the current level string.
func GetLevel() string {
	switch Level(current.Load()) {
	case Debug:
		return "debug"
	case Warn:
		return "warn"
	case Error:
		return "error"
	default:
		return "info"
	}
}

func ts() string { return time.Now().Format(time.RFC3339) }

func Debugf(format string, args ...any) {
	if Level(current.Load()) <= Debug {
		log.Printf(ts()+" [DEBUG] "+format, args...)
	}
}
func Infof(format string, args ...any) {
	if Level(current.Load()) <= Info {
		log.Printf(ts()+" [INFO ] "+format, args...)
	}
}
func Warnf(format string, args ...any) {
	if Level(current.Load()) <= Warn {
		log.Printf(ts()+" [WARN ] "+format, args...)
	}
}
func Errorf(format string, args ...any) {
	if Level(current.Load()) <= Error {
		log.Printf(ts()+" [ERROR] "+format, args...)
	}
}

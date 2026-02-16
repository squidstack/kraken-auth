package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel string
	}{
		{"debug level", "debug", "debug"},
		{"info level", "info", "info"},
		{"warn level", "warn", "warn"},
		{"error level", "error", "error"},
		{"unknown defaults to info", "unknown", "info"},
		{"empty defaults to info", "", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.level)
			if got := GetLevel(); got != tt.expectedLevel {
				t.Errorf("Init(%q) level = %q, want %q", tt.level, got, tt.expectedLevel)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected string
	}{
		{"set debug", "debug", "debug"},
		{"set info", "info", "info"},
		{"set warn", "warn", "warn"},
		{"set error", "error", "error"},
		{"invalid defaults to info", "invalid", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			if got := GetLevel(); got != tt.expected {
				t.Errorf("SetLevel(%q) = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestGetLevel(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			SetLevel(level)
			if got := GetLevel(); got != level {
				t.Errorf("GetLevel() = %q, want %q", got, level)
			}
		})
	}
}

func TestDebugf(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)
	log.SetFlags(0)

	tests := []struct {
		name        string
		level       string
		shouldPrint bool
	}{
		{"debug level prints debug", "debug", true},
		{"info level skips debug", "info", false},
		{"warn level skips debug", "warn", false},
		{"error level skips debug", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			SetLevel(tt.level)
			Debugf("test message")

			output := buf.String()
			hasDebug := strings.Contains(output, "[DEBUG]") && strings.Contains(output, "test message")
			if hasDebug != tt.shouldPrint {
				t.Errorf("Debugf at level %q: shouldPrint=%v, got output=%q", tt.level, tt.shouldPrint, output)
			}
		})
	}
}

func TestInfof(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)
	log.SetFlags(0)

	tests := []struct {
		name        string
		level       string
		shouldPrint bool
	}{
		{"debug level prints info", "debug", true},
		{"info level prints info", "info", true},
		{"warn level skips info", "warn", false},
		{"error level skips info", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			SetLevel(tt.level)
			Infof("test message")

			output := buf.String()
			hasInfo := strings.Contains(output, "[INFO ]") && strings.Contains(output, "test message")
			if hasInfo != tt.shouldPrint {
				t.Errorf("Infof at level %q: shouldPrint=%v, got output=%q", tt.level, tt.shouldPrint, output)
			}
		})
	}
}

func TestWarnf(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)
	log.SetFlags(0)

	tests := []struct {
		name        string
		level       string
		shouldPrint bool
	}{
		{"debug level prints warn", "debug", true},
		{"info level prints warn", "info", true},
		{"warn level prints warn", "warn", true},
		{"error level skips warn", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			SetLevel(tt.level)
			Warnf("test message")

			output := buf.String()
			hasWarn := strings.Contains(output, "[WARN ]") && strings.Contains(output, "test message")
			if hasWarn != tt.shouldPrint {
				t.Errorf("Warnf at level %q: shouldPrint=%v, got output=%q", tt.level, tt.shouldPrint, output)
			}
		})
	}
}

func TestErrorf(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)
	log.SetFlags(0)

	tests := []struct {
		name        string
		level       string
		shouldPrint bool
	}{
		{"debug level prints error", "debug", true},
		{"info level prints error", "info", true},
		{"warn level prints error", "warn", true},
		{"error level prints error", "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			SetLevel(tt.level)
			Errorf("test message")

			output := buf.String()
			hasError := strings.Contains(output, "[ERROR]") && strings.Contains(output, "test message")
			if hasError != tt.shouldPrint {
				t.Errorf("Errorf at level %q: shouldPrint=%v, got output=%q", tt.level, tt.shouldPrint, output)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{Debug, "debug"},
		{Info, "info"},
		{Warn, "warn"},
		{Error, "error"},
		{Level(99), "info"}, // unknown level defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			current.Store(int32(tt.level))
			if got := GetLevel(); got != tt.expected {
				t.Errorf("Level %d: GetLevel() = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

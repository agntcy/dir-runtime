// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck,wsl_v5
package utils

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		kind          string
		componentType string
	}{
		{
			name:          "creates logger with valid params",
			kind:          "process",
			componentType: "discovery",
		},
		{
			name:          "creates logger with empty strings",
			kind:          "",
			componentType: "",
		},
		{
			name:          "creates logger with special characters",
			kind:          "test-kind",
			componentType: "test_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.kind, tt.componentType)
			if logger == nil {
				t.Error("NewLogger returned nil")
			}
			if logger.Logger == nil {
				t.Error("Logger.Logger is nil")
			}
		})
	}
}

func TestLoggerWith(t *testing.T) {
	logger := NewLogger("test", "component")

	childLogger := logger.With("key", "value", "num", 42)
	if childLogger == nil {
		t.Error("With returned nil")
	}
	if childLogger.Logger == nil {
		t.Error("Child logger's Logger is nil")
	}

	// Ensure original logger is not modified
	if logger == childLogger {
		t.Error("With should return a new logger instance")
	}
}

func TestLoggerMethods(t *testing.T) {
	logger := NewLogger("test", "component")

	// These should not panic
	t.Run("Info does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info panicked: %v", r)
			}
		}()
		logger.Info("test message", "key", "value")
	})

	t.Run("Error does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error panicked: %v", r)
			}
		}()
		logger.Error("test error", "key", "value")
	})

	t.Run("Warn does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn panicked: %v", r)
			}
		}()
		logger.Warn("test warning", "key", "value")
	})

	t.Run("Debug does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug panicked: %v", r)
			}
		}()
		logger.Debug("test debug", "key", "value")
	})
}

func TestSetDefaultLogger_RedirectsLoggersCreatedEarlier(t *testing.T) {
	early := NewLogger("store", "sql").With("workload", "w1")

	var buf bytes.Buffer
	SetDefaultLogger(slog.New(slog.NewTextHandler(&buf, nil)).With("component", "daemon"))
	t.Cleanup(func() {
		SetDefaultLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	})

	early.Info("registered workload")

	out := buf.String()
	for _, want := range []string{"msg=\"registered workload\"", "component=daemon", "kind=store", "type=sql", "workload=w1"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q is missing %q", out, want)
		}
	}
}

func TestSetDefaultLogger_KeepsGroupsInOrder(t *testing.T) {
	var buf bytes.Buffer
	SetDefaultLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() {
		SetDefaultLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	})

	logger := NewLogger("runtime", "process")
	logger.Logger.WithGroup("req").Info("probe", "url", "http://x")

	if out := buf.String(); !strings.Contains(out, "req.url=http://x") || !strings.Contains(out, "kind=runtime") {
		t.Errorf("unexpected output %q", out)
	}
}

func TestSetDefaultLogger_IgnoresNil(t *testing.T) {
	SetDefaultLogger(nil)
	NewLogger("k", "t").Info("still logs")
}

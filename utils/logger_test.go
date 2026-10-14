// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck,wsl_v5
package utils

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
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

func restoreDefaultLogger(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		SetDefaultLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	})
}

func TestSetDefaultLogger_WithALoggerFromNewLoggerDoesNotRecurse(t *testing.T) {
	var buf bytes.Buffer
	SetDefaultLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	restoreDefaultLogger(t)

	SetDefaultLogger(NewLogger("runtime", "daemon").Logger)

	done := make(chan struct{})
	go func() {
		defer close(done)
		NewLogger("store", "sql").Info("registered workload")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("logging through a base made by NewLogger did not return")
	}

	if out := buf.String(); !strings.Contains(out, "kind=store") || !strings.Contains(out, "registered workload") {
		t.Errorf("unexpected output %q", out)
	}
}

// ownershipHandler overwrites the attribute slices it is given, as slog.Handler allows.
type ownershipHandler struct {
	slog.Handler
}

func (h ownershipHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	child := h.Handler.WithAttrs(attrs)
	for i := range attrs {
		attrs[i] = slog.String("clobbered", "yes")
	}

	return ownershipHandler{child}
}

func TestDeferredHandler_PassesEachReplayItsOwnAttrs(t *testing.T) {
	var buf bytes.Buffer
	SetDefaultLogger(slog.New(ownershipHandler{slog.NewTextHandler(&buf, nil)}))
	restoreDefaultLogger(t)

	logger := NewLogger("store", "sql")
	logger.Info("first")
	logger.Info("second")

	out := buf.String()
	if strings.Count(out, "kind=store") != 2 || strings.Contains(out, "clobbered") {
		t.Errorf("attributes were not kept across logs: %q", out)
	}
}

// attrLevelHandler enables Debug only for loggers carrying kind=debuggable.
type attrLevelHandler struct {
	slog.Handler
	debug bool
}

func (h attrLevelHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.debug || level >= slog.LevelInfo
}

func (h attrLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	debug := h.debug
	for _, a := range attrs {
		if a.Key == "kind" && a.Value.String() == "debuggable" {
			debug = true
		}
	}

	return attrLevelHandler{h.Handler.WithAttrs(attrs), debug}
}

func TestDeferredHandler_EnabledUsesTheDerivedHandler(t *testing.T) {
	var buf bytes.Buffer
	SetDefaultLogger(slog.New(attrLevelHandler{Handler: slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})}))
	restoreDefaultLogger(t)

	NewLogger("debuggable", "x").Debug("shown")
	NewLogger("plain", "x").Debug("hidden")

	out := buf.String()
	if !strings.Contains(out, "shown") || strings.Contains(out, "hidden") {
		t.Errorf("Enabled ignored the logger's attributes: %q", out)
	}
}

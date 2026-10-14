// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"log/slog"
	"os"
	"slices"
	"sync/atomic"
)

// Logger wraps slog.Logger with predefined attributes.
type Logger struct {
	*slog.Logger
}

// baseHandler is where every logger created by NewLogger writes. It defaults to JSON
// on stdout and can be replaced with SetDefaultLogger, e.g. by a program that embeds
// dir-runtime (such as dirctl daemon) so that all its logs share one format.
var baseHandler atomic.Pointer[slog.Handler]

func init() {
	var h slog.Handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	baseHandler.Store(&h)
}

// SetDefaultLogger makes every dir-runtime logger write through l, including loggers
// created before the call (package-level loggers are created at import time). Their
// kind and type attributes are kept. A nil logger is ignored. A logger made by
// NewLogger is resolved to the handler it writes to now, so it can't become its own base.
func SetDefaultLogger(l *slog.Logger) {
	if l == nil {
		return
	}

	h := l.Handler()
	if deferred, ok := h.(*deferredHandler); ok {
		h = deferred.current()
	}

	baseHandler.Store(&h)
}

// NewLogger creates a logger with kind and type attributes for the component.
func NewLogger(kind, componentType string) *Logger {
	return &Logger{
		Logger: slog.New(&deferredHandler{}).With(
			slog.String("kind", kind),
			slog.String("type", componentType),
		),
	}
}

// deferredHandler applies its attributes and groups to the current base handler each
// time it handles a record, so a logger created early follows a later SetDefaultLogger.
type deferredHandler struct {
	ops []handlerOp
}

// handlerOp is one WithAttrs or WithGroup call, replayed in order on the base handler.
type handlerOp struct {
	attrs []slog.Attr
	group string
}

func (h *deferredHandler) current() slog.Handler {
	base := *baseHandler.Load()

	for _, op := range h.ops {
		if op.group != "" {
			base = base.WithGroup(op.group)
		} else {
			// WithAttrs owns the slice it is given, so each replay passes its own copy.
			base = base.WithAttrs(slices.Clone(op.attrs))
		}
	}

	return base
}

// Enabled asks the same derived handler Handle uses, since a handler's decision may
// depend on the attributes and groups added to it.
func (h *deferredHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.current().Enabled(ctx, level)
}

//nolint:gocritic // slog.Handler requires the record by value.
func (h *deferredHandler) Handle(ctx context.Context, record slog.Record) error {
	//nolint:wrapcheck // the base handler's error is the caller's error.
	return h.current().Handle(ctx, record)
}

func (h *deferredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.with(handlerOp{attrs: attrs})
}

func (h *deferredHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return h.with(handlerOp{group: name})
}

func (h *deferredHandler) with(op handlerOp) *deferredHandler {
	ops := make([]handlerOp, len(h.ops), len(h.ops)+1)
	copy(ops, h.ops)

	return &deferredHandler{ops: append(ops, op)}
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// With returns a new Logger with the provided attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

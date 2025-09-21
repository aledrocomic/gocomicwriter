/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

// Package log provides centralized slog-based logging for the application.
// It wraps the standard slog with a small configuration surface and a custom
// handler that enriches records with common fields as described in
// docs/go_comic_writer_concept.md (component, operation, project path, etc.).
package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gocomic/internal/version"

	// lumberjack is optional; used only if file logging is enabled
	lj "gopkg.in/natefinch/lumberjack.v2"
)

// Options controls logger initialization.
// Values can be provided directly or via environment variables:
//   - GCW_LOG_LEVEL=debug|info|warn|error
//   - GCW_LOG_FORMAT=console|json
//   - GCW_LOG_FILE=<path> (enables file logging with rotation)
//   - GCW_LOG_SOURCE=true|false (include source)
//
// If File is set, a rotating file writer will be used.
// Defaults: INFO level, console format, no source.
//
// Note: Keep this minimal; future phases may add per-project file logs.
type Options struct {
	Level     string
	Format    string // "console" or "json"
	AddSource bool
	File      string // optional path for file logging (rotated)
}

var (
	defaultLoggerMu sync.RWMutex
	defaultLogger   *slog.Logger
)

// L returns the default application logger, initializing from env if needed.
func L() *slog.Logger {
	defaultLoggerMu.RLock()
	l := defaultLogger
	defaultLoggerMu.RUnlock()
	if l != nil {
		return l
	}
	// lazy init from env
	Init(FromEnv())
	defaultLoggerMu.RLock()
	l = defaultLogger
	defaultLoggerMu.RUnlock()
	return l
}

// Init configures the global logger and sets slog.Default as well.
func Init(opts Options) {
	lvl := parseLevel(opts.Level)
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = "console"
	}

	var handlers []slog.Handler
	// Console handler
	var consoleHandler slog.Handler
	if format == "json" {
		consoleHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl, AddSource: opts.AddSource})
	} else {
		consoleHandler = &prettyTextHandler{opts: prettyOpts{Level: lvl, AddSource: opts.AddSource}, w: os.Stderr}
	}
	handlers = append(handlers, withEnricher(consoleHandler))

	// Optional file handler with rotation
	if strings.TrimSpace(opts.File) != "" {
		w := &lj.Logger{Filename: opts.File, MaxSize: 10, MaxBackups: 3, MaxAge: 28, Compress: true}
		fh := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl, AddSource: opts.AddSource})
		handlers = append(handlers, withEnricher(fh))
	}

	var h slog.Handler
	if len(handlers) == 1 {
		h = handlers[0]
	} else {
		h = multiHandler(handlers...)
	}

	logger := slog.New(h)
	// Attach static attrs
	logger = logger.With(
		slog.String("app", "gocomic"),
		slog.String("ver", version.Version),
		slog.Time("ts_init", time.Now()),
	)

	defaultLoggerMu.Lock()
	defaultLogger = logger
	defaultLoggerMu.Unlock()
	slog.SetDefault(logger)
}

// FromEnv builds Options from environment variables.
func FromEnv() Options {
	return Options{
		Level:     getenv("GCW_LOG_LEVEL", "info"),
		Format:    getenv("GCW_LOG_FORMAT", "console"),
		AddSource: strings.EqualFold(getenv("GCW_LOG_SOURCE", "false"), "true"),
		File:      os.Getenv("GCW_LOG_FILE"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// WithComponent returns a logger with the component attribute pre-set.
func WithComponent(name string) *slog.Logger { return L().With(slog.String("component", name)) }

// WithOperation annotates the logger with an operation name.
func WithOperation(l *slog.Logger, op string) *slog.Logger { return l.With(slog.String("op", op)) }

// parseLevel converts a string to slog.Level.
func parseLevel(s string) slog.Leveler {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// multiHandler fans out log records to multiple handlers.
func multiHandler(handlers ...slog.Handler) slog.Handler { return &multi{hs: handlers} }

type multi struct{ hs []slog.Handler }

func (m *multi) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.hs {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multi) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range m.hs {
		if err := h.Handle(ctx, r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *multi) WithAttrs(attrs []slog.Attr) slog.Handler {
	res := make([]slog.Handler, len(m.hs))
	for i, h := range m.hs {
		res[i] = h.WithAttrs(attrs)
	}
	return &multi{hs: res}
}

func (m *multi) WithGroup(name string) slog.Handler {
	res := make([]slog.Handler, len(m.hs))
	for i, h := range m.hs {
		res[i] = h.WithGroup(name)
	}
	return &multi{hs: res}
}

// enricher adds common attributes and passes to the underlying handler.
func withEnricher(h slog.Handler) slog.Handler { return &enrich{next: h} }

type enrich struct{ next slog.Handler }

func (e *enrich) Enabled(ctx context.Context, level slog.Level) bool {
	return e.next.Enabled(ctx, level)
}

func (e *enrich) Handle(ctx context.Context, r slog.Record) error {
	// Potential place to inject global fields (e.g., project path) from ctx.
	return e.next.Handle(ctx, r)
}

func (e *enrich) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &enrich{next: e.next.WithAttrs(attrs)}
}
func (e *enrich) WithGroup(name string) slog.Handler { return &enrich{next: e.next.WithGroup(name)} }

// prettyTextHandler is a minimal custom text handler for console output.
// It prints human-friendly, one-line logs: ts level msg key=val... and supports
// accumulating attributes and groups.

type prettyTextHandler struct {
	opts   prettyOpts
	w      io.Writer
	attrs  []slog.Attr
	groups []string
}

type prettyOpts struct {
	Level     slog.Leveler
	AddSource bool
}

func (h *prettyTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level()
}

func (h *prettyTextHandler) level() slog.Level {
	if h.opts.Level == nil {
		return slog.LevelInfo
	}
	switch v := h.opts.Level.(type) {
	case slog.Level:
		return v
	case *slog.LevelVar:
		return v.Level()
	default:
		return slog.LevelInfo
	}
}

func (h *prettyTextHandler) Handle(_ context.Context, r slog.Record) error {
	b := &strings.Builder{}
	b.Grow(256)
	b.WriteString(time.Now().Format(time.RFC3339))
	b.WriteString(" ")
	b.WriteString(levelString(r.Level))
	b.WriteString(" ")
	if r.Message != "" {
		b.WriteString(r.Message)
	}
	// base attrs
	keyPrefix := ""
	if len(h.groups) > 0 {
		keyPrefix = strings.Join(h.groups, ".") + "."
	}
	writeAttrs := func(attrs []slog.Attr) {
		for i, a := range attrs {
			if i == 0 && b.Len() > 0 {
				b.WriteString(" ")
			} else {
				b.WriteString(" ")
			}
			b.WriteString(keyPrefix)
			b.WriteString(a.Key)
			b.WriteString("=")
			b.WriteString(attrValueString(a.Value))
		}
	}
	if len(h.attrs) > 0 {
		writeAttrs(h.attrs)
	}
	var recAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		recAttrs = append(recAttrs, a)
		return true
	})
	if len(recAttrs) > 0 {
		writeAttrs(recAttrs)
	}
	if h.opts.AddSource {
		// Use a version-tolerant way to access source info. Newer Go versions
		// expose Record.Source() *slog.Source; older ones do not.
		if rw, ok := any(r).(interface{ Source() *slog.Source }); ok {
			if src := rw.Source(); src != nil {
				b.WriteString(" ")
				b.WriteString("src=")
				b.WriteString(src.File)
				b.WriteString(":")
				b.WriteString(strconv.FormatInt(int64(src.Line), 10))
			}
		}
	}
	b.WriteString("\n")
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *prettyTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	na := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	na = append(na, h.attrs...)
	na = append(na, attrs...)
	return &prettyTextHandler{opts: h.opts, w: h.w, attrs: na, groups: append([]string(nil), h.groups...)}
}

func (h *prettyTextHandler) WithGroup(name string) slog.Handler {
	ng := append([]string(nil), h.groups...)
	ng = append(ng, name)
	return &prettyTextHandler{opts: h.opts, w: h.w, attrs: append([]slog.Attr(nil), h.attrs...), groups: ng}
}

func levelString(l slog.Level) string {
	switch l {
	case slog.LevelDebug:
		return "DBG"
	case slog.LevelInfo:
		return "INF"
	case slog.LevelWarn:
		return "WRN"
	case slog.LevelError:
		return "ERR"
	default:
		return l.String()
	}
}

func attrValueString(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindFloat64:
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(v.Float64(), 'f', -1, 64), "0"), ".")
	case slog.KindBool:
		if v.Bool() {
			return "true"
		}
		return "false"
	default:
		return v.String()
	}
}

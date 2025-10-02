/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

// Package telemetry provides a tiny, privacy‑respecting, opt‑in event sender
// for anonymous usage metrics and optional crash uploads.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	applog "gocomicwriter/internal/log"
	"gocomicwriter/internal/version"
)

// Config holds runtime configuration for telemetry and crash uploads.
// All telemetry is strictly opt‑in and disabled by default.
//
// Environment variables (read by FromEnv):
// - GCW_TELEMETRY_OPT_IN: "1", "true", "yes" to enable metrics
// - GCW_TELEMETRY_URL: base URL to POST JSON events to (e.g., https://example.com/telemetry)
// - GCW_CRASH_UPLOAD_URL: URL to POST crash reports to
// - GCW_TELEMETRY_TIMEOUT_MS: optional request timeout, default 1500ms
// - GCW_TELEMETRY_DEBUG: if set, logs event send attempts
//
// If no URLs are set, events are dropped (no‑ops), even if opt‑in is true.
type Config struct {
	OptIn        bool
	EventsURL    string
	CrashURL     string
	Timeout      time.Duration
	DebugLogging bool
}

func FromEnv() Config {
	optIn := parseBool(os.Getenv("GCW_TELEMETRY_OPT_IN"))
	cfg := Config{
		OptIn:        optIn,
		EventsURL:    strings.TrimSpace(os.Getenv("GCW_TELEMETRY_URL")),
		CrashURL:     strings.TrimSpace(os.Getenv("GCW_CRASH_UPLOAD_URL")),
		Timeout:      1500 * time.Millisecond,
		DebugLogging: os.Getenv("GCW_TELEMETRY_DEBUG") != "",
	}
	if ms := strings.TrimSpace(os.Getenv("GCW_TELEMETRY_TIMEOUT_MS")); ms != "" {
		if v, err := time.ParseDuration(ms + "ms"); err == nil {
			cfg.Timeout = v
		}
	}
	return cfg
}

func parseBool(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

// Client is a minimal async sender; it drops events silently on errors.
// It never blocks the UI; channel is bounded.
type Client struct {
	cfg    Config
	log    *slog.Logger
	cli    *http.Client
	q      chan any
	once   sync.Once
	closed chan struct{}
}

var defaultClient *Client
var defaultOnce sync.Once

// InitDefault initializes the package‑level default client from env when first used.
func InitDefault() {
	defaultOnce.Do(func() {
		NewDefault(FromEnv())
	})
}

// NewDefault creates and installs the default client with cfg.
func NewDefault(cfg Config) {
	defaultClient = New(cfg)
}

// New constructs a client.
func New(cfg Config) *Client {
	l := applog.WithComponent("telemetry")
	c := &Client{
		cfg:    cfg,
		log:    l,
		cli:    &http.Client{Timeout: cfg.Timeout},
		q:      make(chan any, 64),
		closed: make(chan struct{}),
	}
	go c.loop()
	return c
}

// Enabled reports whether anonymous telemetry is enabled and an endpoint is configured.
func (c *Client) Enabled() bool { return c != nil && c.cfg.OptIn && c.cfg.EventsURL != "" }

// Enabled reports whether anonymous telemetry is enabled using the default client.
func Enabled() bool {
	InitDefault()
	return defaultClient.Enabled()
}

// Event posts a small JSON event if enabled. Safe to call from anywhere.
func (c *Client) Event(name string, props map[string]any) {
	if !c.Enabled() || name == "" {
		return
	}
	payload := map[string]any{
		"name":    name,
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"version": version.String(),
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	}
	for k, v := range props {
		// best‑effort shallow copy, props must be non‑PII
		payload[k] = v
	}
	select {
	case c.q <- payload:
	default:
		// drop if queue full
	}
}

// Event using default client.
func Event(name string, props map[string]any) { InitDefault(); defaultClient.Event(name, props) }

// Flush waits briefly for the queue to drain.
func (c *Client) Flush(ctx context.Context) {
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if len(c.q) == 0 || time.Now().After(deadline) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// Close stops background goroutine.
func (c *Client) Close() { c.once.Do(func() { close(c.closed) }) }

func (c *Client) loop() {
	for {
		select {
		case <-c.closed:
			return
		case item := <-c.q:
			c.send(item)
		}
	}
}

func (c *Client) send(item any) {
	buf, _ := json.Marshal(item)
	req, err := http.NewRequest(http.MethodPost, c.cfg.EventsURL, bytes.NewReader(buf))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.cli.Do(req)
	if err != nil {
		if c.cfg.DebugLogging {
			c.log.Debug("telemetry send failed", slog.Any("err", err))
		}
		return
	}
	_ = resp.Body.Close()
	if c.cfg.DebugLogging {
		c.log.Debug("telemetry event sent")
	}
}

// UploadCrash posts an already‑serialized crash report to the configured crash URL if opt‑in.
func (c *Client) UploadCrash(report []byte) {
	if c == nil || !c.cfg.OptIn || c.cfg.CrashURL == "" {
		return
	}
	go func(b []byte) {
		req, err := http.NewRequest(http.MethodPost, c.cfg.CrashURL, bytes.NewReader(b))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		resp, err := c.cli.Do(req)
		if err != nil {
			if c.cfg.DebugLogging {
				c.log.Debug("crash upload failed", slog.Any("err", err))
			}
			return
		}
		_ = resp.Body.Close()
		if c.cfg.DebugLogging {
			c.log.Debug("crash report uploaded")
		}
	}(append([]byte(nil), report...))
}

// UploadCrash using default client.
func UploadCrash(report []byte) { InitDefault(); defaultClient.UploadCrash(report) }

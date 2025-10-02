/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package config

import (
	"os"
	"testing"
)

func TestEnvOverridesBackendURL(t *testing.T) {
	old := os.Getenv(EnvBackendURL)
	_ = os.Setenv(EnvBackendURL, "https://example.test:8443")
	t.Cleanup(func() { _ = os.Setenv(EnvBackendURL, old) })
	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got, want := cfg.Backend.BaseURL, "https://example.test:8443"; got != want {
		t.Fatalf("Backend.BaseURL = %q, want %q", got, want)
	}
}

func TestEnvOverridesTelemetry(t *testing.T) {
	old := os.Getenv(EnvTelemetryOptIn)
	_ = os.Setenv(EnvTelemetryOptIn, "true")
	t.Cleanup(func() { _ = os.Setenv(EnvTelemetryOptIn, old) })
	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.General.TelemetryOptIn {
		t.Fatalf("General.TelemetryOptIn expected true from env override")
	}
}

func TestMergeIncludesEnableServer(t *testing.T) {
	// Given a file config that sets enable_server, mergeInto should carry it through
	dst := Defaults()
	src := Defaults()
	src.General.EnableServer = true
	mergeInto(&dst, &src)
	if !dst.General.EnableServer {
		t.Fatalf("EnableServer was not merged from file config")
	}
}

func TestMergeIncludesLogging(t *testing.T) {
	dst := Defaults()
	src := Defaults()
	src.Logging.Level = "debug"
	src.Logging.Format = "json"
	src.Logging.Source = true
	src.Logging.File = "C:/tmp/gcw.log"
	mergeInto(&dst, &src)
	if dst.Logging.Level != "debug" || dst.Logging.Format != "json" || !dst.Logging.Source || dst.Logging.File != "C:/tmp/gcw.log" {
		t.Fatalf("logging fields not merged correctly: %#v", dst.Logging)
	}
}

func TestEnvOverridesLogging(t *testing.T) {
	oldLevel := os.Getenv(EnvLogLevel)
	oldFmt := os.Getenv(EnvLogFormat)
	oldSrc := os.Getenv(EnvLogSource)
	oldFile := os.Getenv(EnvLogFile)
	_ = os.Setenv(EnvLogLevel, "error")
	_ = os.Setenv(EnvLogFormat, "json")
	_ = os.Setenv(EnvLogSource, "1")
	_ = os.Setenv(EnvLogFile, "X:/gcw.log")
	t.Cleanup(func() {
		_ = os.Setenv(EnvLogLevel, oldLevel)
		_ = os.Setenv(EnvLogFormat, oldFmt)
		_ = os.Setenv(EnvLogSource, oldSrc)
		_ = os.Setenv(EnvLogFile, oldFile)
	})
	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Logging.Level != "error" || cfg.Logging.Format != "json" || !cfg.Logging.Source || cfg.Logging.File != "X:/gcw.log" {
		t.Fatalf("env overrides not applied to logging: %#v", cfg.Logging)
	}
}

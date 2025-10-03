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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppConfig is the user-editable configuration persisted to a YAML file in the user scope.
// Environment variables are treated as read-only overrides at runtime.
// Minimal schema to start; can evolve with config_version migrations.
//
// config_version: bump when the structure changes in a backward-incompatible way.
// Unknown fields should be preserved when possible (yaml handles this by ignoring extras on unmarshal).

type BackendConfig struct {
	BaseURL     string `yaml:"base_url"`
	TimeoutMs   int    `yaml:"timeout_ms"`
	TLSInsecure bool   `yaml:"tls_insecure"`
	// Token is not stored on disk; it lives in the OS keychain.
}

type GeneralConfig struct {
	TelemetryOptIn bool   `yaml:"telemetry_opt_in"`
	Theme          string `yaml:"theme"` // "system" | "light" | "dark" (informational for now)
	EnableServer   bool   `yaml:"enable_server"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Source bool   `yaml:"source"`
	File   string `yaml:"file"`
}

type AppConfig struct {
	ConfigVersion int           `yaml:"config_version"`
	General       GeneralConfig `yaml:"general"`
	Backend       BackendConfig `yaml:"backend"`
	Logging       LoggingConfig `yaml:"logging"`
}

// Defaults returns the application defaults.
func Defaults() AppConfig {
	return AppConfig{
		ConfigVersion: 1,
		General:       GeneralConfig{TelemetryOptIn: false, Theme: "system", EnableServer: false},
		Backend:       BackendConfig{BaseURL: "http://localhost:8080", TimeoutMs: 15000, TLSInsecure: false},
		Logging:       LoggingConfig{Level: "info", Format: "console", Source: false, File: ""},
	}
}

// Env var names used as overrides.
const (
	EnvBackendURL       = "GCW_BACKEND_URL"
	EnvBackendTimeoutMs = "GCW_BACKEND_TIMEOUT_MS"
	EnvBackendTLSInsec  = "GCW_TLS_INSECURE"
	EnvTelemetryOptIn   = "GCW_TELEMETRY_OPT_IN"
	EnvEnableServer     = "GCW_ENABLE_SERVER"
	// EnvLogLevel Logging envs
	EnvLogLevel  = "GCW_LOG_LEVEL"
	EnvLogFormat = "GCW_LOG_FORMAT"
	EnvLogSource = "GCW_LOG_SOURCE"
	EnvLogFile   = "GCW_LOG_FILE"
)

// Service/keys for OS keyring.
const (
	keyringService = "GoComicWriter"
	keyringToken   = "backend_token"
)

// tokenStore abstracts keyring, so we can stub in tests.
var tokenStore TokenStore = &osKeyring{}

type TokenStore interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

// osKeyring implements TokenStore using the OS keyring via github.com/zalando/go-keyring.
type osKeyring struct{}

func (k *osKeyring) Get(service, key string) (string, error) {
	kr, err := getKeyring()
	if err != nil {
		return "", err
	}
	return kr.get(service, key)
}
func (k *osKeyring) Set(service, key, value string) error {
	kr, err := getKeyring()
	if err != nil {
		return err
	}
	return kr.set(service, key, value)
}
func (k *osKeyring) Delete(service, key string) error {
	kr, err := getKeyring()
	if err != nil {
		return err
	}
	return kr.delete(service, key)
}

// indirection to avoid hard importing in non-using contexts
type keyringShim interface {
	get(service, key string) (string, error)
	set(service, key, value string) error
	delete(service, key string) error
}

func getKeyring() (keyringShim, error) {
	return &goKeyringShim{}, nil
}

type goKeyringShim struct{}

func (g *goKeyringShim) get(service, key string) (string, error) {
	return keyringGet(service, key)
}
func (g *goKeyringShim) set(service, key, value string) error {
	return keyringSet(service, key, value)
}
func (g *goKeyringShim) delete(service, key string) error {
	return keyringDelete(service, key)
}

// The following vars are defined in keyring_stub.go or keyring_real.go depending on build tags.
var (
	keyringGet    func(service, key string) (string, error)
	keyringSet    func(service, key, value string) error
	keyringDelete func(service, key string) error
)

// ConfigPath returns the per-user config file path.
func ConfigPath() (string, error) {
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("AppData")
		if base == "" { // fallback
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		base = filepath.Join(base, "GoComicWriter")
	case "darwin":
		base = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "GoComicWriter")
	default: // linux and others
		base = filepath.Join(os.Getenv("HOME"), ".config", "gocomicwriter")
	}
	if base == "" {
		return "", errors.New("cannot resolve config directory")
	}
	return filepath.Join(base, "config.yaml"), nil
}

// Load reads user config file (if present), applies defaults, and merges environment overrides.
// It also loads the backend token from keyring (not kept inside the struct; returned separately).
func Load() (AppConfig, string, error) {
	cfg := Defaults()
	path, err := ConfigPath()
	if err != nil {
		return cfg, "", err
	}
	if data, err := os.ReadFile(path); err == nil {
		var fileCfg AppConfig
		if err := yaml.Unmarshal(data, &fileCfg); err == nil {
			mergeInto(&cfg, &fileCfg)
		}
	}
	applyEnvOverrides(&cfg)
	// token from keyring
	tok, _ := tokenStore.Get(keyringService, keyringToken)
	return cfg, tok, nil
}

// Save writes the user config YAML and persists the token into OS keyring (if non-empty).
func Save(cfg AppConfig, token string) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	if token != "" {
		if err := tokenStore.Set(keyringService, keyringToken, token); err != nil {
			return err
		}
	}
	return nil
}

func mergeInto(dst *AppConfig, src *AppConfig) {
	if src.ConfigVersion != 0 {
		dst.ConfigVersion = src.ConfigVersion
	}
	if src.General.Theme != "" {
		dst.General.Theme = src.General.Theme
	}
	// booleans: copy directly from src (file) so user preferences persist
	dst.General.TelemetryOptIn = src.General.TelemetryOptIn
	dst.General.EnableServer = src.General.EnableServer
	if src.Backend.BaseURL != "" {
		dst.Backend.BaseURL = src.Backend.BaseURL
	}
	if src.Backend.TimeoutMs != 0 {
		dst.Backend.TimeoutMs = src.Backend.TimeoutMs
	}
	dst.Backend.TLSInsecure = src.Backend.TLSInsecure
	// logging
	if strings.TrimSpace(src.Logging.Level) != "" {
		dst.Logging.Level = strings.ToLower(strings.TrimSpace(src.Logging.Level))
	}
	if strings.TrimSpace(src.Logging.Format) != "" {
		dst.Logging.Format = strings.ToLower(strings.TrimSpace(src.Logging.Format))
	}
	dst.Logging.Source = src.Logging.Source
	if strings.TrimSpace(src.Logging.File) != "" {
		dst.Logging.File = strings.TrimSpace(src.Logging.File)
	}
}

func applyEnvOverrides(cfg *AppConfig) {
	if v := strings.TrimSpace(os.Getenv(EnvBackendURL)); v != "" {
		cfg.Backend.BaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvBackendTimeoutMs)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Backend.TimeoutMs = n
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvBackendTLSInsec)); v != "" {
		lv := strings.ToLower(v)
		cfg.Backend.TLSInsecure = lv == "1" || lv == "true" || lv == "on" || lv == "yes"
	}
	if v := strings.TrimSpace(os.Getenv(EnvTelemetryOptIn)); v != "" {
		lv := strings.ToLower(v)
		cfg.General.TelemetryOptIn = lv == "1" || lv == "true" || lv == "on" || lv == "yes"
	}
	if v := strings.TrimSpace(os.Getenv(EnvEnableServer)); v != "" {
		lv := strings.ToLower(v)
		cfg.General.EnableServer = lv == "1" || lv == "true" || lv == "on" || lv == "yes"
	}
	// logging overrides
	if v := strings.TrimSpace(os.Getenv(EnvLogLevel)); v != "" {
		cfg.Logging.Level = strings.ToLower(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvLogFormat)); v != "" {
		cfg.Logging.Format = strings.ToLower(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvLogSource)); v != "" {
		lv := strings.ToLower(v)
		cfg.Logging.Source = lv == "1" || lv == "true" || lv == "on" || lv == "yes"
	}
	if v := strings.TrimSpace(os.Getenv(EnvLogFile)); v != "" {
		cfg.Logging.File = v
	}
}

// EnvOverrideFor returns the env var name if the field is overridden by environment variables.
func EnvOverrideFor(key string) (string, bool) {
	switch key {
	case "backend.base_url":
		if os.Getenv(EnvBackendURL) != "" {
			return EnvBackendURL, true
		}
	case "backend.timeout_ms":
		if os.Getenv(EnvBackendTimeoutMs) != "" {
			return EnvBackendTimeoutMs, true
		}
	case "backend.tls_insecure":
		if os.Getenv(EnvBackendTLSInsec) != "" {
			return EnvBackendTLSInsec, true
		}
	case "general.telemetry_opt_in":
		if os.Getenv(EnvTelemetryOptIn) != "" {
			return EnvTelemetryOptIn, true
		}
	case "general.enable_server":
		if os.Getenv(EnvEnableServer) != "" {
			return EnvEnableServer, true
		}
	case "logging.level":
		if os.Getenv(EnvLogLevel) != "" {
			return EnvLogLevel, true
		}
	case "logging.format":
		if os.Getenv(EnvLogFormat) != "" {
			return EnvLogFormat, true
		}
	case "logging.source":
		if os.Getenv(EnvLogSource) != "" {
			return EnvLogSource, true
		}
	case "logging.file":
		if os.Getenv(EnvLogFile) != "" {
			return EnvLogFile, true
		}
	}
	return "", false
}

// EffectiveTimeout returns the backend timeout as a duration-like milliseconds string for http.Client.
func (b BackendConfig) EffectiveTimeout() string {
	if b.TimeoutMs <= 0 {
		return fmt.Sprintf("%dms", Defaults().Backend.TimeoutMs)
	}
	return fmt.Sprintf("%dms", b.TimeoutMs)
}

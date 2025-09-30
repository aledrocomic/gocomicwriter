/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package backend

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds server configuration.
type Config struct {
	DBURL string
	Addr  string // http bind address, e.g., ":8080"
}

func loadConfig() Config {
	cfg := Config{
		DBURL: os.Getenv("DATABASE_URL"),
		Addr:  ":8080",
	}
	if v := os.Getenv("GCW_PG_DSN"); v != "" {
		cfg.DBURL = v
	}
	if v := os.Getenv("PORT"); v != "" {
		cfg.Addr = ":" + v
	}
	if v := os.Getenv("ADDR"); v != "" {
		cfg.Addr = v
	}
	if cfg.DBURL == "" {
		// Reasonable local default; requires a DB set up by the developer
		cfg.DBURL = "postgres://postgres:postgres@localhost:5432/gocomicwriter?sslmode=disable"
	}
	return cfg
}

// Start runs the minimal HTTP server and applies DB migrations at startup.
func Start() error {
	cfg := loadConfig()

	db, err := sql.Open("pgx", cfg.DBURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("db close: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if err := applyMigrations(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	mux := http.NewServeMux()
	// Health endpoints
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("db not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		ver := getVersion()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ver))
	})

	// Auth secret (dev-friendly default)
	secret := os.Getenv("GCW_AUTH_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
		log.Printf("WARN: GCW_AUTH_SECRET not set; using insecure dev secret")
	}

	// POST /api/auth/token → { token, expires_at }
	mux.HandleFunc("/api/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Optional JSON body: { "subject": "name", "ttl_seconds": 3600 }
		var req struct {
			Subject    string `json:"subject"`
			TTLSeconds int64  `json:"ttl_seconds"`
		}
		b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = r.Body.Close()
		_ = json.Unmarshal(b, &req)
		if req.Subject == "" {
			req.Subject = "dev"
		}
		if req.TTLSeconds <= 0 || req.TTLSeconds > 24*3600 {
			req.TTLSeconds = 3600
		}
		exp := time.Now().Add(time.Duration(req.TTLSeconds) * time.Second)
		tok, err := signToken(secret, req.Subject, exp)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"token":      tok,
			"expires_at": exp.UTC().Format(time.RFC3339),
		})
	})

	// GET /api/projects (auth required)
	mux.HandleFunc("/api/projects", withAuth(secret, func(w http.ResponseWriter, r *http.Request, sub string) {
		rows, err := db.QueryContext(r.Context(), `SELECT id, stable_id, name, updated_at, version FROM projects ORDER BY updated_at DESC`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()
		type proj struct {
			ID        int64     `json:"id"`
			StableID  string    `json:"stable_id"`
			Name      string    `json:"name"`
			UpdatedAt time.Time `json:"updated_at"`
			Version   int64     `json:"version"`
		}
		var list []proj
		for rows.Next() {
			var p proj
			if err := rows.Scan(&p.ID, &p.StableID, &p.Name, &p.UpdatedAt, &p.Version); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			list = append(list, p)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}))

	// GET /api/projects/{id}/index (auth required)
	mux.HandleFunc("/api/projects/", withAuth(secret, func(w http.ResponseWriter, r *http.Request, sub string) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Expect path: /api/projects/{id}/index
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 4 || parts[0] != "api" || parts[1] != "projects" || parts[3] != "index" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		pid, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid project id"))
			return
		}
		var (
			version int64
			snap    []byte
			created time.Time
		)
		row := db.QueryRowContext(r.Context(), `SELECT version, snapshot, created_at FROM index_snapshots WHERE project_id = $1 ORDER BY version DESC, id DESC LIMIT 1`, pid)
		switch err := row.Scan(&version, &snap, &created); err {
		case sql.ErrNoRows:
			writeError(w, http.StatusNotFound, fmt.Errorf("no snapshot"))
			return
		case nil:
			// ok
		default:
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		// snapshot stored as JSONB; deliver it back as JSON inside envelope
		var raw any
		if err := json.Unmarshal(snap, &raw); err != nil {
			// If not valid JSON, return raw string
			raw = json.RawMessage(snap)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"project_id": pid,
			"version":    version,
			"created_at": created.UTC().Format(time.RFC3339),
			"snapshot":   raw,
		})
	}))

	// Placeholders for future endpoints
	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/deltas") || strings.HasSuffix(r.URL.Path, "/comments") {
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte("not implemented yet"))
			return
		}
	})

	log.Printf("gcwserver listening on %s", cfg.Addr)
	return http.ListenAndServe(cfg.Addr, mux)
}

func getVersion() string {
	// Avoid importing if package path changes; fall back to env or default
	if v := os.Getenv("GCW_VERSION"); v != "" {
		return v
	}
	return "gcwserver dev"
}

// applyMigrations applies embedded SQL migrations in filename order.
func applyMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	// ensure table exists for explicit versioning as well
	// dialect=PostreSQL
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	applied := map[int64]bool{}
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("select schema_migrations: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("rows close: %v", err)
		}
	}()
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, fname := range files {
		version, err := parseVersion(fname)
		if err != nil {
			return err
		}
		if applied[version] {
			continue
		}
		b, err := migrationsFS.ReadFile(path.Join("migrations", fname))
		if err != nil {
			return err
		}
		sqlText := string(b)
		if strings.TrimSpace(sqlText) == "" {
			continue
		}
		log.Printf("applying migration %s", fname)
		if _, err := db.ExecContext(ctx, sqlText); err != nil {
			return fmt.Errorf("apply %s: %w", fname, err)
		}
	}
	return nil
}

func parseVersion(name string) (int64, error) {
	base := path.Base(name)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 0 {
		return 0, errors.New("invalid migration filename: " + name)
	}
	v, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse version from %s: %w", name, err)
	}
	return v, nil
}

// --- Helpers: auth and JSON ---

type tokenClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"` // unix seconds
}

func signToken(secret, subject string, exp time.Time) (string, error) {
	claims := tokenClaims{Sub: subject, Exp: exp.Unix()}
	b, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write(b)
	sig := h.Sum(nil)
	payload := base64.RawURLEncoding.EncodeToString(b)
	signature := base64.RawURLEncoding.EncodeToString(sig)
	return payload + "." + signature, nil
}

func verifyToken(secret, token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}
	payloadB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid token payload")
	}
	sigB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid token signature")
	}
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write(payloadB)
	expected := h.Sum(nil)
	if !hmac.Equal(expected, sigB) {
		return "", fmt.Errorf("bad signature")
	}
	var claims tokenClaims
	if err := json.Unmarshal(payloadB, &claims); err != nil {
		return "", fmt.Errorf("bad claims")
	}
	if claims.Exp < time.Now().Unix() {
		return "", fmt.Errorf("token expired")
	}
	if claims.Sub == "" {
		claims.Sub = "dev"
	}
	return claims.Sub, nil
}

func withAuth(secret string, next func(w http.ResponseWriter, r *http.Request, subject string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(strings.ToLower(auth), strings.ToLower(prefix)) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("missing bearer token"))
			return
		}
		token := strings.TrimSpace(auth[len(prefix):])
		sub, err := verifyToken(secret, token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid token"))
			return
		}
		next(w, r, sub)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

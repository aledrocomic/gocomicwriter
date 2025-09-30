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
	"database/sql"
	"embed"
	"errors"
	"fmt"
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
		// Use internal/version package if available
		ver := getVersion()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ver))
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

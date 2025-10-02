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
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"gocomicwriter/internal/storage"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds server configuration.
type Config struct {
	DBURL           string
	Addr            string // http bind address, e.g., ":8080"
	TLSEnable       bool
	TLSCertFile     string
	TLSKeyFile      string
	AuthMode        string // dev | static
	AdminAPIKey     string
	ObjectHealthURL string // e.g., http://minio:9000/minio/health/ready
	ObjectHealthReq bool   // if true, failing object health makes readyz fail
}

func getenvBool(name string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	}
	return def
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
	// TLS/env auth/object config
	cfg.TLSEnable = getenvBool("GCW_TLS_ENABLE", false)
	cfg.TLSCertFile = os.Getenv("GCW_TLS_CERT_FILE")
	cfg.TLSKeyFile = os.Getenv("GCW_TLS_KEY_FILE")
	cfg.AuthMode = strings.ToLower(strings.TrimSpace(os.Getenv("GCW_AUTH_MODE")))
	if cfg.AuthMode == "" {
		cfg.AuthMode = "dev"
	}
	cfg.AdminAPIKey = os.Getenv("GCW_ADMIN_API_KEY")
	cfg.ObjectHealthURL = os.Getenv("GCW_OBJECT_HEALTH_URL")
	if cfg.ObjectHealthURL == "" {
		if ep := os.Getenv("GCW_MINIO_ENDPOINT"); ep != "" {
			// assume http unless scheme provided
			if strings.HasPrefix(ep, "http://") || strings.HasPrefix(ep, "https://") {
				cfg.ObjectHealthURL = strings.TrimRight(ep, "/") + "/minio/health/ready"
			} else {
				cfg.ObjectHealthURL = "http://" + strings.TrimRight(ep, "/") + "/minio/health/ready"
			}
		}
	}
	cfg.ObjectHealthReq = getenvBool("GCW_OBJECT_HEALTH_REQUIRED", false)

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
		if isInvalidCatalog(err) {
			if err2 := tryCreateMissingDatabase(ctx, cfg.DBURL); err2 != nil {
				return fmt.Errorf("ping db: %w; additionally failed to create database: %v", err, err2)
			}
			// Retry ping after creating the database
			if err3 := db.PingContext(ctx); err3 != nil {
				return fmt.Errorf("ping db after create: %w", err3)
			}
		} else {
			return fmt.Errorf("ping db: %w", err)
		}
	}

	if err := applyMigrations(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	mux := http.NewServeMux()
	// Health endpoints
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": getVersion(),
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		status := "ready"
		dbOK := true
		objOK := true
		if err := db.PingContext(ctx); err != nil {
			dbOK = false
			status = "degraded"
		}
		if cfg.ObjectHealthURL != "" {
			client := &http.Client{Timeout: 2 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ObjectHealthURL, nil)
			if err != nil {
				objOK = false
				if cfg.ObjectHealthReq {
					status = "not_ready"
				}
			} else {
				resp, err := client.Do(req)
				if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
					objOK = false
					if cfg.ObjectHealthReq {
						status = "not_ready"
					}
				}
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}
		}
		code := http.StatusOK
		if !dbOK || (cfg.ObjectHealthReq && !objOK) {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, map[string]any{
			"status":  status,
			"db":      dbOK,
			"objects": objOK,
			"version": getVersion(),
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
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
		// JSON body: { "email": "user@example.com", "display_name": "User", "subject": "alias", "ttl_seconds": 3600 }
		var req struct {
			Subject     string `json:"subject"`
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
			TTLSeconds  int64  `json:"ttl_seconds"`
		}
		b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		_ = r.Body.Close()
		if err := json.Unmarshal(b, &req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json"))
			return
		}
		sub := strings.TrimSpace(req.Email)
		if sub == "" {
			sub = strings.TrimSpace(req.Subject)
		}
		if req.TTLSeconds <= 0 || req.TTLSeconds > 24*3600 {
			req.TTLSeconds = 3600
		}
		if cfg.AuthMode == "static" {
			if cfg.AdminAPIKey == "" || r.Header.Get("X-API-Key") != cfg.AdminAPIKey {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			if sub == "" {
				writeError(w, http.StatusBadRequest, fmt.Errorf("subject/email required"))
				return
			}
			// ensure user exists or upsert display_name
			if _, err := db.ExecContext(r.Context(), `INSERT INTO users(email, display_name) VALUES ($1, NULLIF($2,'') )
				ON CONFLICT (email) DO UPDATE SET display_name = COALESCE(EXCLUDED.display_name, users.display_name)`, sub, req.DisplayName); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if sub == "" {
			sub = "dev"
		}
		exp := time.Now().Add(time.Duration(req.TTLSeconds) * time.Second)
		tok, err := signToken(secret, sub, exp)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"token":      tok,
			"subject":    sub,
			"expires_at": exp.UTC().Format(time.RFC3339),
		})
	})

	// Auth wrapper verifying token and (in static mode) user existence
	authWrap := func(next func(w http.ResponseWriter, r *http.Request, sub string)) http.HandlerFunc {
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
			if cfg.AuthMode == "static" {
				var x int
				if err := db.QueryRowContext(r.Context(), `SELECT 1 FROM users WHERE email = $1`, sub).Scan(&x); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte("user not allowed"))
						return
					}
					writeError(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				// In dev mode, auto-provision user on first request
				if _, err := db.ExecContext(r.Context(), `INSERT INTO users(email) VALUES ($1) ON CONFLICT (email) DO NOTHING`, sub); err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
			}
			next(w, r, sub)
		}
	}
	// GET /api/projects (auth required)
	mux.HandleFunc("/api/projects", authWrap(func(w http.ResponseWriter, r *http.Request, sub string) {
		rows, err := db.QueryContext(r.Context(), `SELECT p.id, p.stable_id, p.name, p.updated_at, p.version
		FROM projects p
		JOIN project_members pm ON pm.project_id = p.id
		JOIN users u ON u.id = pm.user_id
		WHERE u.email = $1
		ORDER BY p.updated_at DESC`, sub)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = rows.Close() }()
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

	// Project-scoped endpoints (auth required): index snapshot, sync push/pull
	mux.HandleFunc("/api/projects/", authWrap(func(w http.ResponseWriter, r *http.Request, sub string) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 4 || parts[0] != "api" || parts[1] != "projects" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		pid, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid project id"))
			return
		}
		// Enforce membership: user must be a member of this project
		{
			var x int
			if err := db.QueryRowContext(r.Context(), `SELECT 1
				FROM project_members pm
				JOIN users u ON u.id = pm.user_id
				WHERE u.email = $1 AND pm.project_id = $2`, sub, pid).Scan(&x); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("forbidden"))
					return
				}
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
		// /api/projects/{id}/index (GET)
		if len(parts) == 4 && parts[3] == "index" {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
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
			var raw any
			if err := json.Unmarshal(snap, &raw); err != nil {
				raw = json.RawMessage(snap)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"project_id": pid,
				"version":    version,
				"created_at": created.UTC().Format(time.RFC3339),
				"snapshot":   raw,
			})
			return
		}
		// /api/projects/{id}/search (GET)
		if len(parts) == 4 && parts[3] == "search" {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			// Parse query params into storage.SearchQuery
			typList := []string{}
			if v := r.URL.Query().Get("types"); v != "" {
				for _, s := range strings.Split(v, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						typList = append(typList, s)
					}
				}
			}
			tagList := []string{}
			if v := r.URL.Query().Get("tags"); v != "" {
				for _, s := range strings.Split(v, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						tagList = append(tagList, s)
					}
				}
			}
			q := storage.SearchQuery{
				Text:      r.URL.Query().Get("text"),
				Character: r.URL.Query().Get("character"),
				Scene:     r.URL.Query().Get("scene"),
				Tags:      tagList,
				Types:     typList,
			}
			if v := r.URL.Query().Get("page_from"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					q.PageFrom = n
				}
			}
			if v := r.URL.Query().Get("page_to"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					q.PageTo = n
				}
			}
			if v := r.URL.Query().Get("limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					q.Limit = n
				}
			}
			if v := r.URL.Query().Get("offset"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					q.Offset = n
				}
			}
			res, err := SearchPG(r.Context(), db, pid, q)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, res)
			return
		}
		// /api/projects/{id}/sync/push (POST) and /sync/pull (GET)
		if len(parts) == 5 && parts[3] == "sync" {
			switch parts[4] {
			case "push":
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				var req struct {
					ClientVersion int64 `json:"client_version"`
					Ops           []struct {
						OpID       string          `json:"op_id"`
						OpType     string          `json:"op_type"`
						EntityType string          `json:"entity_type"`
						EntityID   string          `json:"entity_id"`
						Payload    json.RawMessage `json:"payload"`
					} `json:"ops"`
				}
				b, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
				if err != nil {
					writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
					return
				}
				_ = r.Body.Close()
				if err := json.Unmarshal(b, &req); err != nil {
					writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %v", err))
					return
				}
				ctx := r.Context()
				tx, err := db.BeginTx(ctx, &sql.TxOptions{})
				if err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				defer func() { _ = tx.Rollback() }()
				var curVersion int64
				row := tx.QueryRowContext(ctx, `SELECT version FROM projects WHERE id = $1 FOR UPDATE`, pid)
				if err := row.Scan(&curVersion); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						writeError(w, http.StatusNotFound, fmt.Errorf("project not found"))
						return
					}
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				newVersion := curVersion
				for i, op := range req.Ops {
					newVersion = curVersion + int64(i+1)
					if op.Payload == nil || len(op.Payload) == 0 {
						op.Payload = json.RawMessage("{}")
					}
					if _, err := tx.ExecContext(ctx, `INSERT INTO sync_ops (op_id, project_id, version, actor, op_type, entity_type, entity_id, payload) VALUES (COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()),$2,$3,$4,$5,$6,$7,$8)`,
						op.OpID, pid, newVersion, sub, op.OpType, op.EntityType, op.EntityID, op.Payload); err != nil {
						writeError(w, http.StatusInternalServerError, err)
						return
					}
				}
				if _, err := tx.ExecContext(ctx, `UPDATE projects SET version = $1 WHERE id = $2`, newVersion, pid); err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				if err := tx.Commit(); err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"project_id":     pid,
					"server_version": newVersion,
					"accepted":       len(req.Ops),
				})
				return
			case "pull":
				if r.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				q := r.URL.Query()
				sinceStr := q.Get("since")
				var since int64
				if sinceStr != "" {
					var err error
					since, err = strconv.ParseInt(sinceStr, 10, 64)
					if err != nil || since < 0 {
						writeError(w, http.StatusBadRequest, fmt.Errorf("invalid since"))
						return
					}
				}
				limit := 500
				if ls := q.Get("limit"); ls != "" {
					if v, err := strconv.Atoi(ls); err == nil && v > 0 {
						if v > 5000 {
							v = 5000
						}
						limit = v
					}
				}
				ctx := r.Context()
				var serverVersion int64
				if err := db.QueryRowContext(ctx, `SELECT version FROM projects WHERE id = $1`, pid).Scan(&serverVersion); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						writeError(w, http.StatusNotFound, fmt.Errorf("project not found"))
						return
					}
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				rows, err := db.QueryContext(ctx, `SELECT op_id, version, actor, op_type, entity_type, entity_id, payload, created_at FROM sync_ops WHERE project_id = $1 AND version > $2 ORDER BY version ASC LIMIT $3`, pid, since, limit)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				defer func() {
					if cerr := rows.Close(); cerr != nil {
						log.Printf("error closing rows: %v", cerr)
					}
				}()
				type op struct {
					OpID       string          `json:"op_id"`
					Version    int64           `json:"version"`
					Actor      string          `json:"actor"`
					OpType     string          `json:"op_type"`
					EntityType string          `json:"entity_type"`
					EntityID   string          `json:"entity_id"`
					Payload    json.RawMessage `json:"payload"`
					CreatedAt  time.Time       `json:"created_at"`
				}
				var ops []op
				for rows.Next() {
					var o op
					if err := rows.Scan(&o.OpID, &o.Version, &o.Actor, &o.OpType, &o.EntityType, &o.EntityID, &o.Payload, &o.CreatedAt); err != nil {
						writeError(w, http.StatusInternalServerError, err)
						return
					}
					ops = append(ops, o)
				}
				if err := rows.Err(); err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"project_id":     pid,
					"server_version": serverVersion,
					"ops":            ops,
				})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))

	// Admin: grant membership endpoint (ensures user exists and grants role on project)
	mux.HandleFunc("/api/admin/membership/grant", authWrap(func(w http.ResponseWriter, r *http.Request, sub string) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// In static mode, require admin API key
		if cfg.AuthMode == "static" {
			if cfg.AdminAPIKey == "" || r.Header.Get("X-API-Key") != cfg.AdminAPIKey {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
		}
		var req struct {
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
			ProjectID   int64  `json:"project_id"`
			ProjectSlug string `json:"project_slug"`
		}
		b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		_ = r.Body.Close()
		if err := json.Unmarshal(b, &req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json"))
			return
		}
		email := strings.TrimSpace(req.Email)
		if email == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("email required"))
			return
		}
		role := strings.TrimSpace(req.Role)
		if role == "" {
			role = "owner"
		}
		// Resolve project id
		pid := req.ProjectID
		if pid == 0 {
			slug := strings.TrimSpace(req.ProjectSlug)
			if slug == "" {
				writeError(w, http.StatusBadRequest, fmt.Errorf("project_id or project_slug required"))
				return
			}
			if err := db.QueryRowContext(r.Context(), `SELECT id FROM projects WHERE slug = $1`, slug).Scan(&pid); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, fmt.Errorf("project not found"))
					return
				}
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
		// Ensure user exists or update display_name
		if _, err := db.ExecContext(r.Context(), `INSERT INTO users(email, display_name) VALUES ($1, NULLIF($2,'') )
			ON CONFLICT (email) DO UPDATE SET display_name = COALESCE(EXCLUDED.display_name, users.display_name)`, email, req.DisplayName); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		// Insert or update membership
		if _, err := db.ExecContext(r.Context(), `INSERT INTO project_members(user_id, project_id, role)
			SELECT u.id, $2, $3 FROM users u WHERE u.email = $1
			ON CONFLICT (user_id, project_id) DO UPDATE SET role = EXCLUDED.role`, email, pid, role); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"project_id": pid,
			"user":       email,
			"role":       role,
			"granted_by": sub,
			"status":     "granted",
		})
	}))

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}
	log.Printf("gcwserver listening on %s tls=%v mode=%s", cfg.Addr, cfg.TLSEnable, cfg.AuthMode)
	if cfg.TLSEnable {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			return fmt.Errorf("TLS enabled but GCW_TLS_CERT_FILE or GCW_TLS_KEY_FILE not set")
		}
		return server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	}
	return server.ListenAndServe()
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

// isInvalidCatalog returns true if the error indicates the target database does not exist (SQLSTATE 3D000).
func isInvalidCatalog(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "3D000" { // invalid_catalog_name
			return true
		}
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "sqlstate 3d000") || (strings.Contains(s, "database") && strings.Contains(s, "does not exist"))
}

// tryCreateMissingDatabase connects to the maintenance database and creates the target DB if it does not exist.
func tryCreateMissingDatabase(ctx context.Context, dsn string) error {
	dbname, ok := getDBNameFromDSN(dsn)
	if !ok || dbname == "" {
		return fmt.Errorf("cannot determine database name from DSN")
	}
	adminDSN := setDBNameInDSN(dsn, "postgres")

	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return fmt.Errorf("open admin db: %w", err)
	}
	defer func() { _ = adminDB.Close() }()

	if err := adminDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping admin db: %w", err)
	}

	var one int
	err = adminDB.QueryRowContext(ctx, "SELECT 1 FROM pg_database WHERE datname=$1", dbname).Scan(&one)
	if err == nil {
		// already exists
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check db existence: %w", err)
	}

	qname := `"` + strings.ReplaceAll(dbname, `"`, `""`) + `"`
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+qname); err != nil {
		return fmt.Errorf("create database %s: %w", dbname, err)
	}
	log.Printf("INFO: created database %q", dbname)
	return nil
}

// getDBNameFromDSN extracts the database name from a postgres DSN (URL or key=value form).
func getDBNameFromDSN(dsn string) (string, bool) {
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		name := strings.TrimPrefix(u.Path, "/")
		if name != "" {
			if i := strings.Index(name, "/"); i >= 0 {
				name = name[:i]
			}
			if name != "" {
				return name, true
			}
		}
		q := u.Query().Get("dbname")
		if q != "" {
			return q, true
		}
		return "", false
	}
	// key=value form
	tokens := splitDSNTokens(dsn)
	for _, t := range tokens {
		lt := strings.ToLower(t)
		if strings.HasPrefix(lt, "dbname=") || strings.HasPrefix(lt, "database=") {
			v := t[strings.Index(t, "=")+1:]
			v = strings.Trim(v, "'\"")
			if v != "" {
				return v, true
			}
		}
	}
	return "", false
}

// setDBNameInDSN returns a DSN string with the database name replaced/added.
func setDBNameInDSN(dsn, newDB string) string {
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		u.Path = "/" + newDB
		q := u.Query()
		q.Set("dbname", newDB)
		u.RawQuery = q.Encode()
		return u.String()
	}
	tokens := splitDSNTokens(dsn)
	found := false
	for i, t := range tokens {
		lt := strings.ToLower(t)
		if strings.HasPrefix(lt, "dbname=") || strings.HasPrefix(lt, "database=") {
			k := t[:strings.Index(t, "=")+1]
			tokens[i] = k + newDB
			found = true
		}
	}
	if !found {
		tokens = append(tokens, "dbname="+newDB)
	}
	return strings.Join(tokens, " ")
}

// splitDSNTokens splits a key=value DSN into tokens, respecting quotes around values.
func splitDSNTokens(s string) []string {
	var tokens []string
	var buf strings.Builder
	inQuote := false
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' || c == '"' {
			if inQuote && c == quote {
				inQuote = false
			} else if !inQuote {
				inQuote = true
				quote = c
			}
			buf.WriteByte(c)
			continue
		}
		if c == ' ' && !inQuote {
			if buf.Len() > 0 {
				tokens = append(tokens, buf.String())
				buf.Reset()
			}
			continue
		}
		buf.WriteByte(c)
	}
	if buf.Len() > 0 {
		tokens = append(tokens, buf.String())
	}
	return tokens
}

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gocomicwriter/internal/storage"
)

// Client is a minimal HTTP client for the thin backend API.
// It supports read-only operations used by the desktop app under a feature flag.
type Client struct {
	BaseURL     string
	Token       string // bearer token
	AdminAPIKey string // optional admin key for static mode admin endpoints
	client      *http.Client
}

// NewClient creates a new backend client. baseURL may include a trailing slash; it will be normalized.
func NewClient(baseURL string, token string) *Client {
	b := strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: b,
		Token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, dest any) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if c.AdminAPIKey != "" {
		req.Header.Set("X-API-Key", c.AdminAPIKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server %s %s: %s", method, u.Path, resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	return dec.Decode(dest)
}

func (c *Client) doJSONWithBody(ctx context.Context, method, path string, body any, dest any) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if body != nil {
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if c.AdminAPIKey != "" {
		req.Header.Set("X-API-Key", c.AdminAPIKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server %s %s: %s", method, u.Path, resp.Status)
	}
	if dest == nil {
		return nil
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	return dec.Decode(dest)
}

// Project is a minimal projection for listing.
type Project struct {
	ID        int64     `json:"id"`
	StableID  string    `json:"stable_id"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// ListProjects returns available projects (read-only).
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var list []Project
	if err := c.doJSON(ctx, http.MethodGet, "/api/projects", &list); err != nil {
		return nil, err
	}
	return list, nil
}

// IndexSnapshotEnvelope matches the server response for latest index snapshot of a project.
type IndexSnapshotEnvelope struct {
	ProjectID int64       `json:"project_id"`
	Version   int64       `json:"version"`
	CreatedAt string      `json:"created_at"`
	Snapshot  interface{} `json:"snapshot"`
}

// GetIndexSnapshot fetches the latest index snapshot for a project.
func (c *Client) GetIndexSnapshot(ctx context.Context, projectID int64) (*IndexSnapshotEnvelope, error) {
	var env IndexSnapshotEnvelope
	path := fmt.Sprintf("/api/projects/%d/index", projectID)
	if err := c.doJSON(ctx, http.MethodGet, path, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Sync types
type SyncOp struct {
	OpID       string          `json:"op_id"`
	Version    int64           `json:"version"`
	Actor      string          `json:"actor"`
	OpType     string          `json:"op_type"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
}

type SyncOpInput struct {
	OpID       string          `json:"op_id,omitempty"`
	OpType     string          `json:"op_type"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type PushResult struct {
	ProjectID     int64 `json:"project_id"`
	ServerVersion int64 `json:"server_version"`
	Accepted      int   `json:"accepted"`
}

type PullResult struct {
	ProjectID     int64    `json:"project_id"`
	ServerVersion int64    `json:"server_version"`
	Ops           []SyncOp `json:"ops"`
}

// PushOps pushes a batch of ops to the server (no conflict resolution).
func (c *Client) PushOps(ctx context.Context, projectID int64, clientVersion int64, ops []SyncOpInput) (*PushResult, error) {
	req := struct {
		ClientVersion int64         `json:"client_version"`
		Ops           []SyncOpInput `json:"ops"`
	}{ClientVersion: clientVersion, Ops: ops}
	var res PushResult
	path := fmt.Sprintf("/api/projects/%d/sync/push", projectID)
	if err := c.doJSONWithBody(ctx, http.MethodPost, path, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// PullOps pulls ops since a given version.
func (c *Client) PullOps(ctx context.Context, projectID int64, since int64, limit int) (*PullResult, error) {
	if limit <= 0 {
		limit = 500
	}
	path := fmt.Sprintf("/api/projects/%d/sync/pull?since=%d&limit=%d", projectID, since, limit)
	var res PullResult
	if err := c.doJSON(ctx, http.MethodGet, path, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Search issues a search request to the backend for a given project using parameters compatible
// with storage.SearchQuery and returns a slice of storage.SearchResult.
func (c *Client) Search(ctx context.Context, projectID int64, q storage.SearchQuery) ([]storage.SearchResult, error) {
	values := url.Values{}
	if s := strings.TrimSpace(q.Text); s != "" {
		values.Set("text", s)
	}
	if s := strings.TrimSpace(q.Character); s != "" {
		values.Set("character", s)
	}
	if s := strings.TrimSpace(q.Scene); s != "" {
		values.Set("scene", s)
	}
	if len(q.Types) > 0 {
		values.Set("types", strings.Join(q.Types, ","))
	}
	if len(q.Tags) > 0 {
		values.Set("tags", strings.Join(q.Tags, ","))
	}
	if q.PageFrom > 0 {
		values.Set("page_from", fmt.Sprintf("%d", q.PageFrom))
	}
	if q.PageTo > 0 {
		values.Set("page_to", fmt.Sprintf("%d", q.PageTo))
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	if q.Offset > 0 {
		values.Set("offset", fmt.Sprintf("%d", q.Offset))
	}
	path := fmt.Sprintf("/api/projects/%d/search?%s", projectID, values.Encode())
	var res []storage.SearchResult
	if err := c.doJSON(ctx, http.MethodGet, path, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// HealthStatus represents the /healthz response from the server.
type HealthStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Time    string `json:"time"`
}

// Health pings the server health endpoint.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	var hs HealthStatus
	if err := c.doJSON(ctx, http.MethodGet, "/healthz", &hs); err != nil {
		return nil, err
	}
	return &hs, nil
}

// --- Admin: Grant membership ---

type GrantMembershipRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role,omitempty"`
	ProjectID   int64  `json:"project_id,omitempty"`
	ProjectSlug string `json:"project_slug,omitempty"`
}

type GrantMembershipResponse struct {
	ProjectID int64  `json:"project_id"`
	User      string `json:"user"`
	Role      string `json:"role"`
	GrantedBy string `json:"granted_by"`
	Status    string `json:"status"`
}

// AdminGrantMembership calls the admin endpoint to provision a user (if necessary) and grant a role
// on a project. Requires c.AdminAPIKey to be set when the server runs in static auth mode.
func (c *Client) AdminGrantMembership(ctx context.Context, req GrantMembershipRequest) (*GrantMembershipResponse, error) {
	var res GrantMembershipResponse
	if err := c.doJSONWithBody(ctx, http.MethodPost, "/api/admin/membership/grant", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

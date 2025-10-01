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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a minimal HTTP client for the thin backend API.
// It supports read-only operations used by the desktop app under a feature flag.
type Client struct {
	BaseURL string
	Token   string // bearer token
	client  *http.Client
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
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server %s %s: %s", method, u.Path, resp.Status)
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

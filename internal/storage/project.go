/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gocomicwriter/internal/domain"
	applog "gocomicwriter/internal/log"
	"log/slog"
)

const (
	ManifestFileName = "comic.json"
	BackupsDirName   = "backups"
)

// Standard subfolders as outlined in the concept document.
var standardSubDirs = []string{
	"script",
	"pages",
	"assets",
	"styles",
	"exports",
	BackupsDirName,
}

// ProjectHandle keeps track of the project state loaded/saved from disk.
// It is intentionally simple for early development.
// Root is the project directory containing comic.json and subfolders.
// Project holds the in-memory representation of the manifest.
type ProjectHandle struct {
	Root         string
	ManifestPath string
	Project      domain.Project
}

// InitProject creates a new project directory at root (creating it if it doesn't exist),
// scaffolds the standard subfolders, and writes the given manifest file transactionally.
func InitProject(root string, proj domain.Project) (*ProjectHandle, error) {
	l := applog.WithOperation(applog.WithComponent("storage"), "init").With(slog.String("root", root))
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("root path is required")
	}
	// Ensure directory exists
	if err := os.MkdirAll(root, 0o755); err != nil {
		l.Error("create project root failed", slog.Any("err", err))
		return nil, fmt.Errorf("create project root: %w", err)
	}
	// Create standard subfolders
	for _, d := range standardSubDirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			l.Error("create subdir failed", slog.String("dir", d), slog.Any("err", err))
			return nil, fmt.Errorf("create subdir %s: %w", d, err)
		}
	}

	ph := &ProjectHandle{
		Root:         root,
		ManifestPath: filepath.Join(root, ManifestFileName),
		Project:      proj,
	}
	if err := Save(ph); err != nil {
		l.Error("initial save failed", slog.Any("err", err))
		return nil, err
	}
	l.Info("project created", slog.String("manifest", ph.ManifestPath))
	return ph, nil
}

// Open loads an existing project from the given root directory.
// If the current manifest cannot be read or parsed, it will attempt last backup.
func Open(root string) (*ProjectHandle, error) {
	l := applog.WithOperation(applog.WithComponent("storage"), "open").With(slog.String("root", root))
	mpath := filepath.Join(root, ManifestFileName)
	b, err := os.ReadFile(mpath)
	if err != nil {
		l.Warn("open manifest failed, trying backup", slog.Any("err", err))
		proj, berr := openFromLatestBackup(root)
		if berr != nil {
			l.Error("backup open failed", slog.Any("err", berr))
			return nil, fmt.Errorf("open manifest: %w; backup attempt: %v", err, berr)
		}
		l.Info("opened from backup", slog.String("manifest", mpath))
		return &ProjectHandle{Root: root, ManifestPath: mpath, Project: *proj}, nil
	}
	var p domain.Project
	if uerr := json.Unmarshal(b, &p); uerr != nil {
		l.Warn("parse manifest failed, trying backup", slog.Any("err", uerr))
		proj, berr := openFromLatestBackup(root)
		if berr != nil {
			l.Error("backup open failed", slog.Any("err", berr))
			return nil, fmt.Errorf("parse manifest: %w; backup attempt: %v", uerr, berr)
		}
		l.Info("opened from backup", slog.String("manifest", mpath))
		return &ProjectHandle{Root: root, ManifestPath: mpath, Project: *proj}, nil
	}
	l.Info("project opened", slog.String("manifest", mpath), slog.String("name", p.Name))
	return &ProjectHandle{Root: root, ManifestPath: mpath, Project: p}, nil
}

// Save writes the current ProjectHandle.Project to disk with transactional semantics
// and a timestamped backup of the previous manifest (if present).
func Save(ph *ProjectHandle) error {
	l := applog.WithOperation(applog.WithComponent("storage"), "save")
	if ph == nil {
		return errors.New("nil ProjectHandle")
	}
	if ph.Root == "" || ph.ManifestPath == "" {
		return errors.New("invalid ProjectHandle: missing paths")
	}
	l.Info("saving manifest", slog.String("path", ph.ManifestPath))
	// Marshal in human-readable form
	data, err := json.MarshalIndent(ph.Project, "", "  ")
	if err != nil {
		l.Error("marshal manifest failed", slog.Any("err", err))
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')

	// Ensure backups dir exists
	bdir := filepath.Join(ph.Root, BackupsDirName)
	if err := os.MkdirAll(bdir, 0o755); err != nil {
		l.Error("ensure backups dir failed", slog.Any("err", err))
		return fmt.Errorf("ensure backups dir: %w", err)
	}

	// If a current manifest exists, copy it to a timestamped backup before replacing
	if _, statErr := os.Stat(ph.ManifestPath); statErr == nil {
		stamp := time.Now().Format("20060102-150405")
		bname := fmt.Sprintf("%s.%s.bak", ManifestFileName, stamp)
		bpath := filepath.Join(bdir, bname)
		l.Debug("backup current manifest", slog.String("backup", bpath))
		if cerr := copyFile(ph.ManifestPath, bpath); cerr != nil {
			l.Error("backup current manifest failed", slog.Any("err", cerr))
			return fmt.Errorf("backup current manifest: %w", cerr)
		}
	}

	// Transactional write: to temp file in same directory, then rename over target
	dir := filepath.Dir(ph.ManifestPath)
	temp := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d-%d", ManifestFileName, os.Getpid(), rand.Int()))
	if werr := writeFileSync(temp, data); werr != nil {
		l.Error("write temp manifest failed", slog.Any("err", werr))
		return fmt.Errorf("write temp manifest: %w", werr)
	}
	// On Windows, replace by removing destination first if needed
	if _, err := os.Stat(ph.ManifestPath); err == nil {
		_ = os.Remove(ph.ManifestPath)
	}
	if rerr := os.Rename(temp, ph.ManifestPath); rerr != nil {
		// attempt cleanup temp
		_ = os.Remove(temp)
		l.Error("replace manifest failed", slog.Any("err", rerr))
		return fmt.Errorf("replace manifest: %w", rerr)
	}
	l.Info("manifest saved", slog.String("path", ph.ManifestPath))
	return nil
}

// SaveAs writes the manifest to a new root folder, scaffolding structure if needed, and updates the handle.
func SaveAs(ph *ProjectHandle, newRoot string) error {
	l := applog.WithOperation(applog.WithComponent("storage"), "saveas").With(slog.String("newRoot", newRoot))
	if ph == nil {
		return errors.New("nil ProjectHandle")
	}
	if newRoot == "" {
		return errors.New("new root is empty")
	}
	l.Info("save as starting")
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		l.Error("create new root failed", slog.Any("err", err))
		return fmt.Errorf("create new root: %w", err)
	}
	for _, d := range standardSubDirs {
		if err := os.MkdirAll(filepath.Join(newRoot, d), 0o755); err != nil {
			l.Error("create subdir failed", slog.String("dir", d), slog.Any("err", err))
			return fmt.Errorf("create subdir %s: %w", d, err)
		}
	}
	ph.Root = newRoot
	ph.ManifestPath = filepath.Join(newRoot, ManifestFileName)
	if err := Save(ph); err != nil {
		l.Error("save as failed", slog.Any("err", err))
		return err
	}
	l.Info("save as completed", slog.String("manifest", ph.ManifestPath))
	return nil
}

// writeFileSync writes data to a file, ensures it is flushed to disk.
func writeFileSync(path string, data []byte) (err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}

// copyFile copies a file from src to dst (overwrites dst if exists).
func copyFile(src, dst string) (err error) {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := sf.Close(); err == nil {
			err = cerr
		}
	}()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	df, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := df.Close(); err == nil {
			err = cerr
		}
	}()
	if _, err := io.Copy(df, sf); err != nil {
		return err
	}
	if err := df.Sync(); err != nil {
		return err
	}
	return nil
}

// openFromLatestBackup tries to open the latest timestamped backup.
func openFromLatestBackup(root string) (*domain.Project, error) {
	bdir := filepath.Join(root, BackupsDirName)
	ents, err := os.ReadDir(bdir)
	if err != nil {
		return nil, fmt.Errorf("read backups dir: %w", err)
	}
	var candidates []string
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ManifestFileName+".") && strings.HasSuffix(name, ".bak") {
			candidates = append(candidates, filepath.Join(bdir, name))
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no backups found")
	}
	sort.Strings(candidates) // timestamp in name yields lexicographic order
	latest := candidates[len(candidates)-1]
	b, err := os.ReadFile(latest)
	if err != nil {
		return nil, fmt.Errorf("read latest backup: %w", err)
	}
	var p domain.Project
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("parse latest backup: %w", err)
	}
	return &p, nil
}

// AutosaveCrashSnapshot writes a crash-recovery snapshot of the current project
// manifest into the backups directory without touching the main manifest file.
// The file name is comic.json.crash-<timestamp>.autosave for easy identification.
func AutosaveCrashSnapshot(ph *ProjectHandle) (string, error) {
	l := applog.WithOperation(applog.WithComponent("storage"), "autosave_crash")
	if ph == nil {
		return "", errors.New("nil ProjectHandle")
	}
	if ph.Root == "" {
		return "", errors.New("invalid ProjectHandle: missing root")
	}
	bdir := filepath.Join(ph.Root, BackupsDirName)
	if err := os.MkdirAll(bdir, 0o755); err != nil {
		l.Error("ensure backups dir failed", slog.Any("err", err))
		return "", fmt.Errorf("ensure backups dir: %w", err)
	}
	stamp := time.Now().Format("20060102-150405")
	fname := fmt.Sprintf("%s.crash-%s.autosave", ManifestFileName, stamp)
	bpath := filepath.Join(bdir, fname)

	data, err := json.MarshalIndent(ph.Project, "", "  ")
	if err != nil {
		l.Error("marshal crash snapshot failed", slog.Any("err", err))
		return "", fmt.Errorf("marshal crash snapshot: %w", err)
	}
	data = append(data, '\n')
	if werr := writeFileSync(bpath, data); werr != nil {
		l.Error("write crash snapshot failed", slog.Any("err", werr))
		return "", fmt.Errorf("write crash snapshot: %w", werr)
	}
	l.Info("crash snapshot written", slog.String("path", bpath))
	return bpath, nil
}

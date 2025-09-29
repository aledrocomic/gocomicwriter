/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * Licensed under the Apache License, Version 2.0.
 */

package stylepack

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	applog "gocomicwriter/internal/log"
)

// ExportProjectStyles zips the project's styles directory (<project>/styles) into a single .zip file.
// The produced archive preserves the directory structure and adds a small manifest file at the root
// named stylepack.manifest.txt for quick human inspection.
// If the styles directory does not exist or is empty, it still creates the archive with only the manifest.
func ExportProjectStyles(projectRoot string, destZipPath string) error {
	l := applog.WithOperation(applog.WithComponent("stylepack"), "export").With(slog.String("project", projectRoot))
	if strings.TrimSpace(projectRoot) == "" {
		return errors.New("projectRoot is required")
	}
	if strings.TrimSpace(destZipPath) == "" {
		return errors.New("destZipPath is required")
	}
	stylesDir := filepath.Join(projectRoot, "styles")
	if _, err := os.Stat(stylesDir); os.IsNotExist(err) {
		// Create empty dir semantics
		if err := os.MkdirAll(stylesDir, 0o755); err != nil {
			return fmt.Errorf("ensure styles dir: %w", err)
		}
	}

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(destZipPath), 0o755); err != nil {
		return fmt.Errorf("ensure zip dir: %w", err)
	}
	// On Windows, remove destination if present before create
	_ = os.Remove(destZipPath)

	zf, err := os.Create(destZipPath)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer func() { _ = zf.Close() }()
	zw := zip.NewWriter(zf)
	defer func() { _ = zw.Close() }()

	// Add manifest text
	manifest := fmt.Sprintf("Go Comic Writer Style Pack\nCreated: %s\nProject: %s\n\nContents mirror the project's /styles directory.\n",
		time.Now().Format(time.RFC3339), projectRoot)
	w, err := zw.Create("stylepack.manifest.txt")
	if err != nil {
		return fmt.Errorf("add manifest: %w", err)
	}
	if _, err := w.Write([]byte(manifest)); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Walk styles folder and add files
	added := 0
	err = filepath.Walk(stylesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		// Normalize to forward slashes inside zip per spec
		zipName := filepath.ToSlash(rel)
		fw, err := zw.Create(zipName)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		if _, err := io.Copy(fw, f); err != nil {
			return err
		}
		added++
		return nil
	})
	if err != nil {
		l.Error("zip build failed", slog.Any("err", err))
		return fmt.Errorf("build zip: %w", err)
	}
	l.Info("style pack exported", slog.Int("files", added), slog.String("zip", destZipPath))
	return nil
}

// InstallPack extracts the given .zip pack into the project's styles directory.
// Existing files are not overwritten; if a file already exists, it is skipped.
// Returns the count of files installed (skipped files are not counted).
func InstallPack(projectRoot string, packZipPath string) (int, error) {
	l := applog.WithOperation(applog.WithComponent("stylepack"), "install").With(slog.String("project", projectRoot))
	if strings.TrimSpace(projectRoot) == "" {
		return 0, errors.New("projectRoot is required")
	}
	if strings.TrimSpace(packZipPath) == "" {
		return 0, errors.New("packZipPath is required")
	}
	stylesDir := filepath.Join(projectRoot, "styles")
	if err := os.MkdirAll(stylesDir, 0o755); err != nil {
		return 0, fmt.Errorf("ensure styles dir: %w", err)
	}

	r, err := zip.OpenReader(packZipPath)
	if err != nil {
		return 0, fmt.Errorf("open pack: %w", err)
	}
	defer func() { _ = r.Close() }()

	installed := 0
	for _, f := range r.File {
		name := f.Name
		// Skip top-level manifest file
		if name == "stylepack.manifest.txt" {
			continue
		}
		// Only install files that target the styles directory or subfolders
		// Accept either paths starting with "styles/" or any other structure — we'll place under styles/pack/<archive-root>/...
		// We detect if the rel path already starts with "styles/"; otherwise we prefix with "styles/".
		targetRel := name
		if !strings.HasPrefix(targetRel, "styles/") {
			targetRel = filepath.ToSlash(filepath.Join("styles", targetRel))
		}
		targetPath := filepath.Join(projectRoot, filepath.FromSlash(targetRel))
		// If file exists, skip
		if _, err := os.Stat(targetPath); err == nil {
			l.Warn("skip existing file", slog.String("path", targetPath))
			continue
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return installed, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return installed, err
		}
		rc, err := f.Open()
		if err != nil {
			return installed, err
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return installed, err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			_ = rc.Close()
			return installed, err
		}
		_ = out.Close()
		_ = rc.Close()
		installed++
	}
	l.Info("style pack installed", slog.Int("files", installed))
	return installed, nil
}

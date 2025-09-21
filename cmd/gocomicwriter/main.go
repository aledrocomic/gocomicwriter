/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gocomicwriter/internal/crash"
	"gocomicwriter/internal/domain"
	applog "gocomicwriter/internal/log"
	"gocomicwriter/internal/storage"
	"gocomicwriter/internal/ui"
	"gocomicwriter/internal/version"
)

func usage() {
	fmt.Println("Go Comic Writer — development skeleton")
	fmt.Printf("Version: %s\n", version.String())
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gocomicwriter version|-v|--version        Show version")
	fmt.Println("  gocomicwriter init <dir> <name>            Create a new project at <dir> with name <name>")
	fmt.Println("  gocomicwriter open <dir>                    Open project at <dir> and print summary")
	fmt.Println("  gocomicwriter save <dir>                    Save project at <dir> (creates backup) ")
	fmt.Println("  gocomicwriter ui [<dir>]                    Launch desktop UI (build with -tags fyne for full UI)")
}

func main() {
	// initialize structured logging using environment defaults
	applog.Init(applog.FromEnv())
	l := applog.WithComponent("cli")
	var ph *storage.ProjectHandle
	defer func() { crash.Recover(ph) }()

	args := os.Args
	l.Debug("start", slog.Int("args", len(args)))
	if len(args) > 1 {
		switch args[1] {
		case "version", "--version", "-v":
			fmt.Println("Go Comic Writer — development skeleton")
			fmt.Println(version.String())
			return
		case "init":
			if len(args) < 4 {
				fmt.Println("init requires <dir> and <name>")
				usage()
				os.Exit(2)
			}
			dir := args[2]
			name := args[3]
			abs, _ := filepath.Abs(dir)
			l.Info("init project", slog.String("root", abs), slog.String("name", name))
			p := domain.Project{Name: name, Issues: []domain.Issue{}}
			h, err := storage.InitProject(abs, p)
			if err != nil {
				l.Error("init failed", slog.Any("err", err))
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			ph = h
			fmt.Println("Created project at", abs)
			return
		case "open":
			if len(args) < 3 {
				fmt.Println("open requires <dir>")
				usage()
				os.Exit(2)
			}
			dir := args[2]
			abs, _ := filepath.Abs(dir)
			l.Info("open project", slog.String("root", abs))
			h, err := storage.Open(abs)
			if err != nil {
				l.Error("open failed", slog.Any("err", err))
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			ph = h
			fmt.Printf("Opened project: %s\n", h.Project.Name)
			fmt.Printf("Issues: %d\n", len(h.Project.Issues))
			fmt.Println("Root:", h.Root)
			return
		case "save":
			if len(args) < 3 {
				fmt.Println("save requires <dir>")
				usage()
				os.Exit(2)
			}
			dir := args[2]
			abs, _ := filepath.Abs(dir)
			l.Info("save project", slog.String("root", abs))
			h, err := storage.Open(abs)
			if err != nil {
				l.Error("open before save failed", slog.Any("err", err))
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			ph = h
			// Touch metadata to ensure changed content for demo purposes
			h.Project.Metadata.Notes = fmt.Sprintf("Saved at %s", time.Now().Format(time.RFC3339))
			if err := storage.Save(h); err != nil {
				l.Error("save failed", slog.Any("err", err))
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			fmt.Println("Saved project and created a backup of previous manifest (if any).")
			return
		case "ui":
			var dir string
			if len(args) >= 3 {
				dir = args[2]
			}
			if err := ui.Run(dir); err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			return
		}
	}

	usage()
}

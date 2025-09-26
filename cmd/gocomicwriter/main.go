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
	"os"
	"strings"

	applog "gocomicwriter/internal/log"
	"gocomicwriter/internal/ui"
)

func main() {
	// Initialize structured logging using environment defaults
	applog.Init(applog.FromEnv())

	if len(os.Args) >= 2 {
		sub := strings.ToLower(os.Args[1])
		if strings.HasPrefix(sub, "export") {
			fmt.Println("Command-line exports have been moved to the UI. Please use the Export menu.")
			os.Exit(2)
		}
	}

	// UI-only launcher: optional first arg is a project directory to open.
	var dir string
	if len(os.Args) >= 2 {
		dir = os.Args[1]
		// Backward compatibility: if someone uses the old `ui` subcommand,
		// treat the next argument as the project directory.
		if dir == "ui" {
			if len(os.Args) >= 3 {
				dir = os.Args[2]
			} else {
				dir = ""
			}
		}
	}

	if err := ui.Run(dir); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

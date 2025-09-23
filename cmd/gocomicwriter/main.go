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

	"gocomic/internal/version"
)

func main() {
	// Minimal CLI entrypoint for the Go Comic Writer project.
	// For now, it prints a banner and an optional version.
	args := os.Args
	if len(args) > 1 {
		switch args[1] {
		case "version", "--version", "-v":
			fmt.Println(version.String())
			return
		}
	}

	fmt.Println("Go Comic Writer — development skeleton")
	fmt.Printf("Version: %s\n", version.String())
}

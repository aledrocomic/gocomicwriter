//go:build fyne && !cgo

/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package ui

import "fmt"

// Run informs the user that Fyne UI requires cgo (OpenGL) and a C toolchain.
// This stub is compiled when the build uses -tags fyne but CGO is disabled.
func Run(_ string) error {
	return fmt.Errorf("Fyne UI requires cgo (OpenGL). Enable cgo and install a C toolchain. On Windows: install MSYS2/MinGW-w64, ensure gcc is on PATH, then run with CGO_ENABLED=1. Example: set CGO_ENABLED=1 && go run -tags fyne ./cmd/gocomicwriter ui [projectDir]")
}

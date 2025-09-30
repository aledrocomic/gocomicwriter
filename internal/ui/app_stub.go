//go:build !fyne

/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package ui

import "fmt"

// Run starts the desktop UI. In non-fyne builds, this is a stub so CI remains headless.
// Pass an optional project directory to open immediately.
func Run(_ string) error {
	return fmt.Errorf("UI not built in this binary. Rebuild with: go run -tags fyne ./cmd/gocomicwriter [projectDir]")
}

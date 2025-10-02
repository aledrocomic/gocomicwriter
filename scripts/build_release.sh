#!/usr/bin/env bash
#
# Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
# This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.  You may obtain a copy of the License at
#   http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
#  specific language governing permissions and limitations under the License.
#

set -euo pipefail

# Simple helper to build cross-platform artifacts using GoReleaser.
# - Snapshot build (default): ./scripts/build_release.sh
# - Full release (requires tag, configured publishing): ./scripts/build_release.sh release

cmd="build --snapshot --clean"
if [[ "${1:-}" == "release" ]]; then
  cmd="release --clean"
fi

if ! command -v goreleaser >/dev/null 2>&1; then
  echo "GoReleaser not found. Install from https://goreleaser.com/install/" >&2
  exit 1
fi

# Run from repo root (this script lives in scripts/)
cd "$(dirname "$0")/.."

goreleaser $cmd

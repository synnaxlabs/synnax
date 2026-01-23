#!/usr/bin/env bash
set -euo pipefail

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

SCRIPT_DIR="$(
  cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1
  pwd
)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." >/dev/null 2>&1 && pwd)"

fail() {
  echo "Error: $*" >&2
  exit 1
}

require_dir() {
  [[ -d "$1" ]] || fail "directory not found: $1"
}

require_file() {
  [[ -f "$1" ]] || fail "file not found: $1"
}

# Extract major.minor (X.Y) from a semver-ish string.
major_minor() {
  echo "$1" | cut -d '.' -f1-2
}

get_expected_version_mm() {
  local version_file="$ROOT_DIR/core/pkg/version/VERSION"
  require_file "$version_file"

  local full
  full="$(<"$version_file")"
  EXPECTED_MM="$(major_minor "$full")"
  [[ -n "$EXPECTED_MM" ]] || fail "failed to parse version from $version_file"
  echo "Expected version: ${EXPECTED_MM}.x (from core/pkg/version/VERSION)"
}

read_node_mm() {
  local pkg_json="$1"
  require_file "$pkg_json"

  local full
  full="$(grep -m1 '"version"[[:space:]]*:' "$pkg_json" | cut -d '"' -f4)"
  [[ -n "$full" ]] || fail "could not read version from $pkg_json"
  major_minor "$full"
}

read_python_mm() {
  local pyproject="$1"
  require_file "$pyproject"

  local full
  full="$(grep -m1 '^version[[:space:]]*=' "$pyproject" | cut -d '"' -f2)"
  [[ -n "$full" ]] || fail "could not read version from $pyproject"
  major_minor "$full"
}

read_tauri_mm() {
  local tauri_conf="$1"
  require_file "$tauri_conf"

  local full
  full="$(grep -m1 '"version"[[:space:]]*:' "$tauri_conf" | cut -d '"' -f4)"
  [[ -n "$full" ]] || fail "could not read version from $tauri_conf"
  major_minor "$full"
}

check_match() {
  local label="$1"
  local found_mm="$2"
  local expected_mm="$3"
  local source="$4"

  if [[ "$found_mm" != "$expected_mm" ]]; then
    echo "❌ $label version mismatch: found ${found_mm}.x, expected ${expected_mm}.x ($source)" >&2
    return 1
  fi
  echo "✅ $label version ok (${found_mm}.x)"
  return 0
}

main() {
  get_expected_version_mm

  local ok=true

  local PYTHON_DIRS=(
    "$ROOT_DIR/alamos/py"
    "$ROOT_DIR/freighter/py"
    "$ROOT_DIR/client/py"
  )

  for d in "${PYTHON_DIRS[@]}"; do
    require_dir "$d"
    local f="$d/pyproject.toml"
    local found
    found="$(read_python_mm "$f")"
    if ! check_match "Python ($d)" "$found" "$EXPECTED_MM" "$f"; then ok=false; fi
  done

  local NODE_DIRS=(
    "$ROOT_DIR/alamos/ts"
    "$ROOT_DIR/client/ts"
    "$ROOT_DIR/configs/eslint"
    "$ROOT_DIR/configs/stylelint"
    "$ROOT_DIR/configs/ts"
    "$ROOT_DIR/configs/vite"
    "$ROOT_DIR/drift"
    "$ROOT_DIR/freighter/ts"
    "$ROOT_DIR/pluto"
    "$ROOT_DIR/x/media"
    "$ROOT_DIR/x/ts"
  )

  for d in "${NODE_DIRS[@]}"; do
    require_dir "$d"
    local f="$d/package.json"
    local found
    found="$(read_node_mm "$f")"
    if ! check_match "Node ($d)" "$found" "$EXPECTED_MM" "$f"; then ok=false; fi
  done

  local TAURI_CONF="$ROOT_DIR/console/src-tauri/tauri.conf.json"
  local tauri_found
  tauri_found="$(read_tauri_mm "$TAURI_CONF")"
  if ! check_match "Tauri (console/src-tauri/tauri.conf.json)" "$tauri_found" "$EXPECTED_MM" "$TAURI_CONF"; then ok=false; fi

  if [[ "$ok" == true ]]; then
    echo "All versions match."
    exit 0
  else
    echo "Version check failed."
    exit 1
  fi
}

main

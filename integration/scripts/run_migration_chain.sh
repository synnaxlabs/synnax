#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Run migration setup chain: for each version in the chain, download Core,
# start it, install the matching synnax client, run the setup script, stop Core.
#
# The data directory persists across versions so each setup builds on the
# previous version's state.
#
# Usage:
#   run_migration_chain.sh --chain "0.54.0,0.55.2"

set -euo pipefail

REPO="synnaxlabs/synnax"
PORT=9090
STARTUP_TIMEOUT=30
STOP_TIMEOUT=10

DATA_DIR="${HOME}/synnax-data"
BINARY_DIR="${HOME}/synnax-binaries"
CACHE_DIR="${HOME}/synnax-binary-cache"
VENV_DIR="${HOME}/migration-client-env"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INTEGRATION_DIR="$(dirname "$SCRIPT_DIR")"

# ── Helpers ──────────────────────────────────────────────────────────────────

get_platform() {
    case "$(uname -s)" in
        Darwin) echo "macos" ;;
        Linux) echo "linux" ;;
        MINGW* | MSYS* | CYGWIN*) echo "windows" ;;
        *)
            echo "Unsupported platform: $(uname -s)" >&2
            exit 1
            ;;
    esac
}

get_binary_name() {
    if [[ "$(get_platform)" == "windows" ]]; then
        echo "synnax.exe"
    else
        echo "synnax"
    fi
}

# Convert version to setup folder name: 0.54.3 -> v0_54
version_to_folder() {
    local version="$1"
    local major_minor="${version%.*}"
    echo "v${major_minor//./_}"
}

# Convert version to pip specifier: 0.54.3 -> "synnax>=0.54,<0.55"
version_to_pip_spec() {
    local version="$1"
    local major_minor="${version%.*}"
    IFS='.' read -r major minor <<< "$major_minor"
    local next_minor=$((minor + 1))
    echo "synnax>=${major}.${minor},<${major}.${next_minor}"
}

install_version() {
    local version="$1"
    local plat binary_name asset cached target tag

    plat="$(get_platform)"
    binary_name="$(get_binary_name)"

    mkdir -p "$BINARY_DIR" "$CACHE_DIR"

    if [[ "$plat" == "windows" ]]; then
        asset="synnax-v${version}-windows.exe"
    else
        asset="synnax-v${version}-${plat}"
    fi

    cached="${CACHE_DIR}/${asset}"
    target="${BINARY_DIR}/${binary_name}"

    if [[ -f "$cached" ]]; then
        echo "Using cached binary for v${version}"
    else
        tag="synnax-v${version}"
        echo "Downloading ${asset} from release ${tag}..."
        gh release download "$tag" \
            --repo "$REPO" \
            --pattern "$asset" \
            --dir "$CACHE_DIR"
    fi

    cp "$cached" "$target"
    if [[ "$plat" != "windows" ]]; then
        chmod +x "$target"
    fi
    echo "Installed v${version} -> ${target}"
}

start_core() {
    local binary_name log_file
    binary_name="$(get_binary_name)"
    log_file="${DATA_DIR}/synnax-core.log"

    mkdir -p "$DATA_DIR"

    echo "Starting Core..."
    "${BINARY_DIR}/${binary_name}" start -i \
        > "$log_file" 2>&1 &
    CORE_PID=$!

    local elapsed=0
    while ((elapsed < STARTUP_TIMEOUT)); do
        if nc -z localhost "$PORT" 2> /dev/null; then
            echo "Core is ready on port ${PORT} (pid ${CORE_PID})"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done

    kill "$CORE_PID" 2> /dev/null || true
    echo "--- Core log (last 2000 chars) ---"
    tail -c 2000 "$log_file" 2> /dev/null || true
    echo "--- end log ---"
    echo "ERROR: Core did not start within ${STARTUP_TIMEOUT}s" >&2
    exit 1
}

stop_core() {
    echo "Stopping Core (pid ${CORE_PID})..."
    kill "$CORE_PID" 2> /dev/null || true

    local elapsed=0
    while ((elapsed < STOP_TIMEOUT)); do
        if ! kill -0 "$CORE_PID" 2> /dev/null; then
            break
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done

    if kill -0 "$CORE_PID" 2> /dev/null; then
        echo "Core did not stop gracefully, killing..."
        kill -9 "$CORE_PID" 2> /dev/null || true
        wait "$CORE_PID" 2> /dev/null || true
    fi

    elapsed=0
    while ((elapsed < STOP_TIMEOUT)); do
        if ! nc -z localhost "$PORT" 2> /dev/null; then
            echo "Core stopped"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done

    echo "WARNING: Port ${PORT} still in use after Core stopped" >&2
}

create_venv_and_install() {
    local pip_spec="$1"
    local python_path

    if [[ "$(get_platform)" == "windows" ]]; then
        python_path="${VENV_DIR}/Scripts/python.exe"
    else
        python_path="${VENV_DIR}/bin/python"
    fi

    rm -rf "$VENV_DIR"
    echo "Creating venv and installing ${pip_spec}..."
    uv venv "$VENV_DIR" --quiet
    uv pip install --quiet --python "$python_path" "$pip_spec"
}

run_setup_script() {
    local folder="$1"
    local setup_script="${INTEGRATION_DIR}/migration/setup/${folder}/setup.py"

    if [[ ! -f "$setup_script" ]]; then
        echo "WARNING: No setup script found at ${setup_script}, skipping"
        return 0
    fi

    echo "Running setup script: ${setup_script}"
    "${VENV_DIR}/bin/python" "$setup_script"
}

cleanup() {
    echo "Cleaning up..."
    for dir in "$VENV_DIR"; do
        if [[ -d "$dir" ]]; then
            rm -rf "$dir" && echo "Cleaned ${dir}" || echo "WARNING: failed to clean ${dir}"
        fi
    done
}

# ── Main ─────────────────────────────────────────────────────────────────────

CHAIN=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --chain)
            CHAIN="$2"
            shift 2
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

if [[ -z "$CHAIN" ]]; then
    echo "Usage: $0 --chain \"0.54.0,0.55.2\"" >&2
    exit 1
fi

echo "Migration setup chain"
echo "  Chain: ${CHAIN}"
echo "########################################################"
echo

CORE_PID=""
trap 'stop_core 2>/dev/null || true; cleanup' EXIT

rm -rf "$DATA_DIR"

IFS=',' read -ra VERSIONS <<< "$CHAIN"
for version in "${VERSIONS[@]}"; do
    version="$(echo "$version" | xargs)"
    folder="$(version_to_folder "$version")"
    pip_spec="$(version_to_pip_spec "$version")"

    echo ""
    echo "=== Version ${version} (folder: ${folder}) ==="

    install_version "$version"
    start_core
    create_venv_and_install "$pip_spec"
    run_setup_script "$folder"
    stop_core
done

echo ""
echo "Setup chain complete."

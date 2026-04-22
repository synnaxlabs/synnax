#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test conductor.

Manages Core binary lifecycle and orchestrates migration test chains.
Python selected for cross-platform compatibility.

This module intentionally uses only stdlib imports — no synnax, xpy, or
framework dependencies — so it can create isolated venvs with different
synnax client versions for each setup step in the migration chain. The
exception is the ``examples`` simulator package, which is imported to
start device simulators during the setup phase.

The chain flow for ``--from 0.55.0``:

1. Discover setup folders with version <= 0.55 (e.g., v0_54)
2. For each: download versioned Core, start it, install matching client
   from PyPI, run setup.py, stop Core
3. Download + start Core v0.55.0 (no-op — triggers migrations on data)
4. Start latest Core, run verify via ``uv run tc migration``, stop Core
5. Clean up migration data directory

Usage:
    uv run migration-conductor --from 0.55.0          # CI mode
    uv run migration-conductor --from 0.55.0 --dev    # Local dev mode
"""

import argparse
import os
import platform
import re
import shutil
import socket
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

MIGRATION_BINARY_DIR = Path.home() / "synnax-migration-binaries"
CI_BINARY_DIR = Path.home() / "synnax-binaries"
DATA_DIR = Path.home() / "synnax-migration-data"
BINARY_NAME = "synnax.exe" if platform.system() == "Windows" else "synnax"
REPO = "synnaxlabs/synnax"
PORT = 9090
STARTUP_TIMEOUT = 30
STARTUP_POLL_INTERVAL = 1
STOP_TIMEOUT = 10
KILL_TIMEOUT = 5
CLEANUP_TIMEOUT = 10
INTEGRATION_DIR = Path(__file__).parent
SETUP_DIR = INTEGRATION_DIR / "migration" / "setup"
CLIENT_VENV_DIR = Path.home() / "migration-client-env"
CORE_DIR = INTEGRATION_DIR.parent / "core"


def log(msg: str) -> None:
    ts = datetime.now().strftime("%H:%M:%S.%f")[:-4]
    print(f"{ts} | {msg}")


def _kill_port(port: int) -> None:
    """Kill any process listening on the given port."""
    try:
        with socket.create_connection(("localhost", port), timeout=0.5):
            pass
    except (ConnectionRefusedError, OSError):
        return
    log(f"Killing process on port {port}...")
    if platform.system() == "Windows":
        subprocess.run(
            ["powershell", "-Command",
             f"Stop-Process -Id (Get-NetTCPConnection -LocalPort {port}).OwningProcess -Force"],
            capture_output=True,
        )
    else:
        subprocess.run(
            ["bash", "-c", f"lsof -ti:{port} | xargs kill -9 2>/dev/null"],
            capture_output=True,
        )
    time.sleep(1)


def _ensure_clean_state() -> None:
    """Kill stale processes and clean up directories."""
    log("Ensuring clean state...")
    _kill_port(PORT)
    _kill_port(4841)  # OPC UA simulator
    _kill_port(5020)  # Modbus simulator
    for d in [DATA_DIR, CLIENT_VENV_DIR]:
        if not d.exists():
            continue
        start = time.monotonic()
        while time.monotonic() - start < CLEANUP_TIMEOUT:
            try:
                shutil.rmtree(d)
                log(f"Cleaned {d}")
                break
            except PermissionError:
                time.sleep(1)
        else:
            log(f"Warning: failed to clean {d} after {CLEANUP_TIMEOUT}s")


def _get_platform() -> str:
    """Map platform.system() to the asset suffix used in release binaries."""
    return {
        "Darwin": "macos",
        "Linux": "linux",
        "Windows": "windows",
    }[platform.system()]


def _venv_python(venv_dir: Path) -> Path:
    """Return the path to the Python executable inside a venv."""
    if platform.system() == "Windows":
        return venv_dir / "Scripts" / "python.exe"
    return venv_dir / "bin" / "python"


def install_version(version: str) -> Path:
    """Download and cache a specific Core version binary.

    Binaries are cached in MIGRATION_BINARY_DIR with version-specific names
    so multiple versions can coexist. Returns the path to the binary.
    """
    MIGRATION_BINARY_DIR.mkdir(parents=True, exist_ok=True)

    plat = _get_platform()
    suffix = "-windows.exe" if plat == "windows" else f"-{plat}"
    asset = f"synnax-v{version}{suffix}"
    cached = MIGRATION_BINARY_DIR / asset

    if cached.exists():
        log(f"Using cached binary for v{version}")
    else:
        tag = f"synnax-v{version}"
        log(f"Downloading {asset} from release {tag}...")
        subprocess.run(
            [
                "gh",
                "release",
                "download",
                tag,
                "--repo",
                REPO,
                "--pattern",
                asset,
                "--dir",
                str(MIGRATION_BINARY_DIR),
            ],
            check=True,
        )

    if platform.system() != "Windows":
        cached.chmod(0o755)
    log(f"Ready: v{version} at {cached}")
    return cached


def start_core(binary: Path) -> subprocess.Popen[bytes]:
    """Start a Core binary against the shared migration data directory."""
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    log_file = DATA_DIR / "synnax-core.log"

    env = os.environ.copy()
    env.setdefault("SYNNAX_LICENSE_KEY", "")

    log(f"Starting Core from {binary}...")
    log_fh = open(log_file, "w")
    proc = subprocess.Popen(
        [str(binary), "start", "-i", "-d", str(DATA_DIR)],
        stdout=log_fh,
        stderr=log_fh,
        env=env,
    )
    log_fh.close()

    start = time.monotonic()
    while time.monotonic() - start < STARTUP_TIMEOUT:
        try:
            with socket.create_connection(("localhost", PORT), timeout=1):
                log(f"Core is ready on port {PORT}")
                return proc
        except (ConnectionRefusedError, OSError):
            time.sleep(STARTUP_POLL_INTERVAL)

    proc.kill()
    if log_file.exists():
        log("--- Core log ---")
        log(log_file.read_text()[-2000:])
        log("--- end log ---")
    raise TimeoutError(
        f"Core did not become ready on port {PORT} within {STARTUP_TIMEOUT}s"
    )


def start_core_dev() -> subprocess.Popen[bytes]:
    """Start Core from source using ``go run`` for local development."""
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    log_file = DATA_DIR / "synnax-core.log"

    env = os.environ.copy()
    env.setdefault("SYNNAX_LICENSE_KEY", "")

    log(f"Starting Core from source (go run) in {CORE_DIR}...")
    log_fh = open(log_file, "w")
    proc = subprocess.Popen(
        ["go", "run", "-tags", "driver", "main.go", "start", "-i", "-d", str(DATA_DIR)],
        cwd=str(CORE_DIR),
        stdout=log_fh,
        stderr=log_fh,
        env=env,
    )
    log_fh.close()

    start = time.monotonic()
    while time.monotonic() - start < STARTUP_TIMEOUT:
        try:
            with socket.create_connection(("localhost", PORT), timeout=1):
                log(f"Core is ready on port {PORT}")
                return proc
        except (ConnectionRefusedError, OSError):
            time.sleep(STARTUP_POLL_INTERVAL)

    proc.kill()
    if log_file.exists():
        log("--- Core log ---")
        log(log_file.read_text()[-2000:])
        log("--- end log ---")
    raise TimeoutError(
        f"Core did not become ready on port {PORT} within {STARTUP_TIMEOUT}s"
    )


def stop_core(proc: subprocess.Popen[bytes]) -> None:
    """Stop Core process and wait for port to be released."""
    log("Stopping Core...")
    proc.terminate()
    try:
        proc.wait(timeout=STOP_TIMEOUT)
    except subprocess.TimeoutExpired:
        log("Core did not stop gracefully, killing...")
        proc.kill()
        proc.wait(timeout=KILL_TIMEOUT)

    start = time.monotonic()
    while time.monotonic() - start < STOP_TIMEOUT:
        try:
            with socket.create_connection(("localhost", PORT), timeout=1):
                time.sleep(STARTUP_POLL_INTERVAL)
        except (ConnectionRefusedError, OSError):
            log("Core stopped")
            return

    log(f"WARNING: Port {PORT} still in use after Core process exited")


def _parse_folder_version(folder_name: str) -> tuple[int, int] | None:
    """Parse a setup folder name like 'v0_54' into (0, 54)."""
    m = re.match(r"^v(\d+)_(\d+)$", folder_name)
    if m is None:
        return None
    return int(m.group(1)), int(m.group(2))


def _parse_version(version_str: str) -> tuple[int, int]:
    """Parse a version string like '0.55.0' into its major.minor tuple (0, 55)."""
    parts = version_str.split(".")
    return int(parts[0]), int(parts[1])


def discover_setup_folders(from_version: str) -> list[tuple[str, tuple[int, int]]]:
    """Scan the setup directory for version folders <= from_version.

    Returns a sorted list of (folder_name, (major, minor)) tuples.
    """
    from_mm = _parse_version(from_version)
    folders: list[tuple[str, tuple[int, int]]] = []
    if not SETUP_DIR.exists():
        return folders
    for entry in SETUP_DIR.iterdir():
        if not entry.is_dir():
            continue
        parsed = _parse_folder_version(entry.name)
        if parsed is None:
            continue
        if parsed <= from_mm:
            folders.append((entry.name, parsed))
    folders.sort(key=lambda x: x[1])
    return folders


def _folder_to_pip_spec(major: int, minor: int) -> str:
    """Convert a (major, minor) tuple to a pip version specifier.

    E.g., (0, 54) -> 'synnax>=0.54,<0.55'
    """
    return f"synnax>={major}.{minor},<{major}.{minor + 1}"


def _folder_to_release_version(major: int, minor: int) -> str:
    """Convert a (major, minor) tuple to a release version for downloading
    Core binaries. E.g., (0, 54) -> '0.54.0'"""
    return f"{major}.{minor}.0"


def create_setup_venv(major: int, minor: int) -> Path:
    """Create an isolated venv and install the matching synnax client from PyPI.

    Returns the path to the venv's Python executable.
    """
    if CLIENT_VENV_DIR.exists():
        shutil.rmtree(CLIENT_VENV_DIR)

    pip_spec = _folder_to_pip_spec(major, minor)
    log(f"Creating venv and installing {pip_spec}...")
    subprocess.run(
        ["uv", "venv", str(CLIENT_VENV_DIR)],
        check=True,
        capture_output=True,
    )

    python = _venv_python(CLIENT_VENV_DIR)
    subprocess.run(
        [
            "uv", "pip", "install", "--quiet", "--python", str(python),
            pip_spec, "pymodbus", "asyncua",
        ],
        check=True,
    )

    log(f"Venv ready with {pip_spec}")
    return python


def run_setup_script(folder_name: str, python: Path) -> None:
    """Run the setup.py script in the given version folder."""
    script = SETUP_DIR / folder_name / "setup.py"
    if not script.exists():
        raise FileNotFoundError(f"Setup script not found: {script}")

    log(f"Running setup: {script}")
    result = subprocess.run(
        [str(python), str(script)],
        cwd=str(INTEGRATION_DIR),
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"Setup script {folder_name}/setup.py failed "
            f"(exit code {result.returncode})"
        )


def run_verify() -> bool:
    """Run migration verify tests via the test conductor."""
    log("Running: uv run tc migration")
    result = subprocess.run(
        ["uv", "run", "tc", "migration"],
        cwd=str(INTEGRATION_DIR),
    )
    if result.returncode != 0:
        log(f"Verify FAILED (exit code {result.returncode})")
        return False
    log("Verify passed")
    return True


def run(from_version: str, dev: bool) -> bool:
    """Run the full migration test chain.

    1. For each setup folder <= from_version: download versioned Core,
       start it, install matching client, run setup.py, stop Core.
    2. Download + start Core at from_version (no-op migration trigger).
    3. Start latest Core, run verify, stop Core.
    """
    setup_folders = discover_setup_folders(from_version)
    if not setup_folders:
        log(f"WARNING: No setup folders found for versions <= {from_version}")

    log(f"Setup folders: {[f[0] for f in setup_folders]}")
    log(f"From version: {from_version}")
    log(f"Mode: {'dev' if dev else 'CI'}\n")

    for folder_name, (major, minor) in setup_folders:
        release_version = _folder_to_release_version(major, minor)
        log(f"{'=' * 60}")
        log(f"Setup: {folder_name} (Core v{release_version})")
        log(f"{'=' * 60}")
        binary = install_version(release_version)
        proc = start_core(binary)
        try:
            python = create_setup_venv(major, minor)
            run_setup_script(folder_name, python)
        finally:
            stop_core(proc)

    from_mm = _parse_version(from_version)
    already_ran = any(v == from_mm for _, v in setup_folders)
    if not already_ran:
        log(f"{'=' * 60}")
        log(f"No-op migration: Core v{from_version}")
        log(f"{'=' * 60}")
        binary = install_version(from_version)
        proc = start_core(binary)
        stop_core(proc)

    log(f"{'=' * 60}")
    log("Verify: latest Core")
    log(f"{'=' * 60}")
    if dev:
        proc = start_core_dev()
    else:
        latest_binary = CI_BINARY_DIR / BINARY_NAME
        if not latest_binary.exists():
            raise FileNotFoundError(
                f"Expected CI-built binary at {latest_binary}. "
                "Pass --dev to run from source instead."
            )
        proc = start_core(latest_binary)

    return run_verify()


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Migration test conductor",
    )
    parser.add_argument(
        "--from",
        dest="from_version",
        required=True,
        help="Version to migrate from (e.g., '0.55.0'). All setup folders "
        "with version <= this value are run before verification.",
    )
    parser.add_argument(
        "--dev",
        action="store_true",
        help="Use 'go run' from source for the latest Core instead of a "
        "pre-built CI binary.",
    )
    args = parser.parse_args()

    log("Migration conductor")
    log(f"  From: {args.from_version}")
    log(f"  Dev:  {args.dev}")
    log(f"{'#' * 60}\n")

    _ensure_clean_state()

    success = False
    try:
        success = run(args.from_version, args.dev)
    finally:
        _ensure_clean_state()

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()

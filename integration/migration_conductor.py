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
import importlib.util
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
SETUP_DIR = INTEGRATION_DIR / "tests" / "migration"
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
            [
                "powershell",
                "-Command",
                f"Stop-Process -Id (Get-NetTCPConnection -LocalPort {port}).OwningProcess -Force",
            ],
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


def _prune_binaries(needed_versions: set[str]) -> None:
    """Remove cached binaries that aren't in the needed set."""
    if not MIGRATION_BINARY_DIR.exists():
        return
    plat = _get_platform()
    suffix = "-windows.exe" if plat == "windows" else f"-{plat}"
    needed_names = {f"synnax-v{v}{suffix}" for v in needed_versions}
    for entry in MIGRATION_BINARY_DIR.iterdir():
        if entry.name not in needed_names:
            entry.unlink()


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
        log(f"  Using cached binary for v{version}")
    else:
        tag = f"synnax-v{version}"
        log(f"  Downloading v{version} binary...")
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
    return cached


def start_core(
    binary: Path | None = None, *, dev: bool = False
) -> subprocess.Popen[bytes]:
    """Start a Core process against the shared migration data directory.

    Pass a binary path for release binaries, or ``dev=True`` to run from
    source via ``go run``.
    """
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    log_file = DATA_DIR / "synnax-core.log"

    env = os.environ.copy()
    env.setdefault("SYNNAX_LICENSE_KEY", "")

    if dev:
        cmd = [
            "go",
            "run",
            "-tags",
            "driver",
            "main.go",
            "start",
            "-i",
            "-d",
            str(DATA_DIR),
        ]
        cwd: str | None = str(CORE_DIR)
        log("  Starting Core (go run)...")
    else:
        assert binary is not None
        cmd = [str(binary), "start", "-i", "-d", str(DATA_DIR)]
        cwd = None
        log("  Starting Core...")

    with open(log_file, "w") as log_fh:
        proc = subprocess.Popen(
            cmd,
            stdout=log_fh,
            stderr=log_fh,
            env=env,
            cwd=cwd,
        )

    start = time.monotonic()
    while time.monotonic() - start < STARTUP_TIMEOUT:
        try:
            with socket.create_connection(("localhost", PORT), timeout=1):
                log("  Core ready")
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
    proc.terminate()
    try:
        proc.wait(timeout=STOP_TIMEOUT)
    except subprocess.TimeoutExpired:
        log("  Core did not stop gracefully, killing...")
        proc.kill()
        proc.wait(timeout=KILL_TIMEOUT)

    start = time.monotonic()
    while time.monotonic() - start < STOP_TIMEOUT:
        try:
            with socket.create_connection(("localhost", PORT), timeout=1):
                time.sleep(STARTUP_POLL_INTERVAL)
        except (ConnectionRefusedError, OSError):
            log("  Core stopped")
            return

    log(f"WARNING: Port {PORT} still in use after Core process exited")


def _resolve_latest_version() -> str:
    """Discover the latest release version from GitHub releases."""
    log("Discovering latest release version...")
    result = subprocess.run(
        [
            "gh",
            "release",
            "list",
            "--repo",
            REPO,
            "--json",
            "tagName",
            "--limit",
            "200",
            "--jq",
            ".[].tagName",
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    versions = []
    for tag in result.stdout.strip().splitlines():
        m = re.match(r"^synnax-v(\d+\.\d+\.\d+)$", tag)
        if m:
            versions.append(m.group(1))
    if not versions:
        raise RuntimeError("Could not find any release version")
    versions.sort(key=lambda v: tuple(int(p) for p in v.split(".")))
    latest = versions[-1]
    log(f"Resolved latest release: v{latest}")
    return latest


def _parse_version(version_str: str) -> tuple[int, int]:
    """Parse a version string like '0.55.0' into its major.minor tuple (0, 55)."""
    parts = version_str.split(".")
    return int(parts[0]), int(parts[1])


def _read_setup_version(script: Path) -> str | None:
    """Read the SETUP_VERSION constant from a setup script by importing it."""
    # Use spec_from_file_location because this module runs from integration/,
    # not inside a package, so importlib.import_module can't resolve the path.
    spec = importlib.util.spec_from_file_location(script.stem, script)
    if spec is None or spec.loader is None:
        return None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return getattr(module, "SETUP_VERSION", None)


def discover_setup_scripts(
    from_version: str,
) -> list[tuple[str, list[Path]]]:
    """Scan the setup directory for scripts with SETUP_VERSION <= from_version.

    Returns a sorted list of (version, [scripts]) tuples, grouped by version.
    """
    from_mm = _parse_version(from_version)
    by_version: dict[str, list[Path]] = {}
    if not SETUP_DIR.exists():
        return []
    for entry in sorted(SETUP_DIR.iterdir()):
        if not entry.is_file() or not entry.name.endswith("_setup.py"):
            continue
        version = _read_setup_version(entry)
        if version is None:
            continue
        if _parse_version(version) <= from_mm:
            by_version.setdefault(version, []).append(entry)

    return sorted(by_version.items(), key=lambda x: _parse_version(x[0]))


def _version_to_pip_spec(version: str) -> str:
    """Convert a version string to a pip version specifier.

    E.g., '0.54' -> 'synnax>=0.54,<0.55'
    """
    major, minor = _parse_version(version)
    return f"synnax>={major}.{minor},<{major}.{minor + 1}"


def _resolve_release_version(version: str) -> str:
    """Resolve a major.minor version to the latest patch release.

    E.g., '0.54' -> '0.54.2' (whatever the latest 0.54.x release is).
    """
    major, minor = _parse_version(version)
    result = subprocess.run(
        [
            "gh",
            "release",
            "list",
            "--repo",
            REPO,
            "--json",
            "tagName",
            "--limit",
            "200",
            "--jq",
            ".[].tagName",
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    prefix = f"synnax-v{major}.{minor}."
    patches = []
    for tag in result.stdout.strip().splitlines():
        if tag.startswith(prefix) and not tag.endswith("-rc"):
            patch = tag[len(prefix) :]
            if patch.isdigit():
                patches.append(int(patch))
    if not patches:
        raise RuntimeError(f"No releases found for v{major}.{minor}")
    latest = f"{major}.{minor}.{max(patches)}"
    log(f"  Resolved v{version} -> v{latest}")
    return latest


def create_setup_venv(version: str) -> Path:
    """Create an isolated venv and install the matching synnax client from PyPI.

    Returns the path to the venv's Python executable.
    """
    if CLIENT_VENV_DIR.exists():
        shutil.rmtree(CLIENT_VENV_DIR)

    pip_spec = _version_to_pip_spec(version)
    log(f"  Installing client {pip_spec}...")
    subprocess.run(
        ["uv", "venv", str(CLIENT_VENV_DIR)],
        check=True,
        capture_output=True,
    )

    python = _venv_python(CLIENT_VENV_DIR)
    subprocess.run(
        [
            "uv",
            "pip",
            "install",
            "--quiet",
            "--python",
            str(python),
            pip_spec,
            "pymodbus",
            "asyncua",
        ],
        check=True,
    )

    return python


def run_setup_script(script: Path, python: Path) -> None:
    """Run a setup script using the given venv python."""
    if not script.exists():
        raise FileNotFoundError(f"Setup script not found: {script}")

    result = subprocess.run(
        [str(python), str(script)],
        cwd=str(INTEGRATION_DIR),
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"Setup script {script.name} failed (exit code {result.returncode})"
        )


def run_verify() -> bool:
    """Run migration verify tests via the test conductor."""
    result = subprocess.run(
        ["uv", "run", "tc", "migration"],
        cwd=str(INTEGRATION_DIR),
    )
    if result.returncode != 0:
        log(f"Verify FAILED (exit code {result.returncode})")
        return False
    return True


def run(from_version: str, dev: bool) -> bool:
    """Run the full migration test chain.

    1. Discover setup scripts, group by SETUP_VERSION.
    2. For each version group: download Core, start it, install matching
       client in a venv, run all scripts for that version, stop Core.
    3. If from_version had no setup scripts, do a no-op Core start to
       trigger migrations.
    4. Start latest Core, run verify, stop Core.
    """
    setup_groups = discover_setup_scripts(from_version)
    if not setup_groups:
        log(f"WARNING: No setup scripts found for versions <= {from_version}")

    total_scripts = sum(len(scripts) for _, scripts in setup_groups)
    log(
        f"From: v{from_version} | Mode: {'dev' if dev else 'CI'} | {total_scripts} setup scripts"
    )
    for version, scripts in setup_groups:
        for s in scripts:
            log(f"  v{version} — {s.stem}")

    resolved_versions = {v: _resolve_release_version(v) for v, _ in setup_groups}
    needed_versions = set(resolved_versions.values())
    needed_versions.add(from_version)
    _prune_binaries(needed_versions)

    for version, scripts in setup_groups:
        release_version = resolved_versions[version]
        log(f"{'=' * 60}")
        log(f"Setup: v{version} ({len(scripts)} scripts)")
        log(f"{'=' * 60}")
        binary = install_version(release_version)
        proc = start_core(binary)
        try:
            python = create_setup_venv(version)
            for script in scripts:
                run_setup_script(script, python)
        finally:
            stop_core(proc)

    from_mm = _parse_version(from_version)
    already_ran = any(_parse_version(v) == from_mm for v, _ in setup_groups)
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
        proc = start_core(dev=True)
    else:
        latest_binary = CI_BINARY_DIR / BINARY_NAME
        if not latest_binary.exists():
            raise FileNotFoundError(
                f"Expected CI-built binary at {latest_binary}. "
                "Pass --dev to run from source instead."
            )
        proc = start_core(latest_binary)

    try:
        return run_verify()
    finally:
        stop_core(proc)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Migration test conductor",
    )
    parser.add_argument(
        "--from",
        dest="from_version",
        default="",
        help="Version to migrate from (e.g., '0.55.0'). Defaults to latest "
        "release. All setup folders with version <= this value are run "
        "before verification.",
    )
    parser.add_argument(
        "--dev",
        action="store_true",
        help="Use 'go run' from source for the latest Core instead of a "
        "pre-built CI binary.",
    )
    args = parser.parse_args()

    log("Migration conductor\n")

    from_version = args.from_version or _resolve_latest_version()

    _ensure_clean_state()

    success = False
    try:
        success = run(from_version, args.dev)
    finally:
        _ensure_clean_state()

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()

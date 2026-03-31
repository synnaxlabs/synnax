#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test orchestrator.

Manages Core binary lifecycle and invokes test-conductor as subprocesses.
Does not import any test framework code.

Usage:
    uv run migration-orchestrator --chain "0.50.0,0.53.0,latest" --test-type inplace --platform linux
    uv run migration-orchestrator --chain "0.50.0,latest" --test-type export_import --platform windows
"""

import argparse
import os
import platform
import shutil
import socket
import subprocess
import sys
from pathlib import Path

import synnax as sy
from xpy import get_platform

BINARY_CACHE_DIR = Path.home() / "synnax-binary-cache"
BINARY_DIR = Path.home() / "synnax-binaries"
DATA_DIR = Path.home() / "synnax-data"
LOG_DIR = Path.home() / "synnax-logs"
BINARY_NAME = "synnax.exe" if platform.system() == "Windows" else "synnax"
REPO = "synnaxlabs/synnax"
PORT = 9090
STARTUP_TIMEOUT = 30 * sy.TimeSpan.SECOND
STARTUP_POLL_INTERVAL = 1 * sy.TimeSpan.SECOND
STOP_TIMEOUT = 10 * sy.TimeSpan.SECOND
KILL_TIMEOUT = 5 * sy.TimeSpan.SECOND


def asset_name(version: str, plat: str) -> str:
    suffix = "-windows.exe" if plat == "windows" else f"-{plat}"
    return f"synnax-v{version}{suffix}"


def release_tag(version: str) -> str:
    return f"synnax-v{version}"


def install_version(version: str, plat: str) -> None:
    """Download and install a specific Core version binary."""
    if version == "latest":
        binary = BINARY_DIR / BINARY_NAME
        if not binary.exists():
            raise FileNotFoundError(
                f"Expected PR-built binary at {binary} but it does not exist"
            )
        print(f"Using PR-built binary at {binary}")
        return

    BINARY_CACHE_DIR.mkdir(parents=True, exist_ok=True)
    BINARY_DIR.mkdir(parents=True, exist_ok=True)

    asset = asset_name(version, plat)
    cached = BINARY_CACHE_DIR / asset
    target = BINARY_DIR / BINARY_NAME

    if cached.exists():
        print(f"Using cached binary for v{version}")
    else:
        tag = release_tag(version)
        print(f"Downloading {asset} from release {tag}...")
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
                str(BINARY_CACHE_DIR),
            ],
            check=True,
        )

    shutil.copy2(str(cached), str(target))
    if platform.system() != "Windows":
        target.chmod(0o755)
    print(f"Installed v{version} -> {target}")


def start_core() -> subprocess.Popen[bytes]:
    """Start Core binary with persistent storage, wait for port readiness."""
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    binary = BINARY_DIR / BINARY_NAME
    log_file = DATA_DIR / "synnax-core.log"

    env = os.environ.copy()
    # Ensure persistent storage (no -m flag)
    env.setdefault("SYNNAX_LICENSE_KEY", "")

    print(f"Starting Core from {binary}...")
    with open(log_file, "w") as log:
        proc = subprocess.Popen[bytes](
            [str(binary), "start", "-i"],
            cwd=str(DATA_DIR),
            stdout=log,
            stderr=log,
            env=env,
        )

    # Wait for port readiness
    timer = sy.Timer()
    while timer.elapsed() < STARTUP_TIMEOUT:
        try:
            with socket.create_connection(("localhost", PORT), timeout=1):
                print(f"Core is ready on port {PORT}")
                return proc
        except (ConnectionRefusedError, OSError):
            sy.sleep(STARTUP_POLL_INTERVAL)

    # Timeout - dump log and fail
    proc.kill()
    if log_file.exists():
        print("--- Core log ---")
        print(log_file.read_text()[-2000:])
        print("--- end log ---")
    raise TimeoutError(
        f"Core did not become ready on port {PORT} within {STARTUP_TIMEOUT}s"
    )


def stop_core(proc: subprocess.Popen[bytes]) -> None:
    """Stop Core process and wait for port to be released."""
    print("Stopping Core...")
    proc.terminate()
    try:
        proc.wait(timeout=STOP_TIMEOUT.seconds)
    except subprocess.TimeoutExpired:
        print("Core did not stop gracefully, killing...")
        proc.kill()
        proc.wait(timeout=KILL_TIMEOUT.seconds)

    timer = sy.Timer()
    while timer.elapsed() < STOP_TIMEOUT:
        try:
            with socket.create_connection(("localhost", PORT), timeout=1):
                sy.sleep(STARTUP_POLL_INTERVAL)
        except (ConnectionRefusedError, OSError):
            print("Core stopped")
            return

    print("Warning: port still in use after Core process exited")


def clean_data() -> None:
    """Remove the data directory for a fresh start."""
    if DATA_DIR.exists():
        shutil.rmtree(DATA_DIR)
        print(f"Cleaned data directory: {DATA_DIR}")


def run_test_conductor(class_filter: str) -> bool:
    """Run test-conductor for migration tests, filtering by class name."""
    cmd = ["uv", "run", "tc", "migration", "-f", class_filter]
    label = f"uv run tc migration -f {class_filter}"
    print(f"Running: {label}")
    result = subprocess.run(cmd, cwd=str(Path(__file__).parent))
    if result.returncode != 0:
        print(f"FAILED: {label} (exit code {result.returncode})")
        return False
    print(f"PASSED: {label}")
    return True


def run_phase(version: str, plat: str, sequence: str) -> bool:
    """Install a version, start Core, run a test sequence, and stop Core."""
    print(f"\n{'=' * 60}")
    print(f"Phase: {sequence} | Version: {version} | Platform: {plat}")
    print(f"{'=' * 60}\n")
    install_version(version, plat)
    proc = start_core()
    try:
        return run_test_conductor(sequence)
    finally:
        stop_core(proc)


def run_inplace(chain: list[str], plat: str) -> bool:
    """Run in-place upgrade test across a version chain."""
    print(f"\n{'#' * 60}")
    print(f"In-place upgrade chain: {' -> '.join(chain)}")
    print(f"{'#' * 60}\n")

    for i, version in enumerate(chain):
        if not run_phase(version, plat, "setup" if i == 0 else "verify"):
            return False

    return True


def run_export_import(chain: list[str], plat: str) -> bool:
    """Run export/import test: export on old version, import on new."""
    if len(chain) != 2:
        print(f"Export/import requires exactly 2 versions, got {len(chain)}: {chain}")
        return False

    old_version, new_version = chain
    print(f"\n{'#' * 60}")
    print(f"Export/import: {old_version} -> {new_version}")
    print(f"{'#' * 60}\n")

    clean_data()
    if not run_phase(old_version, plat, "export"):
        return False

    clean_data()
    if not run_phase(new_version, plat, "import"):
        return False

    return True


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Migration test orchestrator",
    )
    parser.add_argument(
        "--chain",
        required=True,
        help="Comma-separated version chain (e.g., '0.50.0,0.53.0,latest')",
    )
    parser.add_argument(
        "--test-type",
        required=True,
        choices=["inplace", "export_import"],
        help="Type of migration test to run",
    )
    parser.add_argument(
        "--platform",
        choices=["linux", "windows", "macos"],
        default=None,
        help="Target platform for binary downloads (auto-detected if omitted)",
    )

    args = parser.parse_args()
    chain = [v.strip() for v in args.chain.split(",")]
    plat = args.platform or get_platform()

    print(f"Migration orchestrator")
    print(f"  Chain: {chain}")
    print(f"  Test type: {args.test_type}")
    print(f"  Platform: {plat}")
    print()

    clean_data()

    try:
        if args.test_type == "inplace":
            success = run_inplace(chain, plat)
        else:
            success = run_export_import(chain, plat)
    finally:
        for d in [DATA_DIR, BINARY_CACHE_DIR, LOG_DIR]:
            if not d.exists():
                continue
            timer = sy.Timer()
            while timer.elapsed() < STOP_TIMEOUT:
                try:
                    shutil.rmtree(d)
                    break
                except PermissionError:
                    sy.sleep(1)
            else:
                print(f"Failed to clean {d} after {STOP_TIMEOUT}")
        print("Cleanup complete")

    if success:
        print("\nMigration test PASSED")
        sys.exit(0)
    else:
        print("\nMigration test FAILED")
        sys.exit(1)


if __name__ == "__main__":
    main()

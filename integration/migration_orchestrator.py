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
    uv run migration-orchestrator --chain "0.50.0,0.53.0,latest"
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
BINARY_NAME = "synnax.exe" if platform.system() == "Windows" else "synnax"
REPO = "synnaxlabs/synnax"
PORT = 9090
STARTUP_TIMEOUT = 30 * sy.TimeSpan.SECOND
STARTUP_POLL_INTERVAL = 1 * sy.TimeSpan.SECOND
STOP_TIMEOUT = 10 * sy.TimeSpan.SECOND
KILL_TIMEOUT = 5 * sy.TimeSpan.SECOND
CLEANUP_TIMEOUT = 10 * sy.TimeSpan.SECOND


def install_version(version: str) -> None:
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

    plat = get_platform()
    suffix = "-windows.exe" if plat == "windows" else f"-{plat}"
    asset = f"synnax-v{version}{suffix}"
    cached = BINARY_CACHE_DIR / asset
    target = BINARY_DIR / BINARY_NAME

    if cached.exists():
        print(f"Using cached binary for v{version}")
    else:
        tag = f"synnax-v{version}"
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
    env.setdefault("SYNNAX_LICENSE_KEY", "")

    print(f"Starting Core from {binary}...")
    log = open(log_file, "w")
    proc = subprocess.Popen(
        [str(binary), "start", "-i"],
        cwd=str(DATA_DIR),
        stdout=log,
        stderr=log,
        env=env,
    )
    log.close()

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

    raise RuntimeError(f"Port {PORT} still in use after Core process exited")


def clean_data() -> None:
    """Remove the data directory for a fresh start."""
    if DATA_DIR.exists():
        shutil.rmtree(DATA_DIR)
        print(f"Cleaned data directory: {DATA_DIR}")


def run_test_conductor(class_filter: str) -> bool:
    """Run test-conductor for migration tests, filtering by class name.

    The -f flag matches against both file paths and class names, so passing
    "setup" matches classes like ChannelsSetup, WorkspacesSetup, etc.
    """
    cmd = ["uv", "run", "tc", "migration", "-f", class_filter]
    label = f"uv run tc migration -f {class_filter}"
    print(f"Running: {label}")
    result = subprocess.run(cmd, cwd=str(Path(__file__).parent))
    if result.returncode != 0:
        print(f"FAILED: {label} (exit code {result.returncode})")
        return False
    print(f"PASSED: {label}")
    return True


def run(chain: list[str]) -> bool:
    """Run upgrade test across a version chain.

    For each version: install the binary, start Core, run the test conductor
    (setup for the first version, verify for subsequent ones), and stop Core.
    Data directory persists across all versions so each upgrade verifies
    against the previous version's data. Caller is responsible for cleaning
    data before and after.
    """
    print(f"\n{'#' * 60}")
    print(f"Upgrade chain: {' -> '.join(chain)}")
    print(f"{'#' * 60}\n")

    for i, version in enumerate(chain):
        sequence = "setup" if i == 0 else "verify"
        print(f"\n{'=' * 60}")
        print(f"Phase: {sequence} | Version: {version}")
        print(f"{'=' * 60}\n")
        install_version(version)
        proc = start_core()
        try:
            if not run_test_conductor(sequence):
                return False
        finally:
            stop_core(proc)

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
    args = parser.parse_args()
    chain = [v.strip() for v in args.chain.split(",")]

    print("Migration orchestrator")
    print(f"  Chain: {' -> '.join(chain)}")
    print()

    success = False
    clean_data()

    try:
        success = run(chain)
    finally:
        for d in [DATA_DIR, BINARY_CACHE_DIR]:
            if not d.exists():
                continue
            timer = sy.Timer()
            while timer.elapsed() < CLEANUP_TIMEOUT:
                try:
                    shutil.rmtree(d)
                    print(f"Cleaned {d}")
                    break
                except PermissionError:
                    sy.sleep(1)
            else:
                print(f"Warning: failed to clean {d} after {CLEANUP_TIMEOUT}")
        print("Cleanup complete")

    if success:
        sys.exit(0)
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()

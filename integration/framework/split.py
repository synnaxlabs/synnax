#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Starts Cores on one machine and runs a test target split across them at once.

Each part of a split runs in its own test conductor against its own Core, so tests that
share Core state do not clash. Tests that share a machine-wide resource (a simulator
port, the NI devices) must sit in the same part. Only the first Core runs the NI
integration.
"""

import argparse
import os
import shlex
import socket
import subprocess
import sys
import threading
import time
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import IO

from framework.config_client import ConfigClient
from framework.test_conductor import build_parser, filter_from_args

READY_TIMEOUT_S = 60


@dataclass(frozen=True)
class Split:
    """A test target divided into parts that run at once, each on its own Core."""

    parts: tuple[str, ...]
    """Test conductor arguments of each part. Together they select every test of the
    target exactly once."""
    after: str | None = None
    """Test conductor arguments of tests that run on the first Core after every part
    finishes."""


SPLITS: dict[str, Split] = {
    # Every Arc case runs once per rack: 65537 on the embedded Driver, 65538 on the
    # Core's own rack. A latency case uses NI, so latency stays on the first Core.
    "arc": Split(parts=("arc,latency -x _65538", "arc,control -x _65537")),
    # The project sequence uses the OPC UA simulator and pages holds the NI forms.
    "console": Split(
        parts=(
            "console/project,pages/...",
            "console/lifecycle,channel,user,cluster,other/...",
        )
    ),
    # A simulator port belongs to one part. grand_finale uses OPC UA, Modbus, and NI.
    "driver": Split(
        parts=("driver/ni_,modbus", "driver/opcua,http,pagerduty"),
        after="driver/grand_finale",
    ),
}


def _tests(arguments: str) -> list[str]:
    args = build_parser().parse_args(shlex.split(arguments))
    _, definitions = ConfigClient().load(filter_from_args(args))
    return [str(d) for d in definitions]


def check(target: str, split: Split) -> list[str]:
    """Checks that the parts and after-tests of a split select every test of the
    target exactly once.

    :param target: Test conductor arguments selecting every test of the run.
    :param split: The split to check.
    :returns: A description of each test that is missing, extra, or selected twice.
    """
    expected = Counter(_tests(target))
    actual: Counter[str] = Counter()
    for arguments in (*split.parts, *([split.after] if split.after else [])):
        actual.update(_tests(arguments))
    problems = [f"missing: {t}" for t in expected if t not in actual]
    problems += [f"not in the target: {t}" for t in actual if t not in expected]
    problems += [f"selected {n} times: {t}" for t, n in actual.items() if n > 1]
    return problems


def _start_core(
    binary: Path, data: Path, index: int, port: int
) -> subprocess.Popen[bytes]:
    directory = data / f"core-{index}"
    directory.mkdir(parents=True, exist_ok=True)
    command = [str(binary), "start", "-mi", "-l", f"localhost:{port}"]
    command += ["-d", str(directory)]
    if index > 0:
        command += ["--disable-integrations", "ni"]
    log = open(directory / "core.log", "wb")
    return subprocess.Popen(
        command, stdout=log, stderr=subprocess.STDOUT, cwd=directory
    )


def _wait_ready(core: subprocess.Popen[bytes], port: int) -> None:
    deadline = time.monotonic() + READY_TIMEOUT_S
    while time.monotonic() < deadline:
        if core.poll() is not None:
            raise RuntimeError(
                f"Core on port {port} exited with code {core.returncode}"
            )
        try:
            socket.create_connection(("localhost", port), timeout=1).close()
            return
        except OSError:
            time.sleep(1)
    raise RuntimeError(f"Core on port {port} not ready after {READY_TIMEOUT_S} s")


def _prefix(stream: IO[str], label: str) -> None:
    for line in stream:
        print(f"[{label}] {line}", end="", flush=True)


def _conduct(runs: list[tuple[str, str, int, Path]]) -> list[int]:
    """Runs a conductor for each (name, arguments, port, server log) at once and
    returns their exit codes."""
    integration = Path(__file__).resolve().parent.parent
    prefixed = len(runs) > 1
    processes: list[subprocess.Popen[str]] = []
    readers: list[threading.Thread] = []
    for name, arguments, port, log in runs:
        command = [sys.executable, "-m", "framework.test_conductor"]
        command += [*shlex.split(arguments), "--name", name, "--port", str(port)]
        process = subprocess.Popen(
            command,
            cwd=integration,
            env={**os.environ, "SYNNAX_SERVER_LOG": str(log)},
            stdout=subprocess.PIPE if prefixed else None,
            stderr=subprocess.STDOUT if prefixed else None,
            text=True,
            encoding="utf-8",
            errors="replace",
        )
        processes.append(process)
        if prefixed and process.stdout is not None:
            reader = threading.Thread(target=_prefix, args=(process.stdout, name))
            reader.start()
            readers.append(reader)
    codes = [p.wait() for p in processes]
    for reader in readers:
        reader.join()
    return codes


def main() -> None:
    """Runs a test target, split across Cores when a split is named."""
    executable = "synnax.exe" if sys.platform == "win32" else "synnax"
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "split",
        nargs="?",
        choices=sorted(SPLITS),
        help="Split to run the target with. Without one, the target runs on one Core.",
    )
    parser.add_argument(
        "--target",
        required=True,
        help="Test conductor arguments selecting every test of the run.",
    )
    parser.add_argument("--name", default="tc", help="Test conductor name")
    parser.add_argument(
        "--binary",
        type=Path,
        default=Path.home() / "synnax-binaries" / executable,
        help="Core binary",
    )
    parser.add_argument(
        "--data",
        type=Path,
        default=Path.home() / "synnax-data",
        help="Directory holding a data directory and log for each Core",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=9090,
        help="Port of the first Core; each next Core takes the next port",
    )
    args = parser.parse_args()

    split = SPLITS[args.split] if args.split else Split(parts=(args.target,))
    if args.split:
        problems = check(args.target, split)
        if problems:
            print(f"Split '{args.split}' does not match '{args.target}':")
            print("\n".join(f"  {p}" for p in problems))
            sys.exit(1)

    if not args.binary.exists():
        sys.exit(f"Core binary not found at {args.binary}")
    subprocess.run([str(args.binary), "version"], check=True)

    ports = [args.port + i for i in range(len(split.parts))]
    logs = [args.data / f"core-{i}" / "core.log" for i in range(len(split.parts))]
    cores = [
        _start_core(args.binary, args.data, i, port) for i, port in enumerate(ports)
    ]
    try:
        for core, port in zip(cores, ports):
            _wait_ready(core, port)
        names = (
            [args.name]
            if len(cores) == 1
            else [f"{args.name}_{i}" for i in range(len(cores))]
        )
        codes = _conduct(list(zip(names, split.parts, ports, logs)))
        if split.after:
            codes += _conduct([(f"{args.name}_after", split.after, ports[0], logs[0])])
    finally:
        for core in cores:
            core.terminate()
        for core in cores:
            try:
                core.wait(timeout=10)
            except subprocess.TimeoutExpired:
                core.kill()
    sys.exit(0 if all(c == 0 for c in codes) else 1)


if __name__ == "__main__":
    main()

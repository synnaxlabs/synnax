#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Runs a CI job's tests split across Cores that run at once on one machine (see
``jobs.py``)."""

import argparse
import os
import shlex
import socket
import subprocess
import sys
import threading
import time
from collections import Counter
from pathlib import Path
from typing import IO

from framework.config_client import ConfigClient
from framework.test_conductor import build_parser, filter_from_args
from jobs import EXEMPT, JOBS, Job

READY_TIMEOUT_S = 60
TESTS = Path(__file__).resolve().parent.parent / "tests"


def _tests(arguments: str) -> list[str]:
    args = build_parser().parse_args(shlex.split(arguments))
    _, definitions = ConfigClient().load(filter_from_args(args))
    return [str(d) for d in definitions]


def check() -> list[str]:
    """Checks that the jobs together run every test of every test file outside
    EXEMPT exactly once.

    :returns: A description of each test that is missing, extra, or run twice.
    """
    prefixes = sorted(
        p.name.removesuffix("_tests.json") for p in TESTS.glob("*_tests.json")
    )
    target = ",".join(p for p in prefixes if p not in EXEMPT)
    expected = Counter(_tests(target))
    actual: Counter[str] = Counter()
    for job in JOBS.values():
        for arguments in (*(a for part in job.parts for a in part), *job.after):
            actual.update(_tests(arguments))
    problems = [f"missing: {t}" for t in expected if t not in actual]
    problems += [f"not in a test file: {t}" for t in actual if t not in expected]
    problems += [f"run {n} times: {t}" for t, n in actual.items() if n > 1]
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


def _conduct(runs: list[tuple[str, str]], port: int, log: Path, prefixed: bool) -> int:
    """Runs each (name, arguments) conductor in turn against one Core, returning 0 if
    every run passes and 1 otherwise."""
    integration = Path(__file__).resolve().parent.parent
    failed = False
    for name, arguments in runs:
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
        if process.stdout is not None:
            _prefix(process.stdout, name)
        failed |= process.wait() != 0
    return 1 if failed else 0


def _prefix(stream: IO[str], label: str) -> None:
    for line in stream:
        print(f"[{label}] {line}", end="", flush=True)


def _run(job: Job, name: str, binary: Path, data: Path, base_port: int) -> int:
    ports = [base_port + i for i in range(len(job.parts))]
    logs = [data / f"core-{i}" / "core.log" for i in range(len(job.parts))]
    single = len(job.parts) == 1 and len(job.parts[0]) == 1 and not job.after
    lanes = [
        [(name if single else f"{name}_{i}_{k}", a) for k, a in enumerate(part)]
        for i, part in enumerate(job.parts)
    ]
    cores = [_start_core(binary, data, i, port) for i, port in enumerate(ports)]
    codes = [1] * len(lanes)
    try:
        for core, port in zip(cores, ports):
            _wait_ready(core, port)

        def lane(i: int) -> None:
            codes[i] = _conduct(lanes[i], ports[i], logs[i], not single)

        threads = [threading.Thread(target=lane, args=(i,)) for i in range(len(lanes))]
        for thread in threads:
            thread.start()
        for thread in threads:
            thread.join()
        after = [(f"{name}_after_{k}", a) for k, a in enumerate(job.after)]
        codes.append(_conduct(after, ports[0], logs[0], not single))
    finally:
        for core in cores:
            core.terminate()
        for core in cores:
            try:
                core.wait(timeout=10)
            except subprocess.TimeoutExpired:
                core.kill()
    return max(codes)


def main() -> None:
    """Runs a CI job, or a custom target on one Core."""
    executable = "synnax.exe" if sys.platform == "win32" else "synnax"
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("job", nargs="?", choices=sorted(JOBS), help="CI job to run")
    parser.add_argument(
        "--target", help="Test conductor arguments to run on one Core instead of a job"
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
    if (args.job is None) == (args.target is None):
        parser.error("give either a job or --target")

    if args.job:
        problems = check()
        if problems:
            print("The CI jobs do not run every test exactly once:")
            print("\n".join(f"  {p}" for p in problems))
            sys.exit(1)
        job = JOBS[args.job]
    else:
        job = Job(parts=((args.target,),))

    if not args.binary.exists():
        sys.exit(f"Core binary not found at {args.binary}")
    subprocess.run([str(args.binary), "version"], check=True)
    sys.exit(_run(job, args.name, args.binary, args.data, args.port))


if __name__ == "__main__":
    main()

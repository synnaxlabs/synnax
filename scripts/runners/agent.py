#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# /// script
# requires-python = ">=3.12,<4"
# dependencies = ["psutil", "synnax>=0.61,<0.62"]
# ///

"""Writes this machine's CPU, memory, and disk usage to Synnax every 5 seconds.

Runs on each runner instance, started at boot by the runner-agent systemd unit
(Ubuntu) or the RunnerAgent scheduled task (Windows), both deployed over SSM from
scripts/gen_fleet.py. Four float32 channels on one index, all created by
controller.py:

    <runner>_cpu           percent CPU used over the last interval
    <runner>_mem           percent of RAM in use
    <runner>_disk_free     free GiB on the root volume
    <runner>_disk_percent  percent of the root volume in use
    <runner>_sys_time      their index

The actions-runner page logs are tailed into the virtual string channel
<runner>_log; --no-runner skips it. The runner name comes from the actions-runner
.runner file, or the copy cached from the last read when the file is gone; a machine
with neither waits for it and exits only once the wait runs out (a spare machine).

Nothing else ends it: a bad write reopens the writer, three in a row drop the
session, and the next session connects again from scratch. Runs at the OS's lowest
CPU and IO priority and exits if another copy is already running.

The GitHub runner is started and stopped by <runner>_gh_cmd. A stop waits for the
running job to finish. One minute without Synnax at boot starts the runner anyway.

    uv run agent.py
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Protocol

import numpy as np
import psutil
import synnax as sy

from agent_log import RUNNER_DIRS, Log
from gh_runner import Runner

INTERVAL = 5 * sy.TimeSpan.SECOND
# The runner starts anyway if Synnax stays unreachable this long after boot.
CONNECT_GRACE = 1 * sy.TimeSpan.MINUTE
RETRY = 30 * sy.TimeSpan.SECOND
REOPEN_DELAY = 5 * sy.TimeSpan.SECOND
MAX_WRITE_FAILURES = 3
SESSION_DELAY = 15 * sy.TimeSpan.SECOND
REGISTER_WAIT = 10 * sy.TimeSpan.MINUTE
REGISTER_RETRY = 15 * sy.TimeSpan.SECOND
RUNNER_FILES = tuple(p / ".runner" for p in RUNNER_DIRS)
# The name last read from .runner, for a boot where the file is gone.
NAME_CACHE = Path(__file__).with_name("runner_name.txt")
# Verifies the Core's certificate when the deploy puts the file beside this script.
CA_CERT = Path(__file__).with_name("ca.crt")
ROOT = "C:\\" if sys.platform == "win32" else "/"
SCRIPT = Path(__file__).name
ABBREVIATIONS = {"windows": "win", "build_bot": "build", "tester": "test"}
# Measured values cannot be taken over and spoofed.
PUBLISH_AUTHORITY = sy.Authority.ABSOLUTE


def channel_name(source: str, runner: str) -> str:
    """Keep in sync with common.channel_name; runner/ deploys alone."""
    name = re.sub(r"[^A-Za-z0-9]+", "_", runner)
    for long, short in ABBREVIATIONS.items():
        name = name.replace(long, short)
    return f"{name}_{source}"


def warn(what: str, e: Exception) -> None:
    print(f"{what} failed: {type(e).__name__}: {e}", flush=True)


class Closeable(Protocol):
    def close(self) -> object: ...


def close_quietly(resource: Closeable | None) -> None:
    if resource is None:
        return
    try:
        resource.close()
    except Exception:
        pass


def runner_name() -> str | None:
    """Reads agentName from .runner, else the name cached from the last read, else
    waits up to REGISTER_WAIT for the file (a first boot still registering)."""
    timer = sy.Timer()
    while True:
        for path in RUNNER_FILES:
            try:
                name = json.loads(path.read_text(encoding="utf-8-sig"))["agentName"]
            except (OSError, KeyError, ValueError):
                continue
            try:
                NAME_CACHE.write_text(name)
            except OSError as e:
                warn("name cache", e)
            return name
        try:
            if name := NAME_CACHE.read_text().strip():
                return name
        except OSError:
            pass
        if timer.elapsed() >= REGISTER_WAIT:
            return None
        sy.sleep(REGISTER_RETRY)


def already_running() -> bool:
    """True when another python holds this script, ignoring this process and its
    ancestors (uv and any interpreter wrapper carry the script name too)."""
    me = psutil.Process()
    ancestors = {me.pid} | {p.pid for p in me.parents()}
    for p in psutil.process_iter(["pid", "name", "cmdline"]):
        if p.info["pid"] in ancestors:
            continue
        if "python" not in (p.info["name"] or "").lower():
            continue
        if any(SCRIPT in part for part in (p.info["cmdline"] or [])):
            return True
    return False


def deprioritize() -> None:
    p = psutil.Process()
    try:
        if sys.platform == "win32":
            p.nice(psutil.IDLE_PRIORITY_CLASS)
        else:
            p.nice(19)
        if hasattr(p, "ionice"):
            p.ionice(
                psutil.IOPRIO_VERYLOW
                if sys.platform == "win32"
                else psutil.IOPRIO_CLASS_IDLE
            )
    except Exception as e:
        warn("deprioritize", e)


def connect(
    args: argparse.Namespace, names: dict[str, str], control: Runner | None
) -> sy.Synnax:
    """Connects and resolves the channels, retrying forever; a booting machine must not
    depend on anything but the Core coming back, and controller.py creating this
    runner's channels. A runner never started yet starts after CONNECT_GRACE."""
    wanted = set(names.values())
    if args.no_runner:
        wanted -= {names["log"], names["cmd"], names["state"]}
    timer = sy.Timer()
    while True:
        if (
            control is not None
            and control.state is None
            and timer.elapsed() >= CONNECT_GRACE
        ):
            control.apply(True, "no Synnax")
        client = None
        try:
            client = sy.Synnax(
                host=args.host,
                port=args.port,
                username=args.username,
                password=args.password,
                secure=args.secure,
                ca_cert=str(CA_CERT) if args.secure and CA_CERT.exists() else None,
            )
            found = {c.name for c in client.channels.retrieve(sorted(wanted))}
            if missing := wanted - found:
                raise LookupError(f"channels not created yet: {sorted(missing)}")
            return client
        except Exception as e:
            warn("startup", e)
            close_quietly(client)
            sy.sleep(RETRY)


def run(
    args: argparse.Namespace, names: dict[str, str], runner: str, control: Runner | None
) -> None:
    """One connected session, sampling until something breaks. Returning hands control
    back to main(), which connects again; nothing here is fatal."""
    client = connect(args, names, control)
    print(f"reporting {sorted(names.values())}", flush=True)
    psutil.cpu_percent(None)
    loop = sy.Loop(INTERVAL)
    writer = None
    failures = 0
    log = None
    if control is not None:
        log = Log(client, names["log"])
        log.log("agent connected")
        control.listen(client, names["cmd"], names["state"], log)
    last = sy.TimeStamp.now()
    channels = [
        names["index"],
        names["cpu"],
        names["mem"],
        names["disk"],
        names["disk_percent"],
    ]
    try:
        while loop.wait():
            disk = psutil.disk_usage(ROOT)
            values = {
                names["cpu"]: np.float32(psutil.cpu_percent(None)),
                names["mem"]: np.float32(psutil.virtual_memory().percent),
                names["disk"]: np.float32(disk.free / 1024**3),
                names["disk_percent"]: np.float32(disk.percent),
            }
            try:
                if writer is None:
                    start = sy.TimeStamp.now()
                    writer = client.open_writer(
                        start, channels, authorities=PUBLISH_AUTHORITY, name=runner
                    )
                    # Samples must land strictly after the writer's start.
                    last = max(last, start)
                last = max(sy.TimeStamp.now(), last + sy.TimeSpan(1))
                frame = {names["index"]: last, **values}
                writer.write(frame)  # pyright: ignore[reportArgumentType]
                failures = 0
            except Exception as e:
                warn("write", e)
                close_quietly(writer)
                writer = None
                failures += 1
                # A Core restart kills this client's auth token; reopening on the
                # same client can never succeed, so drop the session.
                if failures >= MAX_WRITE_FAILURES:
                    raise
                sy.sleep(REOPEN_DELAY)
            if control is None or log is None:
                continue
            control.tick()
            try:
                log.write()
            except Exception as e:
                warn("log", e)
                close_quietly(log)
                # Reopening talks to the Core, so a failure here ends the session
                # rather than the process.
                queued = log.queue
                log = Log(client, names["log"])
                log.queue = queued
                control.sink = log
    finally:
        if control is not None:
            control.stop_listening()
        close_quietly(writer)
        close_quietly(log)
        close_quietly(client)


def main() -> None:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    p.add_argument("--host", required=True)
    p.add_argument("--port", type=int, default=9090)
    p.add_argument("--username", default="synnax")
    p.add_argument("--password", required=True)
    p.add_argument("--secure", action="store_true", help="connect over TLS")
    p.add_argument("--runner-name", default=None, help="override the .runner name")
    p.add_argument(
        "--no-runner", action="store_true", help="no GitHub runner here; usage only"
    )
    args = p.parse_args()
    if already_running():
        print("another copy is running; exiting", flush=True)
        return
    deprioritize()
    runner = args.runner_name or runner_name()
    if runner is None:
        print("no .runner file after waiting; not a runner", flush=True)
        return
    names = {
        "index": channel_name("sys_time", runner),
        "cpu": channel_name("cpu", runner),
        "mem": channel_name("mem", runner),
        "disk": channel_name("disk_free", runner),
        "disk_percent": channel_name("disk_percent", runner),
        "log": channel_name("log", runner),
        "cmd": channel_name("gh_cmd", runner),
        "state": channel_name("gh_state", runner),
    }
    control = None if args.no_runner else Runner()
    while True:
        try:
            run(args, names, runner, control)
        except Exception as e:
            warn("session", e)
        sy.sleep(SESSION_DELAY)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        os._exit(130)

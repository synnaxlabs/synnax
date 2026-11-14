#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Starts and stops the GitHub runner on this machine, for agent.py."""

import subprocess
import sys
import threading
import time
from typing import Protocol

import psutil
import synnax as sy

SERVICE = "github-runner.service"
TASK = "GitHubActionsRunner"
SERVICE_TIMEOUT = 60  # seconds, for subprocess.run
READ_TIMEOUT = 60 * sy.TimeSpan.SECOND
LISTENER = "runner.listener"
# Runner.Listener spawns this for the length of a job.
WORKER = "runner.worker"
# A start that died is retried this long after the last attempt.
RETRY = 60
LOOKBACKS = (1 * sy.TimeSpan.DAY, 7 * sy.TimeSpan.DAY)
# A command stream that failed is opened again after this many seconds.
LISTEN_RETRY = 5


class Sink(Protocol):
    def log(self, line: str) -> None: ...


class Runner:
    def __init__(self) -> None:
        self.sink: Sink | None = None
        self.state: bool | None = None
        # A stop held back while a job runs.
        self.pending: bool | None = None
        self.attempted = 0.0
        self.stopping = threading.Event()
        self.thread: threading.Thread | None = None

    def listen(self, client: sy.Synnax, name: str, state: str, sink: Sink) -> None:
        """Applies the last state, then each write to the command channel from a
        thread until stop_listening()."""
        self.sink = sink
        # read_latest misses the latest write while its writer is open.
        for span in LOOKBACKS:
            now = sy.TimeStamp.now()
            stored = client.read(sy.TimeRange(now - span, now), state)
            if len(stored):
                self.apply(bool(stored[-1]), "stored")
                break
        else:
            self.apply(True, "none yet")
        # A streamer may label frames by channel name or key.
        labels = (name, client.channels.retrieve(name).key)
        # Each thread has its own event, so a new session cannot revive an old thread.
        self.stopping = threading.Event()
        self.thread = threading.Thread(
            target=self._listen, args=(client, name, labels, self.stopping), daemon=True
        )
        self.thread.start()

    def _listen(
        self,
        client: sy.Synnax,
        name: str,
        labels: tuple[str, int],
        stopping: threading.Event,
    ) -> None:
        streamer = None
        while not stopping.is_set():
            try:
                if streamer is None:
                    streamer = client.open_streamer([name])
                frame = streamer.read(timeout=READ_TIMEOUT)
                if frame is None:
                    continue
                for label in labels:
                    series = frame.get(label)
                    if series is not None and len(series):
                        self.apply(bool(series[-1]), "command")
                        break
            except Exception as e:
                self.log(f"commands failed: {type(e).__name__}: {e}")
                if streamer is not None:
                    try:
                        streamer.close()
                    except Exception:
                        pass
                streamer = None
                stopping.wait(LISTEN_RETRY)
        if streamer is not None:
            try:
                streamer.close()
            except Exception:
                pass

    def stop_listening(self) -> None:
        self.stopping.set()

    def log(self, line: str) -> None:
        if self.sink is None:
            print(line, flush=True)
        else:
            self.sink.log(line)

    def process(self, name: str) -> bool:
        return any(
            (p.info["name"] or "").lower().startswith(name)
            for p in psutil.process_iter(["name"])
        )

    def busy(self) -> bool:
        return self.process(WORKER)

    def tick(self) -> None:
        if self.pending is not None and not self.busy():
            self.apply(self.pending, "job done")
        elif (
            self.state
            and time.monotonic() - self.attempted >= RETRY
            and not self.process(LISTENER)
        ):
            self.state = None
            self.apply(True, "not running")

    def apply(self, enabled: bool, why: str) -> None:
        verb, doing, done = (
            ("start", "starting", "started")
            if enabled
            else ("stop", "stopping", "stopped")
        )
        self.pending = None
        if enabled == self.process(LISTENER):
            self.state = enabled
            self.log(f"gh_cmd={enabled} ({why}): runner already {done}")
            return
        if not enabled and self.busy():
            self.pending = False
            self.log(f"gh_cmd=False ({why}): job running, stopping after it")
            return
        self.log(f"gh_cmd={enabled} ({why}): {doing} runner")
        self.attempted = time.monotonic()
        if sys.platform == "win32":
            quiet = "-ErrorAction SilentlyContinue"
            script = (
                f"Start-ScheduledTask -TaskName '{TASK}'"
                if enabled
                # Stopping the task alone leaves Runner.Listener up.
                else f"Stop-ScheduledTask -TaskName '{TASK}' {quiet}; "
                f"Stop-Process -Name 'Runner.Listener' -Force {quiet}"
            )
            cmd = ["powershell", "-NoProfile", "-NonInteractive", "-Command", script]
        else:
            cmd = ["systemctl", verb, SERVICE]
        try:
            out = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=SERVICE_TIMEOUT,
                check=False,
            )
        except Exception as e:
            self.log(f"runner {verb} failed: {type(e).__name__}: {e}")
            return
        # A stop is judged by the listener being gone; its exit code is noise.
        if out.returncode and (enabled or self.process(LISTENER)):
            err = (out.stderr or out.stdout).strip().splitlines()[-1:]
            self.log(f"runner {verb} failed rc={out.returncode}: {' '.join(err)}")
            return
        self.state = enabled
        self.log(f"runner {done}")

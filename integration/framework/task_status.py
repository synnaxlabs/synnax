#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Headless task-status matching for integration tests.

The driver and Arc runtime report per-task execution state (running, configure
errors, runtime warnings) on the cluster status-set channel, keyed by task key.
The Console surfaces this state in the task controls panel; tests that would
otherwise read it through Playwright can use TaskStatus, which subscribes to the
status-set channel directly through the Python client and filters by task key.

This is generic across every task type (Arc, NI, Modbus, OPC UA, LabJack) since
they all publish ``task.Status`` with ``details.task`` set to their own key.

Statuses are transient, so begin recording before the action whose status should
be observed. TestCase does this automatically for cases that set
collect_task_status; standalone callers can use the context manager form.
"""

import logging
import threading

from pydantic import ValidationError

import synnax as sy
from synnax.status import SET_CHANNEL

logger = logging.getLogger("synnax.task_status")

OPEN_TIMEOUT = 5.0
CLOSE_TIMEOUT = 3.0
DEFAULT_WAIT_TIMEOUT = 5.0
READ_TIMEOUT = 250 * sy.TimeSpan.MILLISECOND


class TaskStatus:
    """Records task status updates published on the cluster status-set channel.

    A background thread drains the status-set channel from open() until close(),
    keeping the latest status and the full message history per task key. wait_for
    scans the recorded history, so a status matches whether it arrived before or
    after the call. A condition variable guards the state and wakes wait_for the
    moment a new status is recorded, so it blocks rather than polls.
    """

    def __init__(self, client: sy.Synnax) -> None:
        self._client = client
        self._stop = threading.Event()
        self._ready = threading.Event()
        self._cond = threading.Condition()
        self._latest: dict[int, sy.task.Status] = {}
        self._messages: dict[int, list[str]] = {}
        self._thread: threading.Thread | None = None
        self._error: Exception | None = None

    def __enter__(self) -> "TaskStatus":
        self.open()
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    def open(self) -> None:
        """Begin recording in the background, blocking until the stream is live.

        Blocking until live ensures a status raised by the next action is not
        missed.
        """
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()
        if not self._ready.wait(timeout=OPEN_TIMEOUT):
            raise TimeoutError("task status stream did not open in time")
        if self._error is not None:
            raise self._error

    def close(self) -> None:
        """Stop recording and tear down the subscription."""
        self._stop.set()
        if self._thread is not None:
            self._thread.join(timeout=CLOSE_TIMEOUT)

    def wait_for(
        self, task_key: int, text: str, timeout: float = DEFAULT_WAIT_TIMEOUT
    ) -> bool:
        """Return True if a status for task_key contains text within timeout.

        Blocks on the condition variable until a matching status is recorded or
        timeout elapses, waking only when the recorder appends a new status.
        """
        timer = sy.Timer()
        with self._cond:
            while True:
                if any(text in m for m in self._messages.get(task_key, [])):
                    return True
                remaining = timeout - timer.elapsed().seconds
                if remaining <= 0:
                    return False
                self._cond.wait(remaining)

    def wait_for_clear(
        self, task_key: int, text: str, timeout: float = DEFAULT_WAIT_TIMEOUT
    ) -> bool:
        """Return True once the latest recorded status stops containing text.

        Inspects only the most recent status, so it must follow an appearance
        assertion; a task that never emitted a status is not treated as cleared.
        """
        timer = sy.Timer()
        with self._cond:
            while True:
                latest = self._latest.get(task_key)
                if latest is not None and text not in latest.message:
                    return True
                remaining = timeout - timer.elapsed().seconds
                if remaining <= 0:
                    return False
                self._cond.wait(remaining)

    def latest(self, task_key: int) -> sy.task.Status | None:
        """Return the most recent recorded status for task_key, or None."""
        with self._cond:
            return self._latest.get(task_key)

    def is_running(self, task_key: int) -> bool:
        """Return True if the most recent status reports the task as running."""
        status = self.latest(task_key)
        return (
            status is not None and status.details is not None and status.details.running
        )

    def _run(self) -> None:
        try:
            with self._client.open_streamer([SET_CHANNEL]) as streamer:
                self._ready.set()
                while not self._stop.is_set():
                    frame = streamer.read(timeout=READ_TIMEOUT)
                    if frame is None or SET_CHANNEL not in frame:
                        continue
                    for raw in frame[SET_CHANNEL]:
                        try:
                            status = sy.task.Status.model_validate(raw)
                        except ValidationError:
                            # The channel carries statuses for all tasks and
                            # racks. Non-task statuses (and rack statuses with a
                            # different detail schema) fail validation and are
                            # skipped.
                            continue
                        if status.details is None:
                            continue
                        with self._cond:
                            key = status.details.task
                            self._latest[key] = status
                            self._messages.setdefault(key, []).append(status.message)
                            self._cond.notify_all()
        except Exception as e:
            if not self._ready.is_set():
                self._error = e
                self._ready.set()
            else:
                logger.error("task status recorder stopped", exc_info=e)

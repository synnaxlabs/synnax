#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Headless recorders for the cluster status-set channel.

The Console surfaces cluster status updates as toast notifications and per-task
execution state in the task controls panel. Tests that would otherwise assert on
those surfaces through Playwright can record the status-set channel directly
through the Python client.

Statuses are transient, so begin recording before the action that should raise
one. TestCase does this automatically for cases that set collect_notifications
or collect_task_status; standalone callers can use the context manager form.
"""

import logging
import threading
from collections.abc import Callable
from typing import Any, Self

from pydantic import ValidationError

import synnax as sy
from synnax.status import SET_CHANNEL

logger = logging.getLogger("synnax.recorder")

OPEN_TIMEOUT = 5.0
CLOSE_TIMEOUT = 3.0
DEFAULT_WAIT_TIMEOUT = 5.0
READ_TIMEOUT = 250 * sy.TimeSpan.MILLISECOND


class Recorder:
    """Drains the status-set channel into in-memory state on a background thread.

    Recording runs from open() until close(). Subclasses decode and store each
    raw status via _record; wait_for-style queries scan the stored state, so a
    status matches whether it arrived before or after the call. A condition
    variable guards the state and wakes waiters the moment a status is recorded,
    so queries block rather than poll.
    """

    _label = "status"

    def __init__(self, client: sy.Synnax) -> None:
        self._client = client
        self._stop = threading.Event()
        self._ready = threading.Event()
        self._cond = threading.Condition()
        self._thread: threading.Thread | None = None
        self._error: Exception | None = None

    def __enter__(self) -> Self:
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
            raise TimeoutError(f"{self._label} stream did not open in time")
        if self._error is not None:
            raise self._error

    def close(self) -> None:
        """Stop recording and tear down the subscription."""
        self._stop.set()
        if self._thread is not None:
            self._thread.join(timeout=CLOSE_TIMEOUT)

    def _record(self, raw: Any) -> bool:
        """Decode raw and store it, returning True if stored. Runs under _cond."""
        raise NotImplementedError

    def _wait_for(self, predicate: Callable[[], bool], timeout: float) -> bool:
        timer = sy.Timer()
        with self._cond:
            while True:
                if predicate():
                    return True
                remaining = timeout - timer.elapsed().seconds
                if remaining <= 0:
                    return False
                self._cond.wait(remaining)

    def _run(self) -> None:
        try:
            with self._client.open_streamer([SET_CHANNEL]) as streamer:
                self._ready.set()
                while not self._stop.is_set():
                    frame = streamer.read(timeout=READ_TIMEOUT)
                    if frame is None or SET_CHANNEL not in frame:
                        continue
                    for raw in frame[SET_CHANNEL]:
                        with self._cond:
                            if self._record(raw):
                                self._cond.notify_all()
        except Exception as e:
            if not self._ready.is_set():
                self._error = e
                self._ready.set()
            else:
                logger.error("%s recorder stopped", self._label, exc_info=e)


class Notifications(Recorder):
    """Records the status messages the Console renders as toast notifications."""

    _label = "status notification"

    def __init__(self, client: sy.Synnax) -> None:
        super().__init__(client)
        self._messages: list[str] = []

    def wait_for(self, text: str, timeout: float = DEFAULT_WAIT_TIMEOUT) -> bool:
        """Return True if a recorded status message contains text within timeout."""
        return self._wait_for(
            lambda: any(text in message for message in self._messages), timeout
        )

    def _record(self, raw: Any) -> bool:
        try:
            self._messages.append(sy.Status.model_validate(raw).message)
            return True
        except ValidationError:
            return False


class TaskStatus(Recorder):
    """Records per-task execution state, keyed by task key.

    Generic across every task type (Arc, NI, Modbus, OPC UA, LabJack): they all
    publish ``task.Status`` with ``details.task`` set to their own key.
    """

    _label = "task status"

    def __init__(self, client: sy.Synnax) -> None:
        super().__init__(client)
        self._latest: dict[sy.task.Key, sy.task.Status] = {}
        self._messages: dict[sy.task.Key, list[str]] = {}

    def wait_for(
        self, task_key: sy.task.Key, text: str, timeout: float = DEFAULT_WAIT_TIMEOUT
    ) -> bool:
        """Return True if a status for task_key contains text within timeout."""
        return self._wait_for(
            lambda: any(text in m for m in self._messages.get(task_key, [])), timeout
        )

    def latest(self, task_key: sy.task.Key) -> sy.task.Status | None:
        """Return the most recent recorded status for task_key, or None."""
        with self._cond:
            return self._latest.get(task_key)

    def is_running(self, task_key: sy.task.Key) -> bool:
        """Return True if the most recent status reports the task as running."""
        status = self.latest(task_key)
        return (
            status is not None and status.details is not None and status.details.running
        )

    def _record(self, raw: Any) -> bool:
        try:
            status = sy.task.Status.model_validate(raw)
        except ValidationError:
            # The channel carries statuses for all tasks and racks. Non-task
            # statuses (and rack statuses with a different detail schema) fail
            # validation and are skipped.
            return False
        if status.details is None:
            return False
        key = status.details.task
        self._latest[key] = status
        self._messages.setdefault(key, []).append(status.message)
        return True

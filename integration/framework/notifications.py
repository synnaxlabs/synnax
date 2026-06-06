#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Headless status-notification matching for integration tests.

The Console renders cluster status updates (task warnings, errors, and info
messages) as toast notifications. Tests that would otherwise assert on those
toasts through Playwright can use StatusNotifications, which subscribes to the
status-set channel directly through the Python client.

Statuses are transient, so begin recording before the action that should raise
the notification. TestCase does this automatically for cases that set
collect_notifications; standalone callers can use the context manager form.
"""

import logging
import threading

from pydantic import ValidationError

import synnax as sy
from synnax.status import SET_CHANNEL

logger = logging.getLogger("synnax.notifications")

OPEN_TIMEOUT = 5.0
CLOSE_TIMEOUT = 3.0
DEFAULT_WAIT_TIMEOUT = 5.0
READ_TIMEOUT = 250 * sy.TimeSpan.MILLISECOND


class StatusNotifications:
    """Records status messages published on the cluster status-set channel.

    A background thread drains the status-set channel into an in-memory list
    from open() until close(). wait_for scans that list, so a notification
    matches whether it arrived before or after the call, mirroring the Console
    toaster. A condition variable guards the list and wakes wait_for the moment
    a new message is recorded, so it blocks rather than polls.
    """

    def __init__(self, client: sy.Synnax) -> None:
        self._client = client
        self._stop = threading.Event()
        self._ready = threading.Event()
        self._cond = threading.Condition()
        self._messages: list[str] = []
        self._thread: threading.Thread | None = None
        self._error: Exception | None = None

    def __enter__(self) -> "StatusNotifications":
        self.open()
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    def open(self) -> None:
        """Begin recording in the background, blocking until the stream is live.

        Blocking until live ensures a notification raised by the next action is
        not missed.
        """
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()
        if not self._ready.wait(timeout=OPEN_TIMEOUT):
            raise TimeoutError("status notification stream did not open in time")
        if self._error is not None:
            raise self._error

    def close(self) -> None:
        """Stop recording and tear down the subscription."""
        self._stop.set()
        if self._thread is not None:
            self._thread.join(timeout=CLOSE_TIMEOUT)

    def wait_for(self, text: str, timeout: float = DEFAULT_WAIT_TIMEOUT) -> bool:
        """Return True if a recorded status message contains text within timeout.

        Blocks on the condition variable until a matching message is recorded or
        timeout elapses, waking only when the recorder appends a new message.
        """
        timer = sy.Timer()
        with self._cond:
            while True:
                if any(text in message for message in self._messages):
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
                        try:
                            message = sy.Status.model_validate(raw).message
                        except ValidationError:
                            continue
                        with self._cond:
                            self._messages.append(message)
                            self._cond.notify_all()
        except Exception as e:
            if not self._ready.is_set():
                self._error = e
                self._ready.set()
            else:
                logger.error("status notification recorder stopped", exc_info=e)

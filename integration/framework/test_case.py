#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# Suppress WebSocket keepalive ping warnings
import warnings

warnings.filterwarnings("ignore", message=".*keepalive ping.*")
warnings.filterwarnings("ignore", message=".*timed out while closing connection.*")

import traceback
from abc import ABC, abstractmethod
from collections.abc import Callable, Iterator
from contextlib import contextmanager
from pathlib import Path
from typing import Any, Literal, NoReturn, overload

import synnax as sy
from framework.log_client import LogClient, LogMode, SynnaxChannelSink
from framework.models import (
    DEFAULT_CONNECTION,
    DEFAULT_OPTIONS,
    STATUS,
    SYMBOLS,
    RunOptions,
    SynnaxConnection,
)
from framework.recorder import Notifications, TaskStatus
from framework.streamer import Streamer
from framework.telemetry import TelemetryWriter
from framework.utils import (
    create_indexed_channel,
    create_time_index,
    read_latest_value,
)
from framework.websocket import is_websocket_error, suppress_websocket_errors
from framework.writer import Writer
from x import validate_and_sanitize_name


class Skip(BaseException):
    """Raised by TestCase.skip. It is not an Exception, so a test's own `except
    Exception` cannot swallow it."""


def _numeric(value: int | float | str) -> float:
    """:raises TypeError: If value comes from a string channel."""
    if isinstance(value, str):
        raise TypeError(f"Expected a numeric channel value, got {value!r}")
    return value


class TestCase(ABC):
    """
    Parent class for all test cases in the integration test framework.

    This class handles the connection to Synnax server and provides
    three key lifecycle methods that can be overridden by subclasses:

    - setup(): Configure, add channels and subscriptions

    - run(): The main test logic (NOT async!)

    - teardown(): Cleanup, unload configs, open vents, etc.

    """

    # Configuration constants
    DEFAULT_READ_TIMEOUT: sy.CrudeTimeSpan = 1 * sy.TimeSpan.SECOND
    # Paces should_stop and every `while self.should_continue` control loop.
    STOP_POLL_RATE: sy.TimeSpan = 200 * sy.TimeSpan.MILLISECOND
    # Background heartbeat cadence; nothing asserts on it, so it runs lazily.
    TELEMETRY_RATE: sy.TimeSpan = 500 * sy.TimeSpan.MILLISECOND
    STATUS_SETTLE_DELAY: sy.TimeSpan = 200 * sy.TimeSpan.MILLISECOND
    WEBSOCKET_RETRY_DELAY: sy.TimeSpan = 500 * sy.TimeSpan.MILLISECOND
    DEFAULT_MANUAL_TIMEOUT: sy.CrudeTimeSpan | None = None

    # Opt-in: when True, the base lifecycle records cluster status notifications
    # for the duration of the test so wait_for_notification can match them.
    collect_notifications: bool = False
    # Opt-in: when True, the base lifecycle records per-task status updates for
    # the duration of the test so wait_for_task_status / task_is_running can
    # read them.
    collect_task_status: bool = False

    log_client: LogClient

    def __init__(
        self,
        *,
        synnax_connection: SynnaxConnection = DEFAULT_CONNECTION,
        name: str,
        options: RunOptions = DEFAULT_OPTIONS,
        results_dir: Path,
        **params: Any,
    ) -> None:
        """Initialize test case with Synnax server connection."""
        self.synnax_connection = synnax_connection
        self.options = options
        self.results_dir = results_dir
        self.params = params
        self.start_time: sy.TimeStamp = sy.TimeStamp.now()
        self._manual_timeout: sy.CrudeTimeSpan | None = self.DEFAULT_MANUAL_TIMEOUT
        self.read_timeout: sy.CrudeTimeSpan = self.DEFAULT_READ_TIMEOUT

        self.name = validate_and_sanitize_name(name)
        self._status: STATUS = STATUS.INITIALIZING
        self._error_message: str | None = None
        self._error_traceback: str | None = None
        self.skip_reason: str | None = None

        # Cache channel name strings
        self._ch_time = f"{self.name}_time"
        self._ch_uptime = f"{self.name}_uptime"
        self._ch_state = f"{self.name}_state"
        self._ch_log = f"{self.name}_log"

        self.client = synnax_connection.create_client()
        self.writer = Writer(self.client)

        self.log_client = LogClient(
            name=self.name,
            mode=LogMode.REALTIME if options.logs_streamed else LogMode.BUFFERED,
            persistent_sinks=[
                SynnaxChannelSink(client=self.client, channel_name=self._ch_log),
            ],
        )

        self.loop = sy.Loop(self.STOP_POLL_RATE)
        self._telemetry_writer: TelemetryWriter | None = None
        self._notifications: Notifications | None = None
        self._task_status: TaskStatus | None = None
        self.streamer = Streamer(
            client=self.client,
            read_timeout=self.read_timeout,
            log=self.log,
        )
        self._should_stop = False
        self.is_running = True

        self.time_index = create_time_index(self.client, self._ch_time)

        self.tlm: dict[str, Any] = {
            self._ch_time: sy.TimeStamp.now(),
        }

        self.add_channel(name="uptime", data_type=sy.DataType.UINT32, initial_value=0)
        self.add_channel(
            name="state", data_type=sy.DataType.UINT8, initial_value=self._status.value
        )

    def _update_tlm(
        self, tlm: dict[str, object], now: sy.TimeStamp, uptime: float
    ) -> None:
        tlm[self._ch_time] = now
        tlm[self._ch_uptime] = uptime
        tlm[self._ch_state] = self._status.value

        if self._status.value >= STATUS.FAILED.value:
            tlm[self._ch_state] = self._status.value
            self._should_stop = True

    def _handle_writer_error(self, e: Exception) -> None:
        if is_websocket_error(e):
            sy.sleep(self.WEBSOCKET_RETRY_DELAY)
        else:
            self.log(f"Writer thread error: {e}\n {traceback.format_exc()}")
            self.STATUS = STATUS.FAILED
            raise e

    def log(self, message: str) -> None:
        """Log a message. Buffered and dumped on failure unless `tc --logs` is set."""
        self.log_client.info(message)

    def _start_client_threads(self) -> None:
        self._telemetry_writer = TelemetryWriter(
            client=self.client,
            tlm=self.tlm,
            channels=list(self.tlm.keys()),
            update=self._update_tlm,
            should_stop=lambda: self._should_stop,
            rate=self.TELEMETRY_RATE,
            name=self.name,
            on_error=self._handle_writer_error,
        )
        self._telemetry_writer.start()
        self.streamer.start()

    def _stop_client(self) -> None:
        """Stop client threads and wait for completion."""
        if self.is_running:
            self._should_stop = True
            self.streamer.stop()
            self.is_running = False
            self.streamer.join()
            if self._telemetry_writer is not None:
                self._telemetry_writer.stop()
            with suppress_websocket_errors():
                self.writer.close()

        if self._status == STATUS.PENDING:
            self.STATUS = STATUS.PASSED

    def _wait_for_client_completion(
        self, timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND
    ) -> None:
        """Wait for client threads to complete."""
        timeout_secs = sy.TimeSpan.from_seconds(timeout).seconds
        self.streamer.join(timeout=timeout_secs)
        if self._telemetry_writer is not None:
            self._telemetry_writer.stop(timeout=timeout_secs)

    def _check_expectation(self) -> None:
        """Check if test met expected outcome and handle failures gracefully."""
        # Convert PENDING to PASSED if no final status set
        if self._status == STATUS.PENDING:
            self.STATUS = STATUS.PASSED

        status_symbol = SYMBOLS.get_symbol(self._status)

        if self._status == STATUS.PASSED:
            self.log(f"PASSED ({status_symbol})")
        elif self._status == STATUS.SKIPPED:
            self.log(f"SKIPPED ({status_symbol}): {self.skip_reason}")
        elif self._status == STATUS.FAILED:
            self.log(f"FAILED ({status_symbol})")
        elif self._status == STATUS.KILLED:
            self.log(f"KILLED ({status_symbol})")

        self.log(f"Uptime: {self.uptime:.1f} s")
        # Brief grace so the live state channel reflects the final status.
        sy.sleep(self.STATUS_SETTLE_DELAY)

    def _shutdown(self) -> None:
        """Gracefully shutdown test case and stop all threads."""
        self.log("Shutting down test case...")
        self.STATUS = STATUS.KILLED
        self._stop_client()
        self.log("Test case shutdown complete")

    def add_channel(
        self,
        name: str,
        data_type: sy.DataType,
        initial_value: Any = None,
        *,
        append_name: bool = True,
    ) -> None:
        """Create a telemetry channel with name {self.name}_{name}."""
        tlm_name = f"{self.name}_{name}" if append_name else name

        create_indexed_channel(
            self.client, tlm_name, data_type=data_type, index_key=self.time_index.key
        )

        self.tlm[tlm_name] = initial_value

    def subscribe(
        self,
        channels: str | list[str],
        *,
        timeout: sy.CrudeTimeSpan = 10 * sy.TimeSpan.SECOND,
    ) -> None:
        """Subscribe to channels.

        Can take either a single channel name or a list of channels.

        :param channels: Channel name(s) to subscribe to.
        :param timeout: Maximum time to wait for channels to be initialized.
        """
        self.streamer.subscribe(channels, timeout=timeout)

    def setup(self) -> None:
        """Load configs, add channels, subscribe to channels, etc."""

    def skip(self, reason: str) -> NoReturn:
        """Stop the test and mark it SKIPPED, such as when the host lacks the
        platform or hardware the test needs. Teardown still runs.

        :param reason: Why the test cannot run here. The summary shows it.
        """
        raise Skip(reason)

    @abstractmethod
    def run(self) -> None:
        """
        Main test logic.
        """
        raise NotImplementedError("Subclasses must implement the run() method")

    @contextmanager
    def _try_to(self, action: str) -> Iterator[None]:
        """Attempt action, marking the test FAILED on error without re-raising.
        A broken cleanup fails the test, but doesn't abort the remaining steps.
        """
        try:
            yield
        except Exception as e:
            self.log(f"Failed to {action}: {e}")
            failure = f"Failed to {action}: {e}"
            if self._error_message is None:
                self._error_message = failure
                self._error_traceback = traceback.format_exc()
            else:
                self._error_message += f"\n{failure}"
            self.STATUS = STATUS.FAILED

    def teardown(self) -> None:
        """Cleanup after test execution. Overrides must call super().teardown()
        so mixin cleanups (e.g. HardwareCase) run."""

    def write_tlm(self, channel: str, value: Any = None) -> None:
        """Write values to telemetry dictionary."""
        self.tlm[channel] = value

    @overload
    def read_tlm(
        self, key: str, default: Literal[None] = None
    ) -> int | float | None: ...

    @overload
    def read_tlm(self, key: str, default: int | float) -> int | float: ...

    @overload
    def read_tlm(self, key: str, default: str) -> str: ...

    def read_tlm(
        self, key: str, default: int | float | str | None = None
    ) -> int | float | str | None:
        return self.streamer.read(key, default)

    def get_value(self, channel_name: str) -> float | None:
        """Get the latest data value for any channel using the synnax client"""
        return read_latest_value(self.client, channel_name)

    def _wait_for_condition(
        self,
        channel_name: str,
        condition: Callable[[int | float | str], bool],
        condition_desc: str,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait until channel_name's value satisfies condition, or fail.

        Subscribed channels block on the streamer; unsubscribed channels fall
        back to polling the database.
        """
        timeout_span = sy.TimeSpan.from_seconds(timeout)

        if self.streamer.is_subscribed(channel_name):
            if self.streamer.wait_for(
                channel_name, predicate=condition, timeout=timeout
            ):
                return
            actual_value = self.read_tlm(channel_name)
            self.fail(
                f"Timeout waiting for {channel_name} {condition_desc}!\n"
                f"Actual: {actual_value}\n"
                f"Timeout: {timeout_span}"
            )

        timer = sy.Timer()
        while self.should_continue:
            actual_value = self.get_value(channel_name)
            if actual_value is not None and condition(actual_value):
                return
            if timer.elapsed() > timeout_span:
                break
        actual_value = self.get_value(channel_name)
        if actual_value is not None and condition(actual_value):
            return
        self.fail(
            f"Timeout waiting for {channel_name} {condition_desc}!\n"
            f"Actual: {actual_value}\n"
            f"Timeout: {timeout_span}"
        )

    def wait_for_eq(
        self,
        channel_name: str,
        expected: int | float | str,
        *,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait for channel value == expected."""
        self._wait_for_condition(
            channel_name, lambda v: v == expected, f"== {expected}", timeout
        )

    def wait_for_gt(
        self,
        channel_name: str,
        threshold: float | int,
        *,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait for channel value > threshold."""
        self._wait_for_condition(
            channel_name, lambda v: _numeric(v) > threshold, f"> {threshold}", timeout
        )

    def wait_for_ge(
        self,
        channel_name: str,
        threshold: float | int,
        *,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait for channel value >= threshold."""
        self._wait_for_condition(
            channel_name,
            lambda v: _numeric(v) >= threshold,
            f">= {threshold}",
            timeout,
        )

    def wait_for_lt(
        self,
        channel_name: str,
        threshold: float | int,
        *,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait for channel value < threshold."""
        self._wait_for_condition(
            channel_name, lambda v: _numeric(v) < threshold, f"< {threshold}", timeout
        )

    def wait_for_le(
        self,
        channel_name: str,
        threshold: float | int,
        *,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait for channel value <= threshold."""
        self._wait_for_condition(
            channel_name,
            lambda v: _numeric(v) <= threshold,
            f"<= {threshold}",
            timeout,
        )

    def wait_for_near(
        self,
        channel_name: str,
        target: float,
        *,
        tolerance: float,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait for channel value to be within tolerance of target."""
        self._wait_for_condition(
            channel_name,
            lambda v: abs(_numeric(v) - target) < tolerance,
            f"≈ {target} (±{tolerance})",
            timeout,
        )

    def wait_for_notification(self, text: str, *, timeout: float = 5.0) -> bool:
        """Return True if a status notification containing text has surfaced.

        Requires collect_notifications = True so the base lifecycle records
        status updates for the duration of the test.
        """
        if self._notifications is None:
            raise RuntimeError(
                "wait_for_notification requires collect_notifications = True"
            )
        return self._notifications.wait_for(text, timeout=timeout)

    def wait_for_task_status(
        self, task_key: sy.task.Key, text: str, *, timeout: float = 5.0
    ) -> bool:
        """Return True if a status for task_key containing text has surfaced.

        Requires collect_task_status = True so the base lifecycle records task
        status updates for the duration of the test.
        """
        if self._task_status is None:
            raise RuntimeError(
                "wait_for_task_status requires collect_task_status = True"
            )
        return self._task_status.wait_for(task_key, text, timeout=timeout)

    def task_is_running(self, task_key: sy.task.Key) -> bool:
        """Return True if the latest status reports task_key as running.

        Requires collect_task_status = True so the base lifecycle records task
        status updates for the duration of the test.
        """
        if self._task_status is None:
            raise RuntimeError("task_is_running requires collect_task_status = True")
        return self._task_status.is_running(task_key)

    @property
    def name(self) -> str:
        """Get the name of the test case."""
        return self._name

    @name.setter
    def name(self, value: str) -> None:
        """Set the name of the test case."""
        self._name = value

    @property
    def STATUS(self) -> STATUS:
        """Get the current test status."""
        return self._status

    @STATUS.setter
    def STATUS(self, value: STATUS) -> None:
        """Set the test status and update telemetry if client is running."""
        # Only allow status changes to higher-priority statuses
        if value.value >= self._status.value:
            self._status = value
            # Update telemetry if client thread is running
            if hasattr(self, "tlm") and self.is_client_running():
                try:
                    self.tlm[self._ch_state] = value.value
                except Exception as e:
                    raise RuntimeError("Failed to set status") from e
        else:
            self.log(f"Invalid status change: {self._status} -> {value}")

    @property
    def uptime(self) -> float:
        """Get the uptime of the test case."""
        return float(self.tlm.get(self._ch_uptime, -1))

    @property
    def time(self) -> float:
        """Get the uptime of the test case."""
        return float(self.tlm.get(self._ch_time, -1))

    @property
    def state(self) -> float:
        """Get the state of the test case."""
        return float(self.tlm.get(self._ch_state, -1))

    @property
    def manual_timeout(self) -> sy.CrudeTimeSpan | None:
        """Get the manual timeout of the test case."""
        return self._manual_timeout

    @property
    def should_stop(self) -> bool:
        self.loop.wait()  # Rate limit checks to avoid busy loops
        timed_out = self._manual_timeout is not None and sy.TimeStamp.since(
            self.start_time
        ) > sy.TimeSpan.from_seconds(self._manual_timeout)
        return timed_out or self._should_stop

    @property
    def should_continue(self) -> bool:
        return not self.should_stop

    def set_manual_timeout(self, value: sy.CrudeTimeSpan) -> None:
        """Set the manual timeout of the test case."""
        self._manual_timeout = value
        self.log(f"Manual timeout set ({sy.TimeSpan.from_seconds(value)})")

    def configure(
        self,
        *,
        read_timeout: sy.CrudeTimeSpan | None = None,
        loop_rate: sy.CrudeTimeSpan | None = None,
        manual_timeout: sy.CrudeTimeSpan | None = None,
    ) -> None:
        """Configure test case parameters.

        :param read_timeout: Timeout for synnax client read operations.
        :param loop_rate: Synnax Client Loop interval.
        :param manual_timeout: Manual timeout for test termination.
        """
        log_parts: list[str] = []
        if read_timeout is not None:
            self.streamer.read_timeout = read_timeout
            log_parts.append(f"read_timeout={read_timeout}")
        if loop_rate is not None:
            self.loop = sy.Loop(sy.TimeSpan.from_seconds(loop_rate))
            log_parts.append(f"loop_rate={loop_rate}")
        if manual_timeout is not None:
            self._manual_timeout = manual_timeout
            log_parts.append(f"manual_timeout={manual_timeout}")
        self.log(f"Configured with: {', '.join(log_parts)}")

    def is_client_running(self) -> bool:
        """Check if client threads are running."""
        return self._telemetry_writer is not None and self._telemetry_writer.is_alive()

    def fail(self, message: str | None = None) -> NoReturn:
        if message is not None:
            self.log(f"FAILED: {message}")
        self._error_message = message
        self.STATUS = STATUS.FAILED
        raise RuntimeError(message or "Test failed")

    def execute(self) -> None:
        """Execute complete test lifecycle: setup -> run -> teardown."""
        phase_started = sy.TimeStamp.now()

        def log_phase(phase: str) -> None:
            nonlocal phase_started
            now = sy.TimeStamp.now()
            self.log(f"{phase} took {sy.TimeSpan(now - phase_started)}")
            phase_started = now

        try:
            # Set STATUS at the top level as opposed to within
            # the override methods. Ensures that the status is set
            # Even if the child classes don't call super()

            self.STATUS = STATUS.INITIALIZING
            self.setup()
            log_phase("setup")

            if self.collect_notifications:
                self._notifications = Notifications(self.client)
                self._notifications.open()

            if self.collect_task_status:
                self._task_status = TaskStatus(self.client)
                self._task_status.open()

            self._start_client_threads()

            self.STATUS = STATUS.RUNNING
            self.run()
            log_phase("run")

            # Set to PENDING only if not in final state
            if self._status not in [STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED]:
                self.STATUS = STATUS.PENDING

            # PASSED set in _check_expectation()

        except Skip as e:
            self.skip_reason = str(e)
            self.STATUS = STATUS.SKIPPED
        except Exception as e:
            tb = traceback.format_exc()
            self.STATUS = STATUS.FAILED
            self.log(f"EXCEPTION: {e}\n{tb}")
            if self._error_message is None:
                self._error_message = str(e)
            self._error_traceback = tb
        finally:
            # A conductor timeout can interrupt teardown, so the client threads stop
            # in a finally of their own.
            try:
                phase_started = sy.TimeStamp.now()
                try:
                    self.teardown()
                except Exception as teardown_error:
                    self.log(f"Teardown error: {teardown_error}")
                log_phase("teardown")
                if self._notifications is not None:
                    self._notifications.close()
                if self._task_status is not None:
                    self._task_status.close()
                self.streamer.stop()
                self.streamer.join()
                with self._try_to("stream telemetry"):
                    self.streamer.check()
                    if self._notifications is not None:
                        self._notifications.check()
                    if self._task_status is not None:
                        self._task_status.check()
                self._check_expectation()
            finally:
                self._stop_client()
                self._wait_for_client_completion()

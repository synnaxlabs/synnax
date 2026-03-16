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

import sys
import threading
import traceback
from abc import ABC, abstractmethod
from collections import deque
from collections.abc import Callable
from dataclasses import dataclass
from enum import Enum, auto
from pathlib import Path
from typing import Any, Literal, overload

import synnax as sy
from synnax.telem import SampleValue

from framework.log_client import LogClient, LogMode, SynnaxChannelSink
from framework.utils import (
    WebSocketErrorFilter,
    ignore_websocket_errors,
    is_websocket_error,
    suppress_websocket_errors,
    validate_and_sanitize_name,
)

# Error filter
sys.excepthook = ignore_websocket_errors
sys.stderr = WebSocketErrorFilter()


@dataclass
class SynnaxConnection:
    """Data class representing the Synnax connection parameters."""

    server_address: str = "localhost"
    port: int = 9090
    username: str = "synnax"
    password: str = "seldon"
    secure: bool = False

    def create_client(self) -> sy.Synnax:
        return sy.Synnax(
            host=self.server_address,
            port=self.port,
            username=self.username,
            password=self.password,
            secure=self.secure,
        )


class STATUS(Enum):
    """Enum representing the status of a test."""

    INITIALIZING = auto()
    RUNNING = auto()
    PENDING = auto()
    PASSED = auto()
    FAILED = auto()
    TIMEOUT = auto()
    KILLED = auto()


class SYMBOLS(Enum):
    PASSED = "✅"
    FAILED = "❌"
    KILLED = "💀"
    TIMEOUT = "⏰"

    @classmethod
    def get_symbol(cls, status: STATUS) -> str:
        """Get symbol for a given status, with fallback to '?' if not found."""
        try:
            return cls[status.name].value
        except (KeyError, AttributeError):
            return "❓"


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
    DEFAULT_LOOP_RATE: sy.CrudeTimeSpan = 200 * sy.TimeSpan.MILLISECOND
    WEBSOCKET_RETRY_DELAY: sy.CrudeTimeSpan = 500 * sy.TimeSpan.MILLISECOND
    MAX_CLEANUP_RETRIES: int = 3
    DEFAULT_TIMEOUT_LIMIT: sy.CrudeTimeSpan | None = None
    DEFAULT_MANUAL_TIMEOUT: sy.CrudeTimeSpan | None = None

    log_client: LogClient
    frame_in: sy.Frame | None

    def __init__(
        self,
        synnax_connection: SynnaxConnection = SynnaxConnection(),
        *,
        name: str,
        **params: Any,
    ) -> None:
        self.synnax_connection = synnax_connection

        """Initialize test case with Synnax server connection."""
        self.params = params
        self.start_time: sy.TimeStamp = sy.TimeStamp.now()
        self._timeout_limit: sy.CrudeTimeSpan | None = self.DEFAULT_TIMEOUT_LIMIT
        self._manual_timeout: sy.CrudeTimeSpan | None = self.DEFAULT_MANUAL_TIMEOUT
        self.read_frame: dict[str, int | float | str] | None = None
        self.read_timeout: sy.CrudeTimeSpan = self.DEFAULT_READ_TIMEOUT

        self.name = validate_and_sanitize_name(name)
        self._status: STATUS = STATUS.INITIALIZING

        # Cache channel name strings
        self._ch_time = f"{self.name}_time"
        self._ch_uptime = f"{self.name}_uptime"
        self._ch_state = f"{self.name}_state"
        self._ch_log = f"{self.name}_log"

        self.client = synnax_connection.create_client()

        self.log_client = LogClient(
            name=self.name,
            mode=LogMode.BUFFERED,
            persistent_sinks=[
                SynnaxChannelSink(self.client, self._ch_log),
            ],
        )

        self.loop = sy.Loop(self.DEFAULT_LOOP_RATE)
        self.client_thread = None
        self.writer_thread: threading.Thread = threading.Thread()
        self.streamer_thread: threading.Thread = threading.Thread()
        self._auto_pass: bool = False
        self._should_stop = False
        self.is_running = True

        self.subscribed_channels: set[str] = set()

        self.time_index = self.client.channels.create(
            name=self._ch_time,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        self.tlm = {
            self._ch_time: sy.TimeStamp.now(),
        }

        self.add_channel(name="uptime", data_type=sy.DataType.UINT32, initial_value=0)
        self.add_channel(
            name="state", data_type=sy.DataType.UINT8, initial_value=self._status.value
        )

    def _writer_loop(self) -> None:
        """Writer thread that writes telemetry at consistent interval."""
        start_time = self.start_time
        client = None

        try:
            client = self.client.open_writer(
                start=start_time,
                channels=list(self.tlm.keys()),
                name=self.name,
            )

            while not self.should_stop:
                now = sy.TimeStamp.now()
                elapsed = now - self.start_time
                uptime_seconds = elapsed / sy.TimeSpan.SECOND
                self.tlm[self._ch_time] = now
                self.tlm[self._ch_uptime] = uptime_seconds
                self.tlm[self._ch_state] = self._status.value

                if (
                    self._timeout_limit is not None
                    and elapsed > sy.TimeSpan.from_seconds(self._timeout_limit)
                ):
                    self.log(f"Timeout at {uptime_seconds:.1f}s")
                    self.STATUS = STATUS.TIMEOUT

                if self._status.value >= STATUS.FAILED.value:
                    self.tlm[self._ch_state] = self._status.value
                    self._should_stop = True

                try:
                    client.write(self.tlm)
                except Exception as e:
                    if is_websocket_error(e):
                        sy.sleep(self.WEBSOCKET_RETRY_DELAY)
                    else:
                        self.STATUS = STATUS.FAILED
                        raise

                # Final write attempt for redundancy
                with suppress_websocket_errors():
                    client.write(self.tlm)

        except Exception as e:
            if not is_websocket_error(e):
                self.log(f"Writer thread error: {e}\n {traceback.format_exc()}")
                self.STATUS = STATUS.FAILED
                raise

        finally:
            if client is not None:
                with suppress_websocket_errors():
                    client.close()

    def _streamer_loop(self) -> None:
        """Streamer thread that reads data on demand with timeout."""
        self.read_frame = {}
        streamer = None
        if len(self.subscribed_channels):
            for channel in self.subscribed_channels:
                self.read_frame[channel] = 0
        else:
            return

        try:
            streamer = self.client.open_streamer(list(self.subscribed_channels))

            while not self._should_stop:
                try:
                    self.frame_in = streamer.read(self.read_timeout)
                    if self.frame_in is not None:
                        for key, value in self.frame_in.items():
                            self.read_frame[key] = value[-1]

                except Exception as e:
                    if is_websocket_error(e):
                        sy.sleep(self.WEBSOCKET_RETRY_DELAY)
                    else:
                        self.log(f"Streamer error: {e}")
                        break

        except Exception as e:
            if not is_websocket_error(e):
                self.log(f"Streamer thread error: {e}\n {traceback.format_exc()}")
                self.STATUS = STATUS.FAILED
                raise

        finally:
            if streamer is not None:
                with suppress_websocket_errors():
                    streamer.close()

    def log(self, message: str) -> None:
        """Log a message. Buffered by default; dumped by conductor on failure."""
        self.log_client.info(message)

    def _start_client_threads(self) -> None:
        # Start writer thread (writes telemetry at consistent interval)
        self.writer_thread = threading.Thread(target=self._writer_loop, daemon=True)
        self.writer_thread.start()

        # Start streamer thread (reads data on demand)
        self.streamer_thread = threading.Thread(target=self._streamer_loop, daemon=True)
        self.streamer_thread.start()

    def _stop_client(self) -> None:
        """Stop client threads and wait for completion."""
        if self.is_running:
            self._should_stop = True
            self.is_running = False

            # Stop streamer thread
            if self.streamer_thread and self.streamer_thread.is_alive():
                self.streamer_thread.join(timeout=5)
                if self.streamer_thread.is_alive():
                    self.log("Warning: streamer thread did not stop within timeout")

            # Stop writer thread
            if self.writer_thread.is_alive():
                self.writer_thread.join(timeout=5)
                if self.writer_thread.is_alive():
                    self.log("Warning: writer thread did not stop within timeout")

        # All done? All done.
        if self._status == STATUS.PENDING:
            self.STATUS = STATUS.PASSED

    def _wait_for_client_completion(
        self, timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND
    ) -> None:
        """Wait for client threads to complete."""
        timeout_secs = sy.TimeSpan.to_seconds(timeout)
        # Wait for streamer thread
        if self.streamer_thread.is_alive():
            self.streamer_thread.join(timeout=timeout_secs)
            if self.streamer_thread.is_alive():
                self.log(
                    "Warning: streamer thread still alive after wait_for_client_completion"
                )

        # Wait for writer thread
        if self.writer_thread.is_alive():
            self.writer_thread.join(timeout=timeout_secs)
            if self.writer_thread.is_alive():
                self.log(
                    "Warning: writer thread still alive after wait_for_client_completion"
                )

    def _check_expectation(self) -> None:
        """Check if test met expected outcome and handle failures gracefully."""
        # Convert PENDING to PASSED if no final status set
        if self._status == STATUS.PENDING:
            self.STATUS = STATUS.PASSED

        status_symbol = SYMBOLS.get_symbol(self._status)

        if self._status == STATUS.PASSED:
            self.log(f"PASSED ({status_symbol})")
        elif self._status == STATUS.FAILED:
            self.log(f"FAILED ({status_symbol})")
        elif self._status == STATUS.TIMEOUT:
            limit_str = (
                sy.TimeSpan.from_seconds(self._timeout_limit)
                if self._timeout_limit is not None
                else "N/A"
            )
            self.log(f"TIMEOUT ({status_symbol}): {limit_str}")
        elif self._status == STATUS.KILLED:
            self.log(f"KILLED ({status_symbol})")

        self.log(f"Uptime: {self.uptime:.1f} s")
        # Sleep for 2 loops to ensure the status is updated
        sy.sleep(self.DEFAULT_LOOP_RATE * 2)

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
        append_name: bool = True,
    ) -> None:
        """Create a telemetry channel with name {self.name}_{name}."""
        if append_name:
            tlm_name = f"{self.name}_{name}"
        else:
            tlm_name = name

        self.client.channels.create(
            name=tlm_name,
            data_type=data_type,
            index=self.time_index.key,
            retrieve_if_name_exists=True,
        )

        self.tlm[tlm_name] = initial_value

    def subscribe(
        self,
        channels: str | list[str],
        timeout: sy.CrudeTimeSpan = 10 * sy.TimeSpan.SECOND,
    ) -> None:
        """Subscribe to channels.

        Can take either a single channel name or a list of channels.

        :param channels: Channel name(s) to subscribe to.
        :param timeout: Maximum time to wait for channels to be initialized.
        """
        timeout_span = sy.TimeSpan.from_seconds(timeout)
        self.log(f"Subscribing to channels: {channels} ({timeout_span} timeout)")

        client = self.client
        time_start = sy.TimeStamp.now()
        found = False

        while self.loop.wait():
            time_now = sy.TimeStamp.now()
            elapsed = time_now - time_start
            if elapsed > timeout_span:
                break

            try:
                existing_channels = client.channels.retrieve(channels)

                if isinstance(channels, str):
                    found = existing_channels is not None
                else:
                    found = isinstance(existing_channels, list) and len(
                        existing_channels
                    ) == len(channels)

                if found:
                    break

            except Exception as e:
                self.log(f"Channel retrieval failed: {e}")
                continue

        if not found:
            raise TimeoutError(f"Unable to retrieve channels: {channels}")

        self.log(f"Channels retrieved")
        if isinstance(channels, str):
            self.subscribed_channels.add(channels)
        elif isinstance(channels, list):
            self.subscribed_channels.update(channels)
        return None

    def setup(self) -> None:
        """Load configs, add channels, subscribe to channels, etc."""
        return None

    def auto_pass(self, msg: str) -> None:
        """Set the auto pass flag. Include reason for passing."""
        self.log(f"AUTO PASS Enabled: {msg}")
        self._auto_pass = True

    @abstractmethod
    def run(self) -> None:
        """
        Main test logic.
        """
        raise NotImplementedError("Subclasses must implement the run() method")

    def teardown(self) -> None:
        """Cleanup after test execution. Override for custom cleanup logic."""
        pass

    def write_tlm(self, channel: str, value: Any = None) -> None:
        """Write values to telemetry dictionary. Can take single key-value or dict of multiple channels."""
        # if isinstance(channel, self.tlm.keys()):
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
        try:
            if self.read_frame is not None:
                return self.read_frame.get(key, default)
            return default
        except:
            return default

    def get_value(self, channel_name: str) -> float | None:
        """Get the latest data value for any channel using the synnax client"""
        try:
            # Retry with short delays for CI resource constraints
            for attempt in range(3):
                latest_value = self.client.read_latest(channel_name)
                if latest_value is not None and len(latest_value) > 0:
                    return float(latest_value[-1])

                # If read_latest is empty, read recent time range
                now = sy.TimeStamp.now()
                recent_range = sy.TimeRange(now - sy.TimeSpan.SECOND * 3, now)
                frame = self.client.read(recent_range, channel_name)
                if len(frame) > 0:
                    return float(frame[-1])
                if attempt < 2:
                    sy.sleep(0.2)

            return None

        except:
            raise RuntimeError(f'Could not get value for channel "{channel_name}"')

    def _wait_for_condition(
        self,
        channel_name: str,
        condition: Callable[[SampleValue], bool],
        condition_desc: str,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
        is_virtual: bool = False,
    ) -> None:
        """Base method for waiting on a channel value condition.

        The condition is always checked at least once before the timeout is evaluated,
        so timeout=0 can be used for immediate assertions without waiting.

        :param channel_name: Name of the channel to read.
        :param condition: Function that takes a value and returns True when met.
        :param condition_desc: Human-readable description (e.g., "== 1", "> 27").
        :param timeout: Maximum time to wait. Use 0 for immediate check.
        :param is_virtual: If True, read from subscribed telemetry (read_tlm).
            If False, read from database (get_value).
        """
        timer = sy.Timer()
        timeout_span = sy.TimeSpan.from_seconds(timeout)

        while self.should_continue:
            actual_value = (
                self.read_tlm(channel_name)
                if is_virtual
                else self.get_value(channel_name)
            )
            if actual_value is not None and condition(actual_value):
                return
            # This last to ensure the check runs at least once.
            if timer.elapsed() > timeout_span:
                break

        actual_value = (
            self.read_tlm(channel_name) if is_virtual else self.get_value(channel_name)
        )
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
        expected: SampleValue,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
        is_virtual: bool = False,
    ) -> None:
        """Wait for channel value == expected."""
        self._wait_for_condition(
            channel_name, lambda v: v == expected, f"== {expected}", timeout, is_virtual
        )

    def wait_for_gt(
        self,
        channel_name: str,
        threshold: float | int,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
        is_virtual: bool = False,
    ) -> None:
        """Wait for channel value > threshold."""
        self._wait_for_condition(
            channel_name, lambda v: v > threshold, f"> {threshold}", timeout, is_virtual
        )

    def wait_for_ge(
        self,
        channel_name: str,
        threshold: float | int,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
        is_virtual: bool = False,
    ) -> None:
        """Wait for channel value >= threshold."""
        self._wait_for_condition(
            channel_name,
            lambda v: v >= threshold,
            f">= {threshold}",
            timeout,
            is_virtual,
        )

    def wait_for_lt(
        self,
        channel_name: str,
        threshold: float | int,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
        is_virtual: bool = False,
    ) -> None:
        """Wait for channel value < threshold."""
        self._wait_for_condition(
            channel_name, lambda v: v < threshold, f"< {threshold}", timeout, is_virtual
        )

    def wait_for_le(
        self,
        channel_name: str,
        threshold: float | int,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
        is_virtual: bool = False,
    ) -> None:
        """Wait for channel value <= threshold."""
        self._wait_for_condition(
            channel_name,
            lambda v: v <= threshold,
            f"<= {threshold}",
            timeout,
            is_virtual,
        )

    def wait_for_near(
        self,
        channel_name: str,
        target: float,
        *,
        tolerance: float,
        timeout: sy.CrudeTimeSpan = 5 * sy.TimeSpan.SECOND,
        is_virtual: bool = False,
    ) -> None:
        """Wait for channel value to be within tolerance of target."""
        self._wait_for_condition(
            channel_name,
            lambda v: abs(v - target) < tolerance,
            f"≈ {target} (±{tolerance})",
            timeout,
            is_virtual,
        )

    @overload
    def get_state(
        self, key: str, default: Literal[None] = None
    ) -> int | float | None: ...

    @overload
    def get_state(self, key: str, default: int | float) -> int | float: ...

    def get_state(
        self, key: str, default: int | float | None = None
    ) -> int | float | None:
        """
        Easily get state of this object.

        - self.name + "state"
        - self.name + "time"
        - self.name + "uptime"
        """

        name_ch = self.name + "_" + key
        value = self.tlm.get(name_ch, default)
        return value

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
                    raise RuntimeError(f"Failed to set status: {e}")
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

    def wait_for_tlm_stale(self, buffer_size: int = 5) -> bool:
        """
        Wait for all subscribed channels to be Stale (inactive).
        Requires the last buffer_size frames to be identical.
        """
        self.log("Waiting for all channels to be stale (inactive)")

        # Buffer to store the last n vals arrays
        vals_buffer: deque[Any] = deque(maxlen=buffer_size)

        while self.should_continue:
            vals_now = []
            for ch in self.subscribed_channels:
                vals_now.append(self.read_tlm(ch))

            # Add current values to buffer
            vals_buffer.append(vals_now)

            # Check if buffer is full and all entries are identical
            if len(vals_buffer) == buffer_size:
                first_vals = vals_buffer[0]
                if all(vals == first_vals for vals in vals_buffer):
                    self.log(
                        f"All channels are stale (last {buffer_size} frames identical)"
                    )
                    return True

        raise TimeoutError("Some Channels remain active")

    def set_manual_timeout(self, value: sy.CrudeTimeSpan) -> None:
        """Set the manual timeout of the test case."""
        self._manual_timeout = value
        self.log(f"Manual timeout set ({sy.TimeSpan.from_seconds(value)})")

    def configure(
        self,
        read_timeout: sy.CrudeTimeSpan | None = None,
        loop_rate: sy.CrudeTimeSpan | None = None,
        timeout_limit: sy.CrudeTimeSpan | None = None,
        manual_timeout: sy.CrudeTimeSpan | None = None,
    ) -> None:
        """Configure test case parameters.

        :param read_timeout: Timeout for synnax client read operations.
        :param loop_rate: Synnax Client Loop interval.
        :param timeout_limit: Maximum execution time before failure.
        :param manual_timeout: Manual timeout for test termination.
        """
        params = {}
        if read_timeout is not None:
            self.read_timeout = read_timeout
            params["read_timeout"] = read_timeout

        if loop_rate is not None:
            self.loop = sy.Loop(loop_rate)
            params["loop_rate"] = self.loop

        if timeout_limit is not None:
            self._timeout_limit = timeout_limit
            params["timeout_limit"] = timeout_limit

        if manual_timeout is not None:
            self._manual_timeout = manual_timeout
            params["manual_timeout"] = manual_timeout

        log_params = {
            k: (f"{v.interval}" if isinstance(v, sy.Loop) else v)
            for k, v in params.items()
        }
        self.log(f"Configured with: {log_params}")

    def is_client_running(self) -> bool:
        """Check if client threads are running."""
        streamer_running = self.streamer_thread.is_alive()
        writer_running = self.writer_thread.is_alive()
        return bool(streamer_running or writer_running)

    def get_client_status(self) -> str:
        """Get client thread status."""
        streamer_status = "Not started"
        writer_status = "Not started"

        if self.streamer_thread:
            streamer_status = (
                "Running" if self.streamer_thread.is_alive() else "Stopped"
            )

        if self.writer_thread:
            writer_status = "Running" if self.writer_thread.is_alive() else "Stopped"

        return f"Streamer: {streamer_status}, Writer: {writer_status}"

    def get_streamer_status(self) -> str:
        """Get streamer thread status."""
        return "Running" if self.streamer_thread.is_alive() else "Stopped"

    def get_writer_status(self) -> str:
        """Get writer thread status."""
        return "Running" if self.writer_thread.is_alive() else "Stopped"

    def fail(self, message: str | None = None) -> None:
        if message is not None:
            self.log(f"FAILED: {message}")
        self.STATUS = STATUS.FAILED
        raise RuntimeError(message or "Test failed")

    def execute(self) -> None:
        """Execute complete test lifecycle: setup -> run -> teardown."""
        try:

            # Set STATUS at the top level as opposed to within
            # the override methods. Ensures that the status is set
            # Even if the child classes don't call super()

            self.STATUS = STATUS.INITIALIZING
            self.setup()

            self._start_client_threads()

            self.STATUS = STATUS.RUNNING
            if not self._auto_pass:
                self.run()

            # Set to PENDING only if not in final state
            if self._status not in [STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED]:
                self.STATUS = STATUS.PENDING

            # PASSED set in _check_expectation()

        except Exception as e:
            if not is_websocket_error(e):
                self.STATUS = STATUS.FAILED
                self.log(f"EXCEPTION: {e}\n{traceback.format_exc()}")
        finally:
            try:
                self.teardown()
            except Exception as teardown_error:
                self.log(f"Teardown error: {teardown_error}")
            self._check_expectation()
            self._stop_client()
            self._wait_for_client_completion()
            self.log_client.close()

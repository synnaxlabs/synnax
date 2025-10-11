#  Copyright 2025 Synnax Labs, Inc.
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

import logging
import sys
import threading
import traceback
from abc import ABC, abstractmethod
from collections import deque
from dataclasses import dataclass, field
from enum import Enum, auto
from typing import Any, Callable, Dict, List, Literal, Optional, Set, Union, overload

import synnax as sy

from framework.utils import (
    WebSocketErrorFilter,
    ignore_websocket_errors,
    is_ci,
    is_websocket_error,
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
    PASSED = "✅"  # Green check mark
    FAILED = "❌"  # Red X
    KILLED = "💀"  # Skull
    TIMEOUT = "⏰"  # Alarm clock

    @classmethod
    def get_symbol(cls, status: STATUS) -> str:
        """Get symbol for a given status, with fallback to '?' if not found."""
        try:
            return cls[status.name].value
        except (KeyError, AttributeError):
            return "❓"  # Question mark emoji


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
    DEFAULT_READ_TIMEOUT: int = 1
    DEFAULT_LOOP_RATE: float = 0.2  # 5 Hz
    WEBSOCKET_RETRY_DELAY: float = 0.5  # s
    MAX_CLEANUP_RETRIES: int = 3
    CLIENT_THREAD_START_DELAY: int = 1
    DEFAULT_TIMEOUT_LIMIT: int = -1
    DEFAULT_MANUAL_TIMEOUT: int = -1

    def __init__(
        self,
        synnax_connection: SynnaxConnection,
        name: str,
        expect: str = "PASSED",
        **params: Any,
    ) -> None:
        self.synnax_connection = synnax_connection

        if expect in ["FAILED", "TIMEOUT", "KILLED"]:
            # Use this wisely!
            if expect == "FAILED":
                self.expected_outcome = STATUS.FAILED
            elif expect == "TIMEOUT":
                self.expected_outcome = STATUS.TIMEOUT
            elif expect == "KILLED":
                self.expected_outcome = STATUS.KILLED
        else:
            self.expected_outcome = STATUS.PASSED

        """Initialize test case with Synnax server connection."""
        self.params = params
        self.start_time: sy.TimeStamp = sy.TimeStamp.now()
        self._timeout_limit: int = self.DEFAULT_TIMEOUT_LIMIT  # -1 = no timeout
        self._manual_timeout: int = self.DEFAULT_MANUAL_TIMEOUT
        self.read_frame: Optional[Dict[str, Union[int, float]]] = None
        self.read_timeout = self.DEFAULT_READ_TIMEOUT

        self.name = validate_and_sanitize_name(name)

        self._setup_logging()
        self._status: STATUS = STATUS.INITIALIZING

        self.client = sy.Synnax(
            host=synnax_connection.server_address,
            port=synnax_connection.port,
            username=synnax_connection.username,
            password=synnax_connection.password,
            secure=synnax_connection.secure,
        )

        self.loop = sy.Loop(self.DEFAULT_LOOP_RATE)
        self.client_thread = None
        self.writer_thread: threading.Thread = threading.Thread()
        self.streamer_thread: threading.Thread = threading.Thread()
        self._should_stop = False
        self.is_running = True

        self.subscribed_channels: Set[str] = set()

        self.time_index = self.client.channels.create(
            name=f"{self.name}_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        self.tlm = {
            f"{self.name}_time": sy.TimeStamp.now(),
        }

        self.add_channel(name="uptime", data_type=sy.DataType.UINT32, initial_value=0)
        self.add_channel(
            name="state", data_type=sy.DataType.UINT8, initial_value=self._status.value
        )

    def _setup_logging(self) -> None:
        """Setup logging for real-time output (same approach as TestConductor)."""
        # Check if running in CI environment
        ci_environment = is_ci()

        # Force unbuffered output in CI environments
        if ci_environment:
            if hasattr(sys.stdout, "reconfigure"):
                sys.stdout.reconfigure(line_buffering=True)

        # Create logger for this test case (don't configure root logger)
        self.logger = logging.getLogger(self.name)
        self.logger.setLevel(logging.INFO)

        # Remove any existing handlers to avoid duplicates
        for handler in self.logger.handlers[:]:
            self.logger.removeHandler(handler)

        # Add single handler
        handler = logging.StreamHandler(sys.stdout)
        handler.setLevel(logging.INFO)
        formatter = logging.Formatter("%(message)s")
        handler.setFormatter(formatter)
        self.logger.addHandler(handler)

        # Prevent propagation to root logger to avoid duplicate output
        self.logger.propagate = False

        # Force immediate flush for real-time output in CI
        for handler in self.logger.handlers:
            if hasattr(handler, "stream") and hasattr(handler.stream, "flush"):

                def make_flush(h: Any) -> Callable[[], None]:
                    return lambda: h.stream.flush()

                setattr(handler, "flush", make_flush(handler))

    def _writer_loop(self) -> None:
        """Writer thread that writes telemetry at consistent interval."""
        start_time = self.start_time
        client = None

        try:
            client = self.client.open_writer(
                start=start_time,
                channels=list(self.tlm.keys()),
                name=self.name,
                enable_auto_commit=True,
            )

            while self.loop.wait() and not self.should_stop:
                """
                # Update telemetry
                """

                now = sy.TimeStamp.now()
                uptime_value = (now - self.start_time) / 1e9
                self.tlm[f"{self.name}_time"] = now
                self.tlm[f"{self.name}_uptime"] = uptime_value
                self.tlm[f"{self.name}_state"] = self._status.value

                # Check for timeout
                if self._timeout_limit > 0 and uptime_value > self._timeout_limit:
                    self._log_message(f"Timeout at {uptime_value}s")
                    self.STATUS = STATUS.TIMEOUT

                # Check for completion due to failure
                if self._status.value >= STATUS.FAILED.value:
                    self.tlm[f"{self.name}_state"] = self._status.value
                    self._should_stop = True

                try:
                    client.write(self.tlm)
                except Exception as e:
                    if is_websocket_error(e):
                        sy.sleep(self.WEBSOCKET_RETRY_DELAY)
                    else:
                        self.STATUS = STATUS.FAILED
                        raise e

                # Final write attempt for redundancy
                try:
                    client.write(self.tlm)
                except:
                    pass
            self._log_message("Writer thread shutting down")

        except Exception as e:
            if is_websocket_error(e):
                pass
            else:
                self._log_message(
                    f"Writer thread error: {e}\n {traceback.format_exc()}"
                )
                self.STATUS = STATUS.FAILED
                raise e

        finally:
            # Cleanup writer
            try:
                if "client" in locals() and client:
                    client.close()
            except Exception as cleanup_error:
                if is_websocket_error(cleanup_error):
                    pass
                else:
                    self._log_message(f"Writer cleanup error: {cleanup_error}")

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
                    # Read data on demand with timeout (not tied to loop.wait())
                    self.frame_in = streamer.read(self.read_timeout)
                    if self.frame_in is not None:
                        for key, value in self.frame_in.items():
                            self.read_frame[key] = value[-1]

                except Exception as e:
                    if is_websocket_error(e):
                        sy.sleep(self.WEBSOCKET_RETRY_DELAY)
                    else:
                        self._log_message(f"Streamer error: {e}")
                        break

            self._log_message("Streamer thread shutting down")

        except Exception as e:
            if is_websocket_error(e):
                pass
            else:
                self._log_message(
                    f"Streamer thread error: {e}\n {traceback.format_exc()}"
                )
                self.STATUS = STATUS.FAILED
                raise e

        finally:
            # Cleanup streamer
            try:
                if "streamer" in locals() and streamer:
                    streamer.close()
            except Exception as cleanup_error:
                if is_websocket_error(cleanup_error):
                    pass
                else:
                    self._log_message(f"Streamer cleanup error: {cleanup_error}")

    def _log_message(self, message: str) -> None:
        """Log a message to the console with real-time output."""
        now = sy.TimeStamp.now()
        timestamp = now.datetime().strftime("%H:%M:%S.%f")[:-4]
        self.logger.info(f"{timestamp} | {self.name} > {message}")

        # Force flush to ensure immediate output in CI
        for handler in self.logger.handlers:
            if hasattr(handler, "flush"):
                handler.flush()

    def _start_client_threads(self) -> None:
        # Start writer thread (writes telemetry at consistent interval)
        self.writer_thread = threading.Thread(target=self._writer_loop, daemon=True)
        self.writer_thread.start()

        # Start streamer thread (reads data on demand)
        self.streamer_thread = threading.Thread(target=self._streamer_loop, daemon=True)
        self.streamer_thread.start()
        self._log_message("Streamer and Writer threads started")

    def _stop_client(self) -> None:
        """Stop client threads and wait for completion."""
        if self.is_running:
            self._should_stop = True
            self.is_running = False

            # Stop streamer thread
            if self.streamer_thread and self.streamer_thread.is_alive():
                self.streamer_thread.join(timeout=5.0)
                if self.streamer_thread.is_alive():
                    self._log_message(
                        "Warning: streamer thread did not stop within timeout"
                    )

            # Stop writer thread
            if self.writer_thread.is_alive():
                self.writer_thread.join(timeout=5.0)
                if self.writer_thread.is_alive():
                    self._log_message(
                        "Warning: writer thread did not stop within timeout"
                    )

        # All done? All done.
        if self._status == STATUS.PENDING:
            self.STATUS = STATUS.PASSED

    def _wait_for_client_completion(self, timeout: Optional[float] = None) -> None:
        """Wait for client threads to complete."""
        # Wait for streamer thread
        if self.streamer_thread.is_alive():
            self.streamer_thread.join(timeout=timeout)

        # Wait for writer thread
        if self.writer_thread.is_alive():
            self.writer_thread.join(timeout=timeout)

    def _check_expectation(self) -> None:
        """Check if test met expected outcome and handle failures gracefully."""
        # Convert PENDING to PASSED if no final status set
        if self._status == STATUS.PENDING:
            self.STATUS = STATUS.PASSED

        status_symbol = SYMBOLS.get_symbol(self._status)
        expected_symbol = SYMBOLS.get_symbol(self.expected_outcome)

        # Handle expected outcome logic
        if self._status == STATUS.PASSED:
            if self.expected_outcome == STATUS.PASSED:
                self._log_message(f"PASSED ({status_symbol})")
            else:
                self.STATUS = STATUS.FAILED
                self._log_message(
                    f"FAILED (❌): Expected {expected_symbol}, got {status_symbol}"
                )

        elif self._status == self.expected_outcome:
            self._log_message(
                f"PASSED (✅): Expected outcome achieved ({status_symbol})"
            )
            # Set _status directly. Setter protects against lower-value statuses. (PASSED)
            self._status = STATUS.PASSED
        elif self._status == STATUS.FAILED:
            self._log_message(f"FAILED ({status_symbol})")
        elif self._status == STATUS.TIMEOUT:
            self._log_message(
                f"TIMEOUT ({status_symbol}): {self._timeout_limit} seconds"
            )
        elif self._status == STATUS.KILLED:
            self._log_message(f"KILLED ({status_symbol})")

        self._log_message(f"Uptime: {self.uptime:.1f} s")
        # Sleep for 2 loops to ensure the status is updated
        sy.sleep(self.DEFAULT_LOOP_RATE * 2)

    def _shutdown(self) -> None:
        """Gracefully shutdown test case and stop all threads."""
        self._log_message("Shutting down test case...")
        self.STATUS = STATUS.KILLED
        self._stop_client()
        self._log_message("Test case shutdown complete")

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
        self, channels: Union[str, List[str]], timeout: Optional[sy.TimeSpan] = 10
    ) -> None:
        """
        Subscribe to channels.
        Can take either a single channel name or a list of channels.
        Timeout is the time (s) to wait for the channels to be initialized.
        """
        self._log_message(f"Subscribing to channels: {channels} ({timeout}s timeout)")

        client = self.client
        time_start = sy.TimeStamp.now()
        timeout_ns = timeout * sy.TimeSpan.SECOND
        found = False

        while self.loop.wait():
            time_now = sy.TimeStamp.now()
            elapsed = time_now - time_start
            if elapsed > timeout_ns:
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
                self._log_message(f"Channel retrieval failed: {e}")
                continue

        if not found:
            raise TimeoutError(f"Unable to retrieve channels: {channels}")

        self._log_message(f"Channels retrieved")
        if isinstance(channels, str):
            self.subscribed_channels.add(channels)
        elif isinstance(channels, list):
            self.subscribed_channels.update(channels)
        return None

    def setup(self) -> None:
        """Load configs, add channels, subscribe to channels, etc."""
        return None

    @abstractmethod
    def run(self) -> None:
        """
        Main test logic method.
        This method must be implemented by all subclasses.
        """
        pass

    def teardown(self) -> None:
        """Cleanup after test execution. Override for custom cleanup logic."""

        # Unload configs
        # or open vents
        # or whatever else
        pass

    def write_tlm(self, channel: str, value: Any = None) -> None:
        """Write values to telemetry dictionary. Can take single key-value or dict of multiple channels."""
        # if isinstance(channel, self.tlm.keys()):
        self.tlm[channel] = value

    @overload
    def read_tlm(
        self, key: str, default: Literal[None] = None
    ) -> Optional[Union[int, float]]:
        ...

    @overload
    def read_tlm(self, key: str, default: Union[int, float]) -> Union[int, float]:
        ...

    def read_tlm(
        self, key: str, default: Optional[Union[int, float]] = None
    ) -> Optional[Union[int, float]]:
        try:
            if self.read_frame is not None:
                value = self.read_frame.get(key, default)
                return value
            else:
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
                    return float(latest_value)

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

    @overload
    def get_state(
        self, key: str, default: Literal[None] = None
    ) -> Optional[Union[int, float]]:
        ...

    @overload
    def get_state(self, key: str, default: Union[int, float]) -> Union[int, float]:
        ...

    def get_state(
        self, key: str, default: Optional[Union[int, float]] = None
    ) -> Optional[Union[int, float]]:
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
                    self.tlm[f"{self.name}_state"] = value.value
                except Exception as e:
                    raise RuntimeError(f"Failed to set status: {e}")
        else:
            self._log_message(f"Invalid status change: {self._status} -> {value}")

    @property
    def uptime(self) -> float:
        """Get the uptime of the test case."""
        return float(self.tlm.get(f"{self.name}_uptime", -1))

    @property
    def time(self) -> float:
        """Get the uptime of the test case."""
        return float(self.tlm.get(f"{self.name}_time", -1))

    @property
    def state(self) -> float:
        """Get the state of the test case."""
        return float(self.tlm.get(f"{self.name}_state", -1))

    @property
    def manual_timeout(self) -> int:
        """Get the manual timeout of the test case."""
        return self._manual_timeout

    @property
    def should_stop(self) -> bool:
        condition_1 = self._manual_timeout >= 0 and self.uptime > self._manual_timeout
        condition_2 = self._should_stop

        return condition_1 or condition_2

    @property
    def should_continue(self) -> bool:
        return not self.should_stop

    def wait_for_tlm_stale(self, buffer_size: int = 5) -> bool:
        """
        Wait for all subscribed channels to be Stale (inactive).
        Requires the last buffer_size frames to be identical.
        """
        self._log_message("Waiting for all channels to be stale (inactive)")

        # Buffer to store the last n vals arrays
        vals_buffer: deque[Any] = deque(maxlen=buffer_size)

        while self.loop.wait() and self.should_continue:
            vals_now = []
            for ch in self.subscribed_channels:
                vals_now.append(self.read_tlm(ch))

            # Add current values to buffer
            vals_buffer.append(vals_now)

            # Check if buffer is full and all entries are identical
            if len(vals_buffer) == buffer_size:
                first_vals = vals_buffer[0]
                if all(vals == first_vals for vals in vals_buffer):
                    self._log_message(
                        f"All channels are stale (last {buffer_size} frames identical)"
                    )
                    return True

        raise TimeoutError("Some Channels remain active")

    def set_manual_timeout(self, value: int) -> None:
        """Set the manual timeout of the test case."""
        self._manual_timeout = value
        self._log_message(f"Manual timeout set ({value}s)")

    def configure(
        self,
        read_timeout: int | None = None,
        loop_rate: float | None = None,
        timeout_limit: int | None = None,
        manual_timeout: int | None = None,
    ) -> None:
        """Configure test case parameters.

        Args:
            read_timeout: Timeout for synnax client read operations (default: 1)
            loop_rate: Synnax Client Loop frequency in Hz (default: 1)
            timeout_limit: Maximum execution time before failure (default: -1, no limit)
            manual_timeout: Manual timeout for test termination (default: -1, no limit)
        """
        params = {}
        if read_timeout is not None:
            self._read_timeout = read_timeout
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

        self._log_message(f"Configured with: {params}")

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

    def fail(self, message: Optional[str] = None) -> None:
        if message is not None:
            self._log_message(f"FAILED: {message}")
        self.STATUS = STATUS.FAILED

    def execute(self) -> None:
        """Execute complete test lifecycle: setup -> run -> teardown."""
        try:
            # Set STATUSat the top level as opposed to within
            # the override methods. Ensures that the status is set
            # Even if the child classes don't call super()

            self.STATUS = STATUS.INITIALIZING
            self.setup()

            self._start_client_threads()

            self.STATUS = STATUS.RUNNING
            self.run()

            # Set to PENDING only if not in final state
            if self._status not in [STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED]:
                self.STATUS = STATUS.PENDING

            self.teardown()

            # PASSED set in _check_expectation()

        except Exception as e:
            if is_websocket_error(e):
                pass
            else:
                self.STATUS = STATUS.FAILED
                self._log_message(f"EXCEPTION: {e}\n{traceback.format_exc()}")
        finally:
            self._check_expectation()
            self._stop_client()
            self._wait_for_client_completion()

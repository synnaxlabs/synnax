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
import os
import re
import sys
import threading
import time
import traceback
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from enum import Enum, auto
from selectors import SelectorKey
from typing import Any

import synnax as sy

from framework.utils import (
    WebSocketErrorFilter,
    ignore_websocket_errors,
    validate_and_sanitize_name,
)

# Error filter
sys.excepthook = ignore_websocket_errors
sys.stderr = WebSocketErrorFilter()


def is_websocket_error(error: Exception) -> bool:
    """Check if an exception is a WebSocket-related error that should be ignored."""
    error_str = str(error)
    return any(
        phrase in error_str
        for phrase in [
            "1011",
            "keepalive ping timeout",
            "keepalive ping failed",
            "timed out while closing connection",
            "ConnectionClosedError",
            "WebSocketException",
        ]
    )


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
    PASSED = "✅"  # Green checkmark
    FAILED = "❌"  # Red X
    KILLED = "💀"  # Skull
    TIMEOUT = "⏰"  # Alarm clock

    @classmethod
    def get_symbol(cls, status):
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
    DEFAULT_READ_TIMEOUT = 1
    DEFAULT_LOOP_RATE = 1
    WEBSOCKET_RETRY_DELAY = 1
    MAX_CLEANUP_RETRIES = 3
    CLIENT_THREAD_START_DELAY = 1
    DEFAULT_TIMEOUT_LIMIT = -1
    DEFAULT_MANUAL_TIMEOUT = -1

    def __init__(
        self,
        synnax_connection: SynnaxConnection,
        name: str = None,
        expect: str = "PASSED",
        **params,
    ):

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
        self._timeout_limit: int = self.DEFAULT_TIMEOUT_LIMIT  # -1 = no timeout
        self._manual_timeout: int = self.DEFAULT_MANUAL_TIMEOUT
        self.read_frame = None
        self.read_timeout = self.DEFAULT_READ_TIMEOUT

        if name is None:
            # Convert PascalCase class name to lowercase with underscores
            self.name = re.sub(
                r"([a-z0-9])([A-Z])", r"\1_\2", self.__class__.__name__
            ).lower()
        else:
            self.name = validate_and_sanitize_name(name)

        self._setup_logging()
        self._status = STATUS.INITIALIZING

        self.client = sy.Synnax(
            host=synnax_connection.server_address,
            port=synnax_connection.port,
            username=synnax_connection.username,
            password=synnax_connection.password,
            secure=synnax_connection.secure,
        )

        self.loop = sy.Loop(self.DEFAULT_LOOP_RATE)
        self.client_thread = None
        self._should_stop = False
        self.is_running = True

        self.subscribed_channels = []

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

    def __enter__(self):
        """Context manager entry point."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit point - ensures cleanup."""
        self._shutdown()

    def _setup_logging(self) -> None:
        """Setup logging for real-time output (same approach as TestConductor)."""
        # Check if running in CI environment
        is_ci = any(
            env_var in os.environ
            for env_var in ["CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"]
        )

        # Force unbuffered output in CI environments
        if is_ci:
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
            if hasattr(handler.stream, "flush"):
                handler.flush = lambda h=handler: h.stream.flush()

    def _writer_loop(self) -> None:
        """Writer thread that writes telemetry at consistent interval."""
        start_time = sy.TimeStamp.now()
        client = None

        try:
            client = self.client.open_writer(
                start=start_time,
                channels=list(self.tlm.keys()),
                name=self.name,
                enable_auto_commit=True,
            )

            while self.loop.wait() and not self._should_stop:
                """
                # Update telemetry

                Keep outside of try/except block to ensure
                we continue to update the internal state of the
                TestCase object. If not updated, then the logic
                dictating if the test should continue will not function.
                """

                now = sy.TimeStamp.now()
                uptime_value = (now - start_time) / 1e9
                self.tlm[f"{self.name}_time"] = now
                self.tlm[f"{self.name}_uptime"] = uptime_value
                self.tlm[f"{self.name}_state"] = self._status.value

                # Check for timeout
                if self._timeout_limit > 0 and uptime_value > self._timeout_limit:
                    self.STATUS = STATUS.TIMEOUT

                # Check for completion
                if self._status in [STATUS.FAILED, STATUS.KILLED, STATUS.TIMEOUT]:
                    self.tlm[f"{self.name}_state"] = self._status.value
                    self._should_stop = True

                try:
                    client.write(self.tlm)
                except Exception as e:
                    if is_websocket_error(e):
                        time.sleep(self.WEBSOCKET_RETRY_DELAY)
                    else:
                        self.STATUS = STATUS.FAILED
                        raise e

                # Final write attempt for redundancy
                try:
                    client.write(self.tlm)
                except:
                    pass
            self._log_message("writer shutting down")

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
            self._log_message(f"Channels subscribed: {self.subscribed_channels}")
            for channel in self.subscribed_channels:
                self.read_frame[channel] = 0
        else:
            return

        try:
            streamer = self.client.open_streamer(self.subscribed_channels)

            while not self._should_stop:
                try:
                    # Read data on demand with timeout (not tied to loop.wait())
                    self.frame_in = streamer.read(self.read_timeout)
                    if self.frame_in is not None:
                        for key, value in self.frame_in.items():
                            self.read_frame[key] = value[-1]

                except Exception as e:
                    if is_websocket_error(e):
                        time.sleep(self.WEBSOCKET_RETRY_DELAY)
                    else:
                        self._log_message(f"Streamer error: {e}")
                        break

            self._log_message("streamer shutting down")

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
        self.logger.info(f"{self.name} > {message}")

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

        time.sleep(self.CLIENT_THREAD_START_DELAY)  # Allow threads to start
        self._log_message("Streamer and Writer threads started")

    def _stop_client(self) -> None:
        """Stop client threads and wait for completion."""
        if self.is_running:
            self._should_stop = True
            self.is_running = False

            # Stop streamer thread
            if (
                hasattr(self, "streamer_thread")
                and self.streamer_thread
                and self.streamer_thread.is_alive()
            ):
                self.streamer_thread.join(timeout=5.0)
                if self.streamer_thread.is_alive():
                    self._log_message(
                        "Warning: streamer thread did not stop within timeout"
                    )

            # Stop writer thread
            if (
                hasattr(self, "writer_thread")
                and self.writer_thread
                and self.writer_thread.is_alive()
            ):
                self.writer_thread.join(timeout=5.0)
                if self.writer_thread.is_alive():
                    self._log_message(
                        "Warning: writer thread did not stop within timeout"
                    )

        # All done? All done.
        if self._status == STATUS.PENDING:
            self.STATUS = STATUS.PASSED

    def _wait_for_client_completion(self, timeout: float = None) -> None:
        """Wait for client threads to complete."""
        # Wait for streamer thread
        if (
            hasattr(self, "streamer_thread")
            and self.streamer_thread
            and self.streamer_thread.is_alive()
        ):
            self.streamer_thread.join(timeout=timeout)

        # Wait for writer thread
        if (
            hasattr(self, "writer_thread")
            and self.writer_thread
            and self.writer_thread.is_alive()
        ):
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

    def _shutdown(self) -> None:
        """Gracefully shutdown test case and stop all threads."""
        self._log_message("Shutting down test case...")
        self.STATUS = STATUS.KILLED
        self.stop_client()
        self._log_message("Test case shutdown complete")

    def add_channel(
        self,
        name: str,
        data_type: sy.DataType,
        initial_value: Any = None,
        append_name: bool = True,
    ):
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

    def subscribe(self, channels) -> None:
        """Subscribe to channels. Can take either a single channel name or a list of channels."""
        if isinstance(channels, str):
            # Single channel name
            self.subscribed_channels.append(channels)
        elif isinstance(channels, list):
            # List of channels - extend the list
            self.subscribed_channels.extend(channels)
        else:
            # Convert to string if it's another type
            self.subscribed_channels.append(str(channels))
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
        # else:
        #    raise KeyError(f"Key {channel} not found in telemetry dictionary ({self.tlm.keys()})")

    def read_tlm(self, key: str, default: Any = None) -> Any:

        try:
            return self.read_frame.get(key, default)
        except:
            return default

    def get_state(self, key: str, default: Any = None) -> Any:
        """
        Easily get state of this object.

        - self.name + "state"
        - self.name + "time"
        - self.name + "uptime"
        """

        name_ch = self.name + "_" + key
        return self.tlm.get(name_ch, default)

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
                except Exception:
                    pass  # Ignore errors if tlm is not fully initialized
        else:
            self._log_message(f"Invalid status change: {self._status} -> {value}")

    def _set_status(self, value: STATUS) -> STATUS:
        """Set the test status and update telemetry if client is running."""
        # Alternative to STATUS setter that returns the status
        self.STATUS = value
        return self.STATUS

    @property
    def uptime(self) -> float:
        """Get the uptime of the test case."""
        return self.tlm.get(f"{self.name}_uptime", -1)

    @property
    def time(self) -> float:
        """Get the uptime of the test case."""
        return self.tlm.get(f"{self.name}_time", -1)

    @property
    def state(self) -> float:
        """Get the state of the test case."""
        return self.tlm.get(f"{self.name}_state", -1)

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

    def set_manual_timeout(self, value: int) -> None:
        """Set the manual timeout of the test case."""
        self._manual_timeout = value
        self._log_message(f"Manual timeout set ({value}s)")

    def configure(self, **kwargs) -> None:
        """Configure test case parameters.

        Args:
            read_timeout: Timeout for read operations (default: 1)
            loop_rate: Loop frequency in Hz (default: 1)
            websocket_retry_delay: Delay before retrying WebSocket operations (default: 1)
            timeout_limit: Maximum execution time in seconds (default: -1, no limit)
            manual_timeout: Manual timeout value (default: -1, no limit)
        """
        if "read_timeout" in kwargs:
            self.read_timeout = kwargs["read_timeout"]
        if "loop_rate" in kwargs:
            self.loop = sy.Loop(kwargs["loop_rate"])
        if "timeout_limit" in kwargs:
            self._timeout_limit = kwargs["timeout_limit"]
        if "manual_timeout" in kwargs:
            self._manual_timeout = kwargs["manual_timeout"]
        self._log_message(f"Configured with: {kwargs}")

    def is_client_running(self) -> bool:
        """Check if client threads are running."""
        streamer_running = (
            hasattr(self, "streamer_thread")
            and self.streamer_thread
            and self.streamer_thread.is_alive()
        )
        writer_running = (
            hasattr(self, "writer_thread")
            and self.writer_thread
            and self.writer_thread.is_alive()
        )
        return streamer_running or writer_running

    def get_client_status(self) -> str:
        """Get client thread status."""
        streamer_status = "Not started"
        writer_status = "Not started"

        if hasattr(self, "streamer_thread") and self.streamer_thread:
            streamer_status = (
                "Running" if self.streamer_thread.is_alive() else "Stopped"
            )

        if hasattr(self, "writer_thread") and self.writer_thread:
            writer_status = "Running" if self.writer_thread.is_alive() else "Stopped"

        return f"Streamer: {streamer_status}, Writer: {writer_status}"

    def get_streamer_status(self) -> str:
        """Get streamer thread status."""
        if not hasattr(self, "streamer_thread") or self.streamer_thread is None:
            return "Not started"
        return "Running" if self.streamer_thread.is_alive() else "Stopped"

    def get_writer_status(self) -> str:
        """Get writer thread status."""
        if not hasattr(self, "streamer_thread") or self.writer_thread is None:
            return "Not started"
        return "Running" if self.writer_thread.is_alive() else "Stopped"

    def fail(self) -> None:
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
                self._log_message(f"EXCEPTION: {e}")
        finally:
            self._stop_client()
            self._wait_for_client_completion()
            self._check_expectation()

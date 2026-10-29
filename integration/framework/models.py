#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass
from enum import Enum, auto
from typing import Protocol

import synnax as sy


class Log(Protocol):
    def __call__(self, message: str, /, *, use_name: bool = True) -> None: ...


class STATUS(Enum):
    """Enum representing the status of a test. A test only moves to a later status."""

    INITIALIZING = auto()
    RUNNING = auto()
    PENDING = auto()
    PASSED = auto()
    FLAKY = auto()
    # Neither passes nor fails the run. A later FAILED, as from `_try_to`, wins.
    SKIPPED = auto()
    FAILED = auto()
    TIMEOUT = auto()
    KILLED = auto()


class SYMBOLS(Enum):
    PASSED = "\u2705"
    FLAKY = "\u26a0\ufe0f"
    SKIPPED = "\u23ed\ufe0f"
    FAILED = "\u274c"
    KILLED = "\U0001f480"
    TIMEOUT = "\u23f0"

    @classmethod
    def get_symbol(cls, status: STATUS) -> str:
        """Get symbol for a given status, with fallback to '?' if not found."""
        try:
            return cls[status.name].value
        except (KeyError, AttributeError):
            return "\u2753"


@dataclass(frozen=True)
class SynnaxConnection:
    """Data class representing the Synnax connection parameters."""

    server_address: str = "localhost"
    port: int = 9090
    username: str = "synnax"
    password: str = "seldon"
    secure: bool = False
    ca_cert: str | None = None

    def create_client(self) -> sy.Synnax:
        return sy.Synnax(
            host=self.server_address,
            port=self.port,
            username=self.username,
            password=self.password,
            secure=self.secure,
            ca_cert=self.ca_cert,
        )


DEFAULT_CONNECTION = SynnaxConnection()
"""Connection to a local Core with the default credentials."""


@dataclass(frozen=True)
class RunOptions:
    """Conductor command-line options that test cases read."""

    logs_streamed: bool = False
    headed: bool = False
    slow_mo: int = 0
    driver_rack: str = "Node 1 Embedded Driver"


DEFAULT_OPTIONS = RunOptions()
"""Options of a conductor run with no flags."""


@dataclass
class Test:
    """Data class to store test execution results."""

    test_name: str
    status: STATUS
    name: str | None = None
    error_message: str | None = None
    error_traceback: str | None = None
    skip_reason: str | None = None
    range: sy.Range | None = None
    # Unix-timestamp seconds bounding the test's wall-clock execution. Populated
    # by ExecutionClient; used by summary.json and by per-test server-log slicing.
    started_at: float | None = None
    ended_at: float | None = None
    # Absolute path to this test's bundle subdirectory under the run dir.
    bundle_dir: str | None = None

    def __str__(self) -> str:
        """Return display name for test result."""
        if self.name and self.name != self.test_name.split("/")[-1]:
            return f"{self.test_name} ({self.name})"
        return self.test_name

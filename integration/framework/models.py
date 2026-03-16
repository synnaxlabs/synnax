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

import synnax as sy


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
    PASSED = "\u2705"
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


@dataclass
class Test:
    """Data class to store test execution results."""

    test_name: str
    status: STATUS
    name: str | None = None
    error_message: str | None = None
    range: sy.Range | None = None

    def __str__(self) -> str:
        """Return display name for test result."""
        if self.name and self.name != self.test_name.split("/")[-1]:
            return f"{self.test_name} ({self.name})"
        return self.test_name

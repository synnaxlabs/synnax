#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import ABC, abstractmethod

import synnax as sy


class Simulator(ABC):
    """Base class for all simulators."""

    def __init__(self, verbose: bool = False):
        self.verbose = verbose
        self._running = False

    def log(self, message: str) -> None:
        """Print message only when verbose mode is enabled."""
        if self.verbose:
            print(f"[{self.__class__.__name__}] {message}")

    @abstractmethod
    def start(self) -> None:
        """Start the simulator."""
        ...

    @abstractmethod
    def stop(self, timeout: sy.TimeSpan = 5 * sy.TimeSpan.SECOND) -> None:
        """Stop the simulator."""
        ...

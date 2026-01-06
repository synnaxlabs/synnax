#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Protocol definitions for console profiling."""

from typing import Protocol

from console.profiling.config import ProfilerConfig


class Profiler(Protocol):
    """Protocol for a profiler that collects performance data from browser sessions.

    Implementations should handle the full lifecycle of profiling:
    1. Start profiling when the session begins
    2. Collect data during the session
    3. Stop profiling and save results when the session ends
    """

    @property
    def config(self) -> ProfilerConfig:
        """Get the profiler configuration.

        :returns: The profiler configuration.
        """
        ...

    def start(self) -> None:
        """Start all enabled profiling features.

        Should be called after the browser page is created but before
        navigation to the application under test.
        """
        ...

    def stop(self, test_name: str) -> None:
        """Stop all profiling and save results.

        :param test_name: Name of the test, used for output file names.
        """
        ...

    def close(self) -> None:
        """Clean up any resources held by the profiler.

        Called during teardown to release CDP sessions, file handles, etc.
        """
        ...

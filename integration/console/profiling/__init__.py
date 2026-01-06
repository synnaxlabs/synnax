#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Console profiling utilities for Playwright-based tests.

This module provides a clean interface for collecting performance data from
Playwright browser sessions using the Chrome DevTools Protocol (CDP).
"""

from console.profiling.client import CDPProfiler
from console.profiling.config import ProfilerConfig
from console.profiling.protocol import Profiler
from console.profiling.writer import ProfileWriter

__all__ = [
    "CDPProfiler",
    "Profiler",
    "ProfilerConfig",
    "ProfileWriter",
]

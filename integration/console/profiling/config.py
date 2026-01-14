#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Configuration for console profiling."""

import os
from dataclasses import dataclass, field
from pathlib import Path


def _env_bool(key: str, default: bool) -> bool:
    """Read a boolean from environment variable."""
    return os.environ.get(key, "1" if default else "0") == "1"


@dataclass
class ProfilerConfig:
    """Configuration for console profiling.

    Controls which profiling features are enabled and where profiles are saved.

    Environment Variables:
        PLAYWRIGHT_CONSOLE_PROFILE: Enable CPU profiling via CDP (default: False)
        PLAYWRIGHT_CONSOLE_TRACE: Enable Playwright tracing (default: False)
        PLAYWRIGHT_CONSOLE_HEAP: Enable heap snapshot via CDP (default: False)

    Attributes:
        cpu_profiling: Enable CPU profiling. Profiles saved as .cpuprofile files.
        tracing: Enable Playwright tracing. Traces saved as .trace.zip files.
        heap_snapshot: Enable heap snapshots. Saved as .heapsnapshot files.
        output_dir: Directory where profile files are saved.
    """

    cpu_profiling: bool = field(
        default_factory=lambda: _env_bool("PLAYWRIGHT_CONSOLE_PROFILE", False)
    )
    tracing: bool = field(
        default_factory=lambda: _env_bool("PLAYWRIGHT_CONSOLE_TRACE", False)
    )
    heap_snapshot: bool = field(
        default_factory=lambda: _env_bool("PLAYWRIGHT_CONSOLE_HEAP", False)
    )
    output_dir: Path = field(
        default_factory=lambda: Path(__file__).parent.parent.parent / "profiles"
    )

    @property
    def requires_cdp(self) -> bool:
        """Whether any CDP-based profiling is enabled."""
        return self.cpu_profiling or self.heap_snapshot

    @classmethod
    def from_params(cls, params: dict) -> "ProfilerConfig":
        """Create config from test parameters, falling back to environment variables.

        :param params: Test parameters dictionary.
        :returns: ProfilerConfig instance.
        """
        return cls(
            cpu_profiling=params.get(
                "profile", _env_bool("PLAYWRIGHT_CONSOLE_PROFILE", False)
            ),
            tracing=params.get("trace", _env_bool("PLAYWRIGHT_CONSOLE_TRACE", False)),
            heap_snapshot=params.get(
                "heap", _env_bool("PLAYWRIGHT_CONSOLE_HEAP", False)
            ),
        )

    @classmethod
    def disabled(cls) -> "ProfilerConfig":
        """Create a config with all profiling disabled.

        :returns: ProfilerConfig with all features disabled.
        """
        return cls(
            cpu_profiling=False,
            tracing=False,
            heap_snapshot=False,
        )

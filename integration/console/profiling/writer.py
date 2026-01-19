#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Profile writer for saving profiling results to disk."""

import json
from pathlib import Path
from typing import Any


class ProfileWriter:
    """Writes profiling results to disk.

    Handles file creation and formatting for various profile types including
    CPU profiles and heap snapshots.

    :param output_dir: Directory where profile files are saved.
    """

    def __init__(self, output_dir: Path) -> None:
        self._output_dir = output_dir
        self._output_dir.mkdir(exist_ok=True)

    def write_cpu_profile(self, test_name: str, profile: dict[str, Any]) -> Path:
        """Write a CPU profile to disk.

        :param test_name: Name of the test for the output file.
        :param profile: CPU profile data from CDP Profiler.stop.
        :returns: Path to the saved profile file.
        """
        path = self._output_dir / f"{test_name}.cpuprofile"
        with open(path, "w") as f:
            json.dump(profile, f)
        return path

    def write_heap_snapshot(self, test_name: str, chunks: list[str]) -> Path:
        """Write a heap snapshot to disk.

        :param test_name: Name of the test for the output file.
        :param chunks: Heap snapshot chunks from CDP HeapProfiler.
        :returns: Path to the saved snapshot file.
        """
        path = self._output_dir / f"{test_name}.heapsnapshot"
        content = "".join(chunks)
        with open(path, "w") as f:
            f.write(content)
        return path

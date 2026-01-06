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
    CPU profiles, metrics, coverage, and heap snapshots.

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

    def write_metrics(self, test_name: str, raw_metrics: dict[str, float]) -> Path:
        """Write performance metrics to disk.

        :param test_name: Name of the test for the output file.
        :param raw_metrics: Raw metrics from CDP Performance.getMetrics.
        :returns: Path to the saved metrics file.
        """
        metrics = self._format_metrics(test_name, raw_metrics)
        path = self._output_dir / f"{test_name}.metrics.json"
        with open(path, "w") as f:
            json.dump(metrics, f, indent=2)
        return path

    def write_coverage(self, test_name: str, coverage_data: list[dict]) -> Path:
        """Write code coverage data to disk.

        :param test_name: Name of the test for the output file.
        :param coverage_data: Coverage data from CDP Profiler.takePreciseCoverage.
        :returns: Path to the saved coverage file.
        """
        # Filter to only include app code (exclude node_modules, extensions)
        app_coverage = [
            entry
            for entry in coverage_data
            if entry.get("url", "").startswith("http")
            and "node_modules" not in entry.get("url", "")
        ]
        path = self._output_dir / f"{test_name}.coverage.json"
        with open(path, "w") as f:
            json.dump(app_coverage, f, indent=2)
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

    def _format_metrics(
        self, test_name: str, raw_metrics: dict[str, float]
    ) -> dict[str, Any]:
        """Format raw metrics into a structured report.

        :param test_name: Name of the test.
        :param raw_metrics: Raw metrics from CDP.
        :returns: Formatted metrics dictionary.
        """
        return {
            "test_name": test_name,
            "memory": {
                "js_heap_used_mb": round(
                    raw_metrics.get("JSHeapUsedSize", 0) / 1024 / 1024, 2
                ),
                "js_heap_total_mb": round(
                    raw_metrics.get("JSHeapTotalSize", 0) / 1024 / 1024, 2
                ),
            },
            "dom": {
                "nodes": int(raw_metrics.get("Nodes", 0)),
                "documents": int(raw_metrics.get("Documents", 0)),
                "frames": int(raw_metrics.get("Frames", 0)),
                "js_event_listeners": int(raw_metrics.get("JSEventListeners", 0)),
            },
            "layout": {
                "layout_count": int(raw_metrics.get("LayoutCount", 0)),
                "recalc_style_count": int(raw_metrics.get("RecalcStyleCount", 0)),
                "layout_duration_ms": round(
                    raw_metrics.get("LayoutDuration", 0) * 1000, 2
                ),
                "recalc_style_duration_ms": round(
                    raw_metrics.get("RecalcStyleDuration", 0) * 1000, 2
                ),
            },
            "script": {
                "script_duration_ms": round(
                    raw_metrics.get("ScriptDuration", 0) * 1000, 2
                ),
                "task_duration_ms": round(
                    raw_metrics.get("TaskDuration", 0) * 1000, 2
                ),
            },
            "raw": raw_metrics,
        }

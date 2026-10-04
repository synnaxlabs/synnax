#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import csv
import threading
import time
from pathlib import Path

import sample_resources


def start(path: Path) -> threading.Thread:
    thread = threading.Thread(target=sample_resources.sample, args=(path, 0.01))
    thread.start()
    return thread


def wait_for_samples(path: Path, count: int) -> None:
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline:
        if path.exists():
            with open(path, newline="", encoding="utf-8") as f:
                if len(list(csv.DictReader(f))) >= count:
                    return
        time.sleep(0.01)
    raise TimeoutError(f"fewer than {count} samples in {path}")


class TestPercentile:
    """Tests for the nearest-rank percentile."""

    def test_should_return_the_nearest_rank_at_p95(self) -> None:
        values = [float(v) for v in range(1, 21)]
        assert sample_resources.percentile(values, 0.95) == 19

    def test_should_return_the_median_of_an_odd_count(self) -> None:
        assert sample_resources.percentile([3.0, 1.0, 2.0], 0.5) == 2

    def test_should_return_the_only_value(self) -> None:
        assert sample_resources.percentile([7.0], 0.95) == 7

    def test_should_return_the_minimum_at_p0(self) -> None:
        assert sample_resources.percentile([3.0, 1.0, 2.0], 0) == 1


class TestSample:
    """Tests for the sampling loop and its stop file."""

    def test_should_stop_when_reported(self, tmp_path: Path) -> None:
        path = tmp_path / "resources.csv"
        thread = start(path)
        wait_for_samples(path, 2)
        summary = sample_resources.report(path)
        thread.join(timeout=5)
        assert not thread.is_alive()
        assert "| CPU % |" in summary

    def test_should_ignore_a_stale_stop_file(self, tmp_path: Path) -> None:
        path = tmp_path / "resources.csv"
        sample_resources.stop_path(path).touch()
        thread = start(path)
        wait_for_samples(path, 1)
        sample_resources.report(path)
        thread.join(timeout=5)
        assert not thread.is_alive()


class TestReport:
    """Tests for the Markdown summary."""

    def test_should_report_no_samples(self, tmp_path: Path) -> None:
        path = tmp_path / "resources.csv"
        path.write_text(",".join(sample_resources.FIELDS) + "\n")
        assert sample_resources.report(path) == f"No resource samples in {path}.\n"

    def test_should_summarize_samples(self, tmp_path: Path) -> None:
        path = tmp_path / "resources.csv"
        path.write_text("time,cpu_percent,memory_used_gb\n0,10,4\n60,30,6\n")
        summary = sample_resources.report(path)
        assert "2 samples over 1.0 minutes." in summary
        assert "| CPU % | 20 | 10 | 30 | 30 |" in summary
        assert "| Memory GB | 5.0 | 4.0 | 6.0 | 6.0 |" in summary

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Sample Resources - Records machine CPU and memory use while a test job runs.

`sample` writes one CSV line every interval until `report` stops it. `report` then
summarizes the CSV to stdout and, inside GitHub Actions, to the job summary.
"""

import argparse
import csv
import math
import os
import statistics
import sys
import time
from pathlib import Path

import psutil

FIELDS = ["time", "cpu_percent", "memory_used_gb"]
GB = 1024**3


def stop_path(path: Path) -> Path:
    """Returns the file whose existence stops the sampler writing to path."""
    return path.with_suffix(".stop")


def sample(path: Path, interval: float) -> None:
    """Writes a sample to the CSV at path every interval seconds until stopped."""
    stop_path(path).unlink(missing_ok=True)
    with open(path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(FIELDS)
        while not stop_path(path).exists():
            cpu = psutil.cpu_percent(interval=interval)
            memory = psutil.virtual_memory()
            used = (memory.total - memory.available) / GB
            writer.writerow([f"{time.time():.0f}", f"{cpu:.1f}", f"{used:.2f}"])
            f.flush()


def percentile(values: list[float], fraction: float) -> float:
    """Returns the nearest-rank percentile of values."""
    ordered = sorted(values)
    return ordered[max(0, math.ceil(len(ordered) * fraction) - 1)]


def report(path: Path) -> str:
    """Stops the sampler and returns a Markdown summary of its samples."""
    stop_path(path).touch()
    with open(path, newline="", encoding="utf-8") as f:
        rows = list(csv.DictReader(f))
    if not rows:
        return f"No resource samples in {path}.\n"
    cpu = [float(r["cpu_percent"]) for r in rows]
    memory = [float(r["memory_used_gb"]) for r in rows]
    total = psutil.virtual_memory().total / GB
    minutes = (float(rows[-1]["time"]) - float(rows[0]["time"])) / 60
    return (
        f"### Resource use on {psutil.cpu_count()} CPUs and {total:.0f} GB\n\n"
        f"{len(rows)} samples over {minutes:.1f} minutes.\n\n"
        "| Metric | Mean | p50 | p95 | Max |\n"
        "| --- | --- | --- | --- | --- |\n"
        f"| CPU % | {statistics.mean(cpu):.0f} | {percentile(cpu, 0.5):.0f} | "
        f"{percentile(cpu, 0.95):.0f} | {max(cpu):.0f} |\n"
        f"| Memory GB | {statistics.mean(memory):.1f} | {percentile(memory, 0.5):.1f} | "
        f"{percentile(memory, 0.95):.1f} | {max(memory):.1f} |\n"
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    sample_parser = commands.add_parser("sample", help="record samples until reported")
    sample_parser.add_argument("csv", type=Path)
    sample_parser.add_argument("--interval", type=float, default=5.0)
    report_parser = commands.add_parser("report", help="summarize recorded samples")
    report_parser.add_argument("csv", type=Path)
    args = parser.parse_args()

    if args.command == "sample":
        sample(args.csv, args.interval)
        return
    summary = report(args.csv)
    sys.stdout.write(summary)
    step_summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if step_summary:
        with open(step_summary, "a", encoding="utf-8") as f:
            f.write(summary)


if __name__ == "__main__":
    main()

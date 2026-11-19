#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Compares the gate benchmarks of two Git refs and fails on a regression.

Usage: uv run --project scripts scripts/compare_benchmarks.py --base main --head HEAD
    [--rounds 10] [--suite cpu|io] [--io-dir <dir>] [--only <regex>] [--out <dir>]

Builds each ref once in its own worktree. Each round then runs every group once per
ref, back to back, and odd rounds run head first. One process runs at a time.
benchstat compares the two result files.

A benchmark fails on a sec/op rise over 5% at p < 0.05, on an allocs/op rise of any
size at p < 0.05, or when head does not have it. A failure exits with 1. A control
group never fails the run: a control that breaks the rule means the host is noisy.

Pass one ref as both --base and --head for an A/A run, which measures the noise of the
host. The io suite puts its files in --io-dir, the disk to measure.
"""

import csv
import io
from dataclasses import dataclass

SIDES = ("base", "head")
MAX_TIME_RISE = 0.05


@dataclass(frozen=True)
class Finding:
    """One benchmark that breaks the rule, or that only head has."""

    # One of "slower", "allocations", "missing", and "new".
    kind: str
    package: str
    benchmark: str
    detail: str = ""


def judge(table: str) -> list[Finding]:
    """Applies the rule to the CSV that benchstat prints for a base and a head file."""
    findings = []
    package = unit = ""
    for row in csv.reader(io.StringIO(table)):
        if not row or row[0] == "geomean":
            continue
        if row[0].startswith("pkg: "):
            package = row[0].removeprefix("pkg: ")
        if len(row) == 1:
            continue
        if row[0] == "":
            unit = row[1] if row[1] not in SIDES else unit
            continue
        name, base, _, head, _, change = (*row, "", "", "", "")[:6]
        if base == "" or head == "":
            if unit == "sec/op":
                kind = "missing" if head == "" else "new"
                findings.append(Finding(kind, package, name))
            continue
        before, after = float(base), float(head)
        if change in ("", "~") or after <= before:
            continue
        if unit == "sec/op" and after > before * (1 + MAX_TIME_RISE):
            detail = f"{after / before - 1:+.2%}"
            findings.append(Finding("slower", package, name, detail))
        if unit == "allocs/op":
            detail = f"{before:g} to {after:g} allocs/op"
            findings.append(Finding("allocations", package, name, detail))
    return findings

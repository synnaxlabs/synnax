#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Defines the integration CI jobs and prints their GitHub Actions matrix.

Each job runs on its own machine and starts one Core per part, running its parts at
once. Tests that share a machine-wide resource (a simulator port, the NI devices) must
sit in the same part. Only the first Core runs the NI integration. Uses only the
standard library, so CI can print the matrix without installing the framework.
"""

import json
from dataclasses import dataclass


@dataclass(frozen=True)
class Job:
    """The tests that one CI job runs, split across Cores."""

    parts: tuple[tuple[str, ...], ...]
    """Test conductor arguments for each Core. A part's runs go one after another on
    its Core; the parts run at once."""
    after: tuple[str, ...] = ()
    """Test conductor arguments of runs on the first Core after every part finishes."""


EXEMPT = ("example", "migration")
"""Prefixes of test files that run in dedicated workflows."""

PRODUCTS = {
    "arc": "arc",
    "console": "console",
    "control": "arc",
    "driver": "driver",
    "latency": "arc",
}
"""The product each test file covers, by the file's prefix."""

# Together, the jobs run every test of every test file outside EXEMPT exactly once.
# test-split checks this before each job starts.
JOBS: dict[str, Job] = {
    # Every Arc case runs once per rack: 65537 on the embedded Driver, 65538 on the
    # Core's own rack. A latency case uses NI, so latency stays on the first Core.
    "arc": Job(
        parts=(
            ("arc,latency -x _65538", "console/user/..."),
            ("arc,control -x _65537", "console/lifecycle/..."),
        )
    ),
    # The project sequence uses the OPC UA simulator and pages holds the NI forms.
    "console": Job(
        parts=(("console/project,other,cluster/...",), ("console/pages/...",))
    ),
    # A simulator port belongs to one part. grand_finale uses OPC UA, Modbus, and NI.
    "driver": Job(
        parts=(
            ("driver/ni_,modbus", "console/channel/... -x stress"),
            ("driver/opcua,http,pagerduty", "console/channel_stress/..."),
        ),
        after=("driver/grand_finale",),
    ),
}


def title(name: str, job: Job) -> str:
    """Returns the job's name followed by the other products whose tests it runs, as
    in "arc + console"."""
    runs = (*(a for part in job.parts for a in part), *job.after)
    files = {f for run in runs for f in run.split()[0].split("/")[0].split(",")}
    others = sorted({PRODUCTS[f] for f in files} - {name})
    return " + ".join((name, *others))


def main() -> None:
    """Prints the job matrix as a GitHub Actions output line."""
    jobs = [{"name": name, "title": title(name, job)} for name, job in JOBS.items()]
    matrix = {"include": jobs}
    print(f"TEST_MATRIX={json.dumps(matrix, separators=(',', ':'))}")


if __name__ == "__main__":
    main()

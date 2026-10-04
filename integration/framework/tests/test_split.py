#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import replace

from framework.split import check
from jobs import JOBS, Job


class TestCheck:
    """Tests for the check that the CI jobs run every test exactly once."""

    def test_should_pass_for_the_ci_jobs(self) -> None:
        assert check(JOBS) == []

    def test_should_report_tests_no_job_runs(self) -> None:
        jobs = {name: job for name, job in JOBS.items() if name != "console"}
        problems = check(jobs)
        assert problems
        assert all(p.startswith("missing: console") for p in problems)

    def test_should_report_tests_run_twice(self) -> None:
        jobs = {**JOBS, "again": Job(parts=(("driver/pagerduty",),))}
        assert check(jobs) == [
            "run 2 times: client/pagerduty_alert",
        ]

    def test_should_report_tests_of_exempt_files(self) -> None:
        driver = replace(JOBS["driver"], after=("driver/grand_finale", "example"))
        problems = check({**JOBS, "driver": driver})
        assert problems
        assert all(p.startswith("not in a test file: example") for p in problems)

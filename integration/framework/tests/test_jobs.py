#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from jobs import EXEMPT, JOBS, PRODUCTS, Job, title

TESTS = Path(__file__).resolve().parents[2] / "tests"


class TestProducts:
    """Tests for the product of each test file."""

    def test_should_cover_every_test_file(self) -> None:
        prefixes = {
            p.name.removesuffix("_tests.json") for p in TESTS.glob("*_tests.json")
        }
        assert prefixes - set(EXEMPT) == set(PRODUCTS)


class TestTitle:
    """Tests for the name each CI job shows."""

    def test_should_list_the_other_products_a_job_runs(self) -> None:
        assert {name: title(name, job) for name, job in JOBS.items()} == {
            "arc": "arc + console",
            "console": "console",
            "driver": "driver + console",
        }

    def test_should_read_files_of_every_run(self) -> None:
        job = Job(parts=(("arc,latency -x _65538",),), after=("driver/grand_finale",))
        assert title("arc", job) == "arc + driver"

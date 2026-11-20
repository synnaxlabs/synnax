#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "image_channel.sh"

TAGS = [
    "core/v0.10.0",
    "core/v0.59.0",
    "core/v0.60.0",
    "core/v0.60.1",
    "core/v0.61.0-rc.1",
    "core/v0.61.0-rc.2",
]


def channel(name: str, version: str, tags: list[str]) -> str:
    stdin = "".join(f"abc123\trefs/tags/{tag}\n" for tag in tags)
    result = subprocess.run(
        [str(SCRIPT), name, version],
        input=stdin,
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout.strip()


class TestLatest:
    def test_should_move_for_the_newest_stable_past_newer_candidates(self) -> None:
        assert channel("latest", "0.60.1", TAGS) == "latest"

    def test_should_not_move_for_an_old_train_hotfix(self) -> None:
        assert channel("latest", "0.59.0", TAGS) == ""

    def test_should_not_move_on_a_rerun_after_a_newer_release(self) -> None:
        assert channel("latest", "0.60.0", TAGS) == ""

    def test_should_compare_versions_numerically(self) -> None:
        assert channel("latest", "0.10.0", ["core/v0.10.0", "core/v0.9.0"]) == "latest"


class TestNext:
    def test_should_move_for_the_newest_candidate(self) -> None:
        assert channel("next", "0.61.0-rc.2", TAGS) == "next"

    def test_should_not_move_on_a_rerun_after_a_newer_candidate(self) -> None:
        assert channel("next", "0.61.0-rc.1", TAGS) == ""

    def test_should_not_move_below_its_stable_version(self) -> None:
        assert channel("next", "0.61.0-rc.2", [*TAGS, "core/v0.61.0"]) == ""


def test_should_print_nothing_without_a_channel() -> None:
    assert channel("", "0.60.1", TAGS) == ""

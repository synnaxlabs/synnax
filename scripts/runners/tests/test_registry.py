#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from registry import RunnerRegistry, known_runners


class TestRunnerRegistry:
    """Tests for the persisted runner names."""

    def test_should_persist_names_across_instances(self, tmp_path: Path) -> None:
        path = tmp_path / "registry.json"
        RunnerRegistry(path).add({"a", "b"})
        RunnerRegistry(path).add({"c"})
        assert known_runners(path) == {"a", "b", "c"}
        assert RunnerRegistry(path).snapshot() == {"a", "b", "c"}

    def test_should_leave_no_partial_file(self, tmp_path: Path) -> None:
        path = tmp_path / "registry.json"
        RunnerRegistry(path).add({"a"})
        assert [p.name for p in tmp_path.iterdir()] == ["registry.json"]

    def test_should_start_empty_without_a_file(self, tmp_path: Path) -> None:
        assert known_runners(tmp_path / "registry.json") == set()

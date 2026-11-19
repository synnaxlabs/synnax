#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from pathlib import Path

from registry import RunnerRegistry, known_runners


class TestRunnerRegistry:
    """Tests for the persisted runners and their labels."""

    def test_should_persist_names_across_instances(self, tmp_path: Path) -> None:
        path = tmp_path / "registry.json"
        RunnerRegistry(path).add({"a": set(), "b": set()})
        RunnerRegistry(path).add({"c": set()})
        assert known_runners(path) == {"a", "b", "c"}
        assert RunnerRegistry(path).snapshot() == {"a", "b", "c"}

    def test_should_keep_the_labels_of_a_runner_github_no_longer_lists(
        self, tmp_path: Path
    ) -> None:
        path = tmp_path / "registry.json"
        RunnerRegistry(path).add({"a": {"pool"}, "b": {"old"}})
        RunnerRegistry(path).add({"b": {"new"}})
        assert RunnerRegistry(path).labels() == {"a": {"pool"}, "b": {"new"}}

    def test_should_drop_the_old_name_of_a_renamed_instance(
        self, tmp_path: Path
    ) -> None:
        path = tmp_path / "registry.json"
        RunnerRegistry(path).add({"old-i-0aaa": {"old"}, "b-i-0bbb": {"pool"}})
        RunnerRegistry(path).add({"new-i-0aaa": {"new"}})
        assert RunnerRegistry(path).labels() == {
            "new-i-0aaa": {"new"},
            "b-i-0bbb": {"pool"},
        }

    def test_should_read_a_file_of_names(self, tmp_path: Path) -> None:
        path = tmp_path / "registry.json"
        path.write_text(json.dumps(["a", "b"]))
        assert RunnerRegistry(path).labels() == {"a": set(), "b": set()}

    def test_should_leave_no_partial_file(self, tmp_path: Path) -> None:
        path = tmp_path / "registry.json"
        RunnerRegistry(path).add({"a": set()})
        assert [p.name for p in tmp_path.iterdir()] == ["registry.json"]

    def test_should_start_empty_without_a_file(self, tmp_path: Path) -> None:
        assert known_runners(tmp_path / "registry.json") == set()

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import pytest

from go_affected import affected

ROOT = Path("/repo")
MODULE = ROOT / "core"
# go list -deps lists every dependency, not only the direct ones.
PACKAGES = [
    ("x/a", "/repo/x/a", []),
    ("core", "/repo/core", ["core/b", "x/a"]),
    ("core/b", "/repo/core/b", ["x/a"]),
    ("core/b [core/b.test]", "/repo/core/b", ["x/a", "core/t"]),
    ("core/b.test", "", ["core/b [core/b.test]"]),
    ("core/c", "/repo/core/c", []),
    ("core/c [core/c.test]", "/repo/core/c", ["core/b", "x/a"]),
    ("core/t", "/repo/core/t", []),
]


def run(*changed: str) -> list[str]:
    return affected([ROOT / c for c in changed], MODULE, PACKAGES)


class TestAffected:
    """Tests finding the packages a change reaches."""

    def test_should_reach_the_importers_of_a_changed_package(self) -> None:
        assert run("core/b/b.go") == [".", "./b", "./c"]

    def test_should_reach_through_another_module(self) -> None:
        assert run("x/a/a.go") == [".", "./b", "./c"]

    def test_should_reach_a_package_whose_tests_import_the_change(self) -> None:
        assert run("core/t/t.go") == ["./b", "./t"]

    def test_should_reach_only_the_package_of_a_test_file(self) -> None:
        assert run("core/b/b_test.go") == ["./b"]

    @pytest.mark.parametrize("path", ["docs/a.go", "core/gone/a.go"])
    def test_should_reach_nothing_for_a_file_outside_every_package(
        self, path: str
    ) -> None:
        assert run(path) == []

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

import go_affected

# x and base replace each other; app needs both; tool sits in a sibling of lib.
GO_MODS = {
    "x/go": "module x\n\nreplace base => ../../base\n",
    "base": "module base\n\nreplace x => ../x/go\n",
    "lib/go": "module lib\n\nreplace (\n\tx => ../../x/go\n\tother v1 => other v2\n)\n",
    "lib/tool": "module tool\n\nreplace lib => ../go\n",
    "app": "module app\n\nreplace (\n\tlib => ../lib/go\n\tx => ../x/go\n)\n",
    "leaf": "module leaf\n",
}
MODULES = list(GO_MODS)


@pytest.fixture
def root(tmp_path: Path) -> Path:
    for module, text in GO_MODS.items():
        (tmp_path / module).mkdir(parents=True)
        (tmp_path / module / "go.mod").write_text(text)
    return tmp_path


@pytest.mark.parametrize(
    ("changed", "expected"),
    [
        pytest.param([], [], id="nothing"),
        pytest.param([".github/workflows/ci.yaml"], [], id="no module"),
        pytest.param(["leaf/a.go"], ["leaf"], id="no dependents"),
        pytest.param(["app/main.go"], ["app"], id="a module only it uses"),
        pytest.param(
            ["lib/go/lib.go"], ["lib/go", "lib/tool", "app"], id="a sibling path"
        ),
        pytest.param(
            ["base/base.go"],
            ["x/go", "base", "lib/go", "lib/tool", "app"],
            id="a replace cycle",
        ),
        pytest.param(
            ["leaf/a.go", "lib/tool/b.go"], ["lib/tool", "leaf"], id="two modules"
        ),
        pytest.param(["lib/gopher.go"], [], id="a path that only shares a prefix"),
    ],
)
def test_affected(root: Path, changed: list[str], expected: list[str]) -> None:
    assert go_affected.affected(root, MODULES, changed) == expected


def test_affected_ignores_modules_not_given(root: Path) -> None:
    assert go_affected.affected(root, ["x/go", "leaf"], ["base/base.go"]) == []

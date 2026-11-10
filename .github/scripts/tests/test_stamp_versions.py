#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import shutil
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "stamp_versions.sh"
PACKAGES = ["alamos", "x", "freighter", "client"]


class Tree:
    """A scratch repository root with every manifest the script writes."""

    def __init__(self, root: Path) -> None:
        self.root = root
        script = root / ".github" / "scripts" / "stamp_versions.sh"
        script.parent.mkdir(parents=True)
        shutil.copy(SCRIPT, script)
        self.script = script
        for package in PACKAGES:
            ts = root / package / "ts"
            ts.mkdir(parents=True)
            (ts / "package.json").write_text(
                json.dumps({"name": package, "version": "0.61.0"}, indent=2) + "\n"
            )
            py = root / package / "py"
            py.mkdir(parents=True)
            (py / "pyproject.toml").write_text(
                f'[project]\nname = "{package}"\nversion = "0.61.0"\n'
            )

    def stamp(self, version: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", str(self.script), version], capture_output=True, text=True
        )


@pytest.fixture
def tree(tmp_path: Path) -> Tree:
    return Tree(tmp_path)


class TestStampVersions:
    """Tests for writing a release version into the package manifests."""

    @pytest.mark.parametrize(
        "version", ["0.62.3", "0.62.0-rc.2", "0.0.0-dev.1.abc1234"]
    )
    def test_should_write_every_manifest(self, tree: Tree, version: str) -> None:
        result = tree.stamp(version)
        assert result.returncode == 0, result.stderr
        for package in PACKAGES:
            pkg = json.loads((tree.root / package / "ts" / "package.json").read_text())
            assert pkg == {"name": package, "version": version}
            pyproject = (tree.root / package / "py" / "pyproject.toml").read_text()
            assert f'version = "{version}"\n' in pyproject

    @pytest.mark.parametrize("version", ["1.2", "v0.62.0", "0.62.0 ", ""])
    def test_should_reject_a_malformed_version(self, tree: Tree, version: str) -> None:
        result = tree.stamp(version)
        assert result.returncode != 0
        pkg = json.loads((tree.root / "client" / "ts" / "package.json").read_text())
        assert pkg["version"] == "0.61.0"

    def test_should_fail_without_a_version_line(self, tree: Tree) -> None:
        (tree.root / "x" / "py" / "pyproject.toml").write_text(
            '[project]\nname = "x"\n'
        )
        result = tree.stamp("0.62.0")
        assert result.returncode != 0
        assert "no version line" in result.stderr

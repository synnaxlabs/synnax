#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from check_exports_py import Finding, Package, check

PACKAGES = (
    Package(root="lib", checked=("widget",)),
    Package(root="app", checked=("tool",)),
)


def write(root: Path, files: dict[str, str]) -> None:
    for name, source in files.items():
        path = root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(source)
    for pkg in PACKAGES:
        toml = root / pkg.root / "pyproject.toml"
        if not toml.exists():
            toml.parent.mkdir(parents=True, exist_ok=True)
            toml.write_text("[project]\nname = 'p'\n")


def names(root: Path) -> list[str]:
    return [f.name for f in check(root, packages=PACKAGES)]


class TestCheck:
    """Tests which top-level names the check reports."""

    def test_should_report_a_name_only_its_own_module_uses(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LIMIT = 1\n\ndef twice():\n    return LIMIT\n",
            },
        )
        assert check(tmp_path, packages=PACKAGES) == [
            Finding(path="lib/widget/parts.py", line=1, name="LIMIT", tests_only=False),
            Finding(path="lib/widget/parts.py", line=3, name="twice", tests_only=False),
        ]

    def test_should_report_names_an_unpacking_assigns(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LEFT, (RIGHT, _MID) = 1, (2, 3)\n",
            },
        )
        assert names(tmp_path) == ["LEFT", "RIGHT"]

    def test_should_skip_private_names(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {"lib/widget/__init__.py": "", "lib/widget/parts.py": "_LIMIT = 1\n"},
        )
        assert names(tmp_path) == []

    def test_should_keep_a_name_another_module_imports(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LIMIT = 1\n",
                "lib/widget/valve.py": "from widget.parts import LIMIT\n",
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_a_name_read_through_a_module_alias(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LIMIT = 1\n",
                "app/tool/run.py": "import widget.parts as p\n\nX = p.LIMIT\n",
                "app/tool/__init__.py": "from tool.run import X\n",
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_a_name_a_relative_import_reads(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "from .parts import LIMIT\n",
                "lib/widget/parts.py": "LIMIT = 1\n",
            },
        )
        assert names(tmp_path) == []

    def test_should_report_a_name_only_its_own_tests_import(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LIMIT = 1\n",
                "lib/tests/test_parts.py": "from widget.parts import LIMIT\n",
            },
        )
        assert check(tmp_path, packages=PACKAGES) == [
            Finding(path="lib/widget/parts.py", line=1, name="LIMIT", tests_only=True)
        ]

    def test_should_report_a_name_only_another_packages_tests_import(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LIMIT = 1\n",
                "app/tests/test_run.py": "from widget.parts import LIMIT\n",
            },
        )
        assert check(tmp_path, packages=PACKAGES) == [
            Finding(path="lib/widget/parts.py", line=1, name="LIMIT", tests_only=True)
        ]

    def test_should_keep_a_test_utility_name_another_packages_tests_import(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/testutil/__init__.py": "",
                "lib/widget/testutil/fake.py": "def fake_valve(): ...\n",
                "app/tests/test_run.py": (
                    "from widget.testutil.fake import fake_valve\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_report_a_name_only_a_test_utility_imports(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LIMIT = 1\n",
                "lib/widget/testutil/__init__.py": "",
                "lib/widget/testutil/fake.py": (
                    "from widget.parts import LIMIT\n\nFAKE = LIMIT\n"
                ),
                "app/tests/test_run.py": "from widget.testutil.fake import FAKE\n",
            },
        )
        assert check(tmp_path, packages=PACKAGES) == [
            Finding(path="lib/widget/parts.py", line=1, name="LIMIT", tests_only=True)
        ]

    def test_should_keep_a_type_in_the_signature_of_a_name_in_use(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "from widget.parts import open_valve\n",
                "lib/widget/parts.py": (
                    "class Valve: ...\n\n"
                    "class Mode: ...\n\n"
                    "def open_valve(v: 'Valve') -> Mode: ...\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_the_value_of_an_annotated_type_alias_in_use(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "from widget.parts import Alias, Quoted\n",
                "lib/widget/parts.py": (
                    "from typing import TypeAlias\n\n"
                    "class Valve: ...\n\n"
                    "class Mode: ...\n\n"
                    "Alias: TypeAlias = Valve\n"
                    "Quoted: TypeAlias = 'Mode'\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_names_in_a_quoted_tuple_annotation(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "from widget.parts import pair\n",
                "lib/widget/parts.py": (
                    "class Valve: ...\n\n"
                    "class Mode: ...\n\n"
                    "def pair() -> 'tuple[Valve, Mode]': ...\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_names_read_through_one_alias_in_two_scopes(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "LIMIT = 1\n",
                "lib/widget/other.py": "SIZE = 1\n",
                "app/tool/__init__.py": "",
                "app/tool/run.py": (
                    "def a():\n"
                    "    import widget.parts as p\n"
                    "    return p.LIMIT\n\n"
                    "def b():\n"
                    "    import widget.other as p\n"
                    "    return p.SIZE\n"
                ),
            },
        )
        assert names(tmp_path) == ["a", "b"]

    def test_should_keep_a_name_a_quoted_annotation_reads_through_an_alias(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "class Valve: ...\n",
                "app/tool/__init__.py": "from tool.run import pair\n",
                "app/tool/run.py": (
                    "import widget.parts as p\n\ndef pair() -> 'p.Valve': ...\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_a_name_a_quoted_alias_or_base_reads(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "class Valve: ...\n\nclass Pump: ...\n",
                "app/tool/__init__.py": "from tool.run import Alias, Child\n",
                "app/tool/run.py": (
                    "from typing import Generic, TypeAlias\n"
                    "import widget.parts as p\n\n"
                    "Alias: TypeAlias = 'p.Valve'\n\n"
                    "class Child(Generic['p.Pump']): ...\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_report_a_name_only_a_plain_string_mentions(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "",
                "lib/widget/parts.py": "class Valve: ...\n",
                "app/tool/__init__.py": "from tool.run import LABEL\n",
                "app/tool/run.py": "import widget.parts as p\n\nLABEL = 'p.Valve'\n",
            },
        )
        assert names(tmp_path) == ["Valve"]

    def test_should_keep_a_type_a_special_method_returns(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "from widget.parts import Rack\n",
                "lib/widget/parts.py": (
                    "from collections.abc import Iterator\n\n"
                    "class Item: ...\n\n"
                    "class Rack:\n"
                    "    def __iter__(self) -> Iterator[Item]: ...\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_a_type_parameter_bound_of_a_name_in_use(
        self, tmp_path: Path
    ) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "from widget.parts import identity\n",
                "lib/widget/parts.py": (
                    "class Bound: ...\n\ndef identity[T: Bound](value: T) -> T: ...\n"
                ),
            },
        )
        assert names(tmp_path) == []

    def test_should_keep_an_entry_point(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {
                "app/pyproject.toml": "[project.scripts]\nrun = 'tool.run:main'\n",
                "app/tool/__init__.py": "",
                "app/tool/run.py": "def main(): ...\n",
            },
        )
        assert names(tmp_path) == []

    def test_should_skip_generated_files(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {"lib/widget/__init__.py": "", "lib/widget/types_gen.py": "LIMIT = 1\n"},
        )
        assert names(tmp_path) == []

    def test_should_keep_every_name_a_star_import_reads(self, tmp_path: Path) -> None:
        write(
            tmp_path,
            {
                "lib/widget/__init__.py": "from widget.parts import *\n",
                "lib/widget/parts.py": "LIMIT = 1\n",
            },
        )
        assert names(tmp_path) == []

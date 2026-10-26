#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from surface_py import report

SOURCE = '''
"""A widget package."""

from widget.parts import Valve

__all__ = ["Valve", "open_valve", "LIMIT"]

LIMIT: int = 10


def open_valve(valve: Valve, *, force: bool = False) -> None:
    """Opens the valve."""


def _helper() -> None: ...
'''

PARTS = """
class Valve:
    name: str = ""

    def __init__(self, name: str) -> None:
        self._secret = 1

    @property
    def open(self) -> bool:
        return True
"""


def write(root: Path, source: str = SOURCE) -> str:
    pkg = root / "widget"
    pkg.mkdir(exist_ok=True)
    (pkg / "__init__.py").write_text(source)
    (pkg / "parts.py").write_text(PARTS)
    return report(str(root), "widget")


class TestReport:
    def test_lists_public_objects_with_signatures(self, tmp_path: Path) -> None:
        assert write(tmp_path) == (
            "widget module\n"
            "widget.LIMIT: int = 10 [module-attribute]\n"
            "widget.Valve = widget.parts.Valve\n"
            "widget.open_valve(valve: Valve, *, force: bool = False) -> None\n"
            "widget.parts module\n"
            "widget.parts.Valve class()\n"
            "widget.parts.Valve.__init__(self, name: str) -> None\n"
            "widget.parts.Valve.name: str = '' [class-attribute, instance-attribute]\n"
            "widget.parts.Valve.open: bool [property]\n"
        )

    def test_ignores_docs_and_private_objects(self, tmp_path: Path) -> None:
        before = write(tmp_path)
        after = write(
            tmp_path,
            SOURCE.replace("Opens the valve.", "Opens it.").replace(
                "_helper", "_other"
            ),
        )
        assert after == before

    def test_changes_when_a_signature_changes(self, tmp_path: Path) -> None:
        before = write(tmp_path)
        after = write(tmp_path, SOURCE.replace("force: bool", "force: int"))
        assert after != before

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

import pytest

from spec_only import changed, is_spec, main, spec_only


class TestIsSpec:
    """Tests matching one path."""

    @pytest.mark.parametrize(
        "path",
        [
            "core/pkg/channel/channel_test.go",
            "driver/modbus/device_test.cpp",
            "console/src/range/slice.spec.ts",
            "pluto/src/button/Button.spec.tsx",
            "client/py/tests/test_channel.py",
        ],
    )
    def test_should_match_a_unit_test(self, path: str) -> None:
        assert is_spec(path)

    @pytest.mark.parametrize(
        "path",
        [
            "integration/framework/tests/test_watchdog.py",
            "integration/tests/console/label/label_lifecycle.py",
            "core/pkg/channel/channel.go",
            "client/py/synnax/testing.py",
            "client/py/tests/conftest.py",
            "console/src/range/slice.ts",
            "test_data.json",
        ],
    )
    def test_should_not_match_other_files(self, path: str) -> None:
        assert not is_spec(path)


class TestSpecOnly:
    """Tests deciding for a whole diff."""

    def test_should_be_true_for_only_unit_tests(self) -> None:
        assert spec_only(["core/a_test.go", "console/src/a.spec.ts"])

    def test_should_be_false_for_a_mixed_diff(self) -> None:
        assert not spec_only(["core/a_test.go", "core/a.go"])

    def test_should_be_false_for_an_empty_diff(self) -> None:
        assert not spec_only([])


class TestChanged:
    """Tests listing the changed files of a real repository."""

    def test_should_list_both_names_of_a_rename(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        def git(*args: str) -> str:
            return subprocess.run(
                ["git", "-c", "user.name=t", "-c", "user.email=t@t", *args],
                cwd=tmp_path,
                capture_output=True,
                text=True,
                check=True,
            ).stdout.strip()

        git("init", "-q")
        (tmp_path / "a.go").write_text("package a\n")
        git("add", ".")
        git("commit", "-q", "-m", "base")
        base = git("rev-parse", "HEAD")
        git("mv", "a.go", "a_test.go")
        git("commit", "-q", "-m", "rename")
        monkeypatch.chdir(tmp_path)
        paths = changed(base)
        assert sorted(paths) == ["a.go", "a_test.go"]
        assert not spec_only(paths)


class TestMain:
    """Tests the command line."""

    def test_should_print_false_without_a_base(
        self, capsys: pytest.CaptureFixture[str]
    ) -> None:
        main(["spec_only.py", ""])
        assert capsys.readouterr().out == "false\n"

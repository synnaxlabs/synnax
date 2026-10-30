#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import subprocess
from pathlib import Path

import pytest

import check_exports_ts
from check_exports_ts import Finding, check, knip, module_dir, users

TRACE = """\
app/src/kv/kv.ts:MemoryKV
├── app/src/kv/external.ts:reExportStar[MemoryKV]
│   └── app/src/kv/index.ts:reExportNS[KV.MemoryKV]
│       ├── app/src/kv/kv.spec.ts:import[KV] ⎆ ✓
│       │     refs: [KV.MemoryKV, KV.Lens]
│       └── app/src/store/store.ts:import[KV] ⎆ ✓
│             refs: [KV.Lens]
└── app/src/kv/direct.spec.ts:import[MemoryKV] ✓
app/src/other/kv.ts:MemoryKV
└── app/src/other/other.spec.ts:import[Other] ⎆ ✓
      refs: [Other.MemoryKVs]
"""


class TestUsers:
    """Tests reading a Knip trace."""

    def test_should_keep_files_that_reference_the_export(self) -> None:
        assert users(TRACE)[("app/src/kv/kv.ts", "MemoryKV")] == {
            "app/src/kv/kv.spec.ts",
            "app/src/kv/direct.spec.ts",
        }

    def test_should_skip_files_that_reference_other_members(self) -> None:
        assert users(TRACE)[("app/src/other/kv.ts", "MemoryKV")] == set()


class TestKnip:
    """Tests running Knip."""

    def test_should_raise_with_knip_errors_when_knip_fails(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        def run(*_: object, **__: object) -> subprocess.CompletedProcess[str]:
            return subprocess.CompletedProcess([], 2, "", "config not found")

        monkeypatch.setattr(subprocess, "run", run)
        with pytest.raises(RuntimeError, match="config not found"):
            knip(tmp_path, "--trace-export", "MemoryKV")

    def test_should_return_output_when_knip_reports_issues(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        def run(*_: object, **__: object) -> subprocess.CompletedProcess[str]:
            return subprocess.CompletedProcess([], 1, "issues", "")

        monkeypatch.setattr(subprocess, "run", run)
        assert knip(tmp_path, "--production") == "issues"


class TestModuleDir:
    """Tests finding the module of a file."""

    def test_should_return_the_nearest_directory_with_an_index(
        self, tmp_path: Path
    ) -> None:
        (tmp_path / "app/src/kv/deep").mkdir(parents=True)
        (tmp_path / "app/src/kv/index.ts").touch()
        assert (
            module_dir(tmp_path, file="app/src/kv/deep/a.ts") == tmp_path / "app/src/kv"
        )

    def test_should_stop_at_the_source_root(self, tmp_path: Path) -> None:
        (tmp_path / "app/src/kv").mkdir(parents=True)
        (tmp_path / "app/index.ts").touch()
        assert module_dir(tmp_path, file="app/src/kv/a.ts") == tmp_path / "app/src/kv"


def report(*names: tuple[str, str]) -> str:
    issues = [
        {"file": file, "exports": [{"name": name, "line": 1}], "types": []}
        for file, name in names
    ]
    return "Synnax - Development mode\n" + json.dumps({"issues": issues}) + "\n"


class TestCheck:
    """Tests which Knip findings the check keeps."""

    @pytest.fixture
    def repo(self, tmp_path: Path) -> Path:
        for d in ("app/src/kv", "app/src/store", "app/src/other"):
            (tmp_path / d).mkdir(parents=True)
            (tmp_path / d / "index.ts").touch()
        return tmp_path

    def test_should_report_an_export_only_its_module_specs_use(
        self, repo: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        trace = TRACE.replace("│       └── app/src/store/store.ts:import[KV] ⎆ ✓\n", "")
        outputs = {"--production": report(("app/src/kv/kv.ts", "MemoryKV"))}
        monkeypatch.setattr(
            check_exports_ts,
            "knip",
            lambda _, *args: outputs.get(args[0], trace),
        )
        assert check(repo) == [
            Finding(path="app/src/kv/kv.ts", line=1, name="MemoryKV")
        ]

    @pytest.mark.parametrize(
        ("export", "user", "flagged"),
        [
            pytest.param(
                "app/src/kv/kv.ts", "app/src/store/store.ts", False, id="production"
            ),
            pytest.param(
                "app/src/kv/kv.ts", "app/src/store/store.spec.ts", True, id="spec"
            ),
            pytest.param(
                "app/src/kv/kv.ts", "app/src/testutil/store.ts", True, id="testutil"
            ),
            pytest.param(
                "app/src/kv/kv.ts", "app/src/kv/child/child.ts", False, id="child"
            ),
            pytest.param(
                "app/src/testutil/kv.ts",
                "app/src/store/store.spec.ts",
                False,
                id="testutil-export",
            ),
            pytest.param(
                "app/src/testutil/kv.ts",
                "app/src/testutil/kv.spec.ts",
                True,
                id="testutil-own-spec",
            ),
        ],
    )
    def test_should_judge_a_use_from_another_module(
        self,
        repo: Path,
        monkeypatch: pytest.MonkeyPatch,
        export: str,
        user: str,
        flagged: bool,
    ) -> None:
        for d in ("app/src/kv/child", "app/src/testutil"):
            (repo / d).mkdir(exist_ok=True)
            (repo / d / "index.ts").touch()
        trace = f"{export}:MemoryKV\n└── {user}:import[MemoryKV] ✓\n"
        outputs = {"--production": report((export, "MemoryKV"))}
        monkeypatch.setattr(
            check_exports_ts,
            "knip",
            lambda _, *args: outputs.get(args[0], trace),
        )
        expected = [Finding(path=export, line=1, name="MemoryKV")] if flagged else []
        assert check(repo) == expected

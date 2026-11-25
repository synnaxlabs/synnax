#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import subprocess
import sys
from dataclasses import replace
from pathlib import Path

import pytest

import compare_benchmarks as cb
from test_compare_benchmarks import GBENCH, Repo

GO = "/usr/local/go/bin"


@pytest.fixture(autouse=True)
def go_on_path(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PATH", f"{GO}{os.pathsep}{os.environ['PATH']}")


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    return Repo(tmp_path)


def real_benchstat(base: str, head: str, tmp: Path) -> str:
    (tmp / "base.txt").write_text(base)
    (tmp / "head.txt").write_text(head)
    return subprocess.run(
        [*cb.BENCHSTAT, "-alpha", "0.05", "-format", "csv",
         f"base={tmp / 'base.txt'}", f"head={tmp / 'head.txt'}"],
        check=True, capture_output=True, text=True,
    ).stdout


def test_judge_reads_a_group_new_on_head_as_new(tmp_path: Path) -> None:
    # What measure writes for `--suite io --only '^cesium/commit$'` when base lacks
    # BenchmarkCommit: the base binary matches nothing and prints only PASS.
    header = "goos: linux\ngoarch: arm64\npkg: github.com/synnaxlabs/cesium\n"
    base = "pkg: cesium/commit\nPASS\n" * 10
    head = "".join(
        f"pkg: cesium/commit\n{header}"
        f"BenchmarkCommit/W1-32 100 {10000 + i} ns/op 10 B/op 3 allocs/op\nPASS\n"
        for i in range(10)
    )
    findings = cb.judge(real_benchstat(base, head, tmp_path))
    assert [f.kind for f in findings] == ["new"]


def test_report_passes_a_group_with_no_benchmark_on_base(
    repo: Repo, capsys: pytest.CaptureFixture[str]
) -> None:
    repo.commit("0", "empty")
    cfg = replace(
        repo.config(repo.group("x")), base="empty", benchstat=cb.BENCHSTAT
    )
    cb.measure(cfg)
    passed = cb.report(cfg)
    out = capsys.readouterr().out
    assert "FAIL" not in out
    assert passed


def test_measure_removes_the_base_worktree_when_the_head_ref_is_bad(
    repo: Repo,
) -> None:
    cfg = replace(repo.config(repo.group("x")), head="no-such-ref")
    with pytest.raises(subprocess.CalledProcessError):
        cb.measure(cfg)
    assert len(repo.worktrees()) == 1


def report_group(script: str) -> cb.Group:
    """Returns a gbench group whose run executes script with the report path."""
    run = cb.Step(".", (sys.executable, "-c", script, "{json}"))
    return cb.Group("gb", "io", (), run, False, "gbench")


def test_measure_reads_no_stale_report_on_head(repo: Repo) -> None:
    # Base writes its report. Head exits 0 and writes none. Head must not get the
    # numbers of base.
    group = report_group(
        "import sys, os\n"
        "if os.path.exists('report.json'):\n"
        "    open(sys.argv[1], 'w').write(open('report.json').read())\n"
    )
    (repo.root / "report.json").write_text(json.dumps(GBENCH))
    repo.commit("300", "report")
    repo.git("rm", "-q", "report.json")
    repo.git("commit", "-q", "-m", "no report")
    repo.git("tag", "noreport")
    cfg = replace(repo.config(group), base="report", head="noreport", rounds=1)
    try:
        cb.measure(cfg)
    except Exception:
        return
    assert "BenchmarkEncode" not in cfg.results("head", False).read_text()


def test_measure_runs_with_a_relative_out(
    repo: Repo, monkeypatch: pytest.MonkeyPatch
) -> None:
    # The documented run is from the repo root, with ROOT as the repo.
    monkeypatch.chdir(repo.root)
    group = report_group(
        f"import sys; open(sys.argv[1], 'w').write({json.dumps(GBENCH)!r})"
    )
    cfg = replace(repo.config(group), out=Path("rel"), rounds=1)
    cfg.out.mkdir()
    cb.measure(cfg)
    want = "pkg: gb\n" + cb.from_gbench(GBENCH)
    assert cfg.results("head", False).read_text() == want

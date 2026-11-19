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
import sys
from dataclasses import replace
from pathlib import Path

import pytest

import compare_benchmarks as cb

PACKAGE = "example.com/pkg"


def table(unit: str, *rows: str, package: str = PACKAGE) -> str:
    """Returns one table in the CSV form that benchstat prints."""
    header = [f"pkg: {package}", ",base,,head,,,", f",{unit},CI,{unit},CI,vs base,P"]
    return "\n".join([*header, *rows, "geomean,1,,1,,+0.00%,", "", ""])


@pytest.mark.parametrize(
    ("unit", "row", "want"),
    [
        (
            "sec/op",
            "Write-8,1e-06,1%,1.06e-06,1%,+6.00%,p=0.000 n=10",
            [cb.Finding("slower", PACKAGE, "Write-8", "+6.00%")],
        ),
        ("sec/op", "Write-8,1e-06,9%,1.06e-06,9%,~,p=0.300 n=10", []),
        ("sec/op", "Write-8,1e-06,1%,1.04e-06,1%,+4.00%,p=0.000 n=10", []),
        ("sec/op", "Write-8,1e-06,1%,9e-07,1%,-10.00%,p=0.000 n=10", []),
        (
            "allocs/op",
            "Write-8,300,0%,301,0%,+0.33%,p=0.000 n=10",
            [cb.Finding("allocations", PACKAGE, "Write-8", "300 to 301 allocs/op")],
        ),
        (
            "allocs/op",
            "Write-8,0,0%,1,0%,?,p=0.000 n=10",
            [cb.Finding("allocations", PACKAGE, "Write-8", "0 to 1 allocs/op")],
        ),
        ("allocs/op", "Write-8,40,9%,42,9%,~,p=0.400 n=10", []),
        ("allocs/op", "Write-8,4,0%,3,0%,-25.00%,p=0.000 n=10", []),
        ("sec/op", "Gone-8,5e-09,0%", [cb.Finding("missing", PACKAGE, "Gone-8")]),
        ("sec/op", "New-8,,,5e-09,0%", [cb.Finding("new", PACKAGE, "New-8")]),
        ("allocs/op", "Gone-8,3,0%", []),
        ("B/op", "Write-8,512,0%,1024,0%,+100.00%,p=0.000 n=10", []),
        ("samples/sec", "Write-8,100,0%,200,0%,+100.00%,p=0.000 n=10", []),
    ],
)
def test_judge_applies_the_rule(unit: str, row: str, want: list[cb.Finding]) -> None:
    assert cb.judge(table(unit, row)) == want


def test_judge_names_the_package_of_each_table() -> None:
    slow = "Write-8,1e-06,1%,2e-06,1%,+100.00%,p=0.000 n=10"
    csv = (
        'goos: linux\n"cpu: A CPU, 2.5GHz"\n'
        + table("sec/op", slow, package="example.com/a")
        + table("allocs/op", "Write-8,1,0%,2,0%,+100.00%,p=0.000 n=10")
    )
    assert cb.judge(csv) == [
        cb.Finding("slower", "example.com/a", "Write-8", "+100.00%"),
        cb.Finding("allocations", PACKAGE, "Write-8", "1 to 2 allocs/op"),
    ]


def test_order_flips_on_odd_rounds() -> None:
    assert [cb.order(r) for r in range(3)] == [
        ("base", "head"),
        ("head", "base"),
        ("base", "head"),
    ]


GBENCH = {
    "context": {"num_cpus": 8},
    "benchmarks": [
        {
            "name": "BM_Decode/100/10",
            "run_type": "iteration",
            "iterations": 5,
            "real_time": 1.5,
            "cpu_time": 1.4,
            "time_unit": "us",
            "allocs/op": 2.0,
            "bytes/op": 64.0,
        },
        {
            "name": "BM_Decode/100/10_mean",
            "run_type": "aggregate",
            "iterations": 3,
            "real_time": 1.5,
            "time_unit": "us",
        },
        {
            "name": "BM_Encode",
            "run_type": "iteration",
            "iterations": 7,
            "real_time": 20.0,
            "time_unit": "ns",
        },
    ],
}
VITEST = {
    "testResults": [
        {
            "name": "/work/src/head/client/ts/src/framer/codec.bench.ts",
            "assertionResults": [
                {
                    "fullName": "decode one channel",
                    "benchmarks": [
                        {
                            "name": "decode one channel",
                            "tasks": [
                                {
                                    "name": "100ser x 10smp",
                                    "latency": {"mean": 0.0625, "samplesCount": 16},
                                }
                            ],
                        }
                    ],
                },
                {"fullName": "a skipped test"},
            ],
        }
    ]
}


def test_from_gbench_keeps_iteration_runs_in_nanoseconds() -> None:
    assert cb.from_gbench(GBENCH) == (
        "BenchmarkDecode/100/10 5 1500.0 ns/op 2.0 allocs/op\n"
        "BenchmarkEncode 7 20.0 ns/op\n"
    )


def test_from_vitest_names_each_task_by_file_and_test() -> None:
    assert cb.from_vitest(VITEST) == (
        "BenchmarkTS/codec/decode_one_channel/100ser_x_10smp 16 62500.0 ns/op\n"
    )


BENCH = """import os, sys
value = open(sys.argv[1]).read().strip()
if value != "0":
    print(f"BenchmarkX 1 {value} ns/op")
with open(sys.argv[2], "a") as log:
    log.write(f"{value} {os.environ['TMPDIR']}\\n")
"""
BUILD = "import shutil, sys; shutil.copy('value.txt', sys.argv[1])"
BENCHSTAT = """import sys
csv = "-format" in sys.argv
print(open(sys.argv[1]).read() if csv and "control" not in sys.argv[-1] else "table")
"""
SLOWER = table("sec/op", "X,1e-07,0%,2e-07,0%,+100.00%,p=0.000 n=3")


def lines(group: str, value: int) -> str:
    """Returns what one run of a group in Repo adds to a result file."""
    return f"pkg: {group}\nBenchmarkX 1 {value} ns/op\n"


class Repo:
    """A Git repository where base and head each print their own benchmark value."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.root = path / "repo"
        self.root.mkdir()
        self.log = path / "log.txt"
        self.io_dir = path / "io"
        self.io_dir.mkdir()
        self.csv = path / "benchstat.csv"
        self.csv.write_text("")
        (path / "benchstat.py").write_text(BENCHSTAT)
        self.git("init", "-q", "-b", "main")
        (self.root / "bench.py").write_text(BENCH)
        self.commit("100", "base")
        self.commit("200", "head")

    def git(self, *args: str) -> str:
        identity = ["-c", "user.name=test", "-c", "user.email=test@example.com"]
        return subprocess.run(
            ["git", *identity, *args],
            cwd=self.root,
            check=True,
            capture_output=True,
            text=True,
        ).stdout

    def commit(self, value: str, tag: str) -> None:
        (self.root / "value.txt").write_text(value)
        self.git("add", ".")
        self.git("commit", "-q", "-m", tag)
        self.git("tag", tag)

    def group(self, name: str, *, control: bool = False, code: int = 0) -> cb.Group:
        """Returns a group whose run exits with code after it prints its value."""
        build = cb.Step(".", (sys.executable, "-c", BUILD, "{bin}/built.txt"))
        bench = f"exec(open('bench.py').read()); sys.exit({code})"
        argv = (sys.executable, "-c", bench, "{bin}/built.txt", str(self.log))
        return cb.Group(name, "io", (build,), cb.Step(".", argv), control)

    def config(self, *groups: cb.Group) -> cb.Config:
        out = self.path / "out"
        out.mkdir()
        return cb.Config(
            base="base",
            head="head",
            out=out,
            rounds=3,
            suite="io",
            io_dir=self.io_dir,
            repo=self.root,
            groups=groups,
            benchstat=(sys.executable, str(self.path / "benchstat.py"), str(self.csv)),
            host_facts=((sys.executable, "-c", "print('disk {dir}')"),),
        )

    def worktrees(self) -> list[str]:
        return self.git("worktree", "list", "--porcelain").split("\n\n")[:-1]


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    return Repo(tmp_path)


def test_measure_runs_each_ref_once_per_round_in_flipped_order(repo: Repo) -> None:
    cfg = repo.config(repo.group("x"))
    cb.measure(cfg)
    assert cfg.results("base", False).read_text() == lines("x", 100) * 3
    assert cfg.results("head", False).read_text() == lines("x", 200) * 3
    values = ("100", "200", "200", "100", "100", "200")
    assert repo.log.read_text().splitlines() == [f"{v} {repo.io_dir}" for v in values]
    assert len(repo.worktrees()) == 1


def test_measure_keeps_control_results_apart(repo: Repo) -> None:
    cfg = repo.config(repo.group("x"), repo.group("floor", control=True))
    cb.measure(cfg)
    assert cfg.results("base", True).read_text() == lines("floor", 100) * 3
    assert cfg.results("base", False).read_text() == lines("x", 100) * 3


def test_measure_records_the_host(repo: Repo) -> None:
    cfg = repo.config(repo.group("x"))
    cb.measure(cfg)
    start, disk, end = (cfg.out / "host.txt").read_text().splitlines()
    assert start.startswith("load at start: (")
    assert disk == f"disk {repo.io_dir}"
    assert end.startswith("load at end: (")


def test_measure_accepts_a_group_with_no_benchmark_on_base(repo: Repo) -> None:
    repo.commit("0", "empty")
    cfg = replace(repo.config(repo.group("x")), base="empty")
    cb.measure(cfg)
    assert cfg.results("base", False).read_text() == "pkg: x\n" * 3
    assert cfg.results("head", False).read_text() == lines("x", 200) * 3


def test_measure_fails_on_a_group_with_no_benchmark_on_head(repo: Repo) -> None:
    repo.commit("0", "empty")
    cfg = replace(repo.config(repo.group("x")), head="empty")
    with pytest.raises(RuntimeError, match=r"^x ran no benchmark on head$"):
        cb.measure(cfg)
    assert len(repo.worktrees()) == 1


def test_measure_converts_the_report_of_a_run(repo: Repo) -> None:
    write = "import sys; open(sys.argv[1], 'w').write(open('report.json').read())"
    group = cb.Group(
        "gb",
        "io",
        (),
        cb.Step(".", (sys.executable, "-c", write, "{json}")),
        False,
        "gbench",
    )
    (repo.root / "report.json").write_text(json.dumps(GBENCH))
    repo.commit("300", "report")
    cfg = replace(repo.config(group), base="report", head="report", rounds=1)
    cb.measure(cfg)
    want = "pkg: gb\n" + cb.from_gbench(GBENCH)
    assert cfg.results("head", False).read_text() == want


def test_measure_fails_on_an_unknown_format(repo: Repo) -> None:
    group = replace(repo.group("x"), format="tap")
    with pytest.raises(KeyError, match="tap"):
        cb.measure(repo.config(group))
    assert len(repo.worktrees()) == 1


def test_measure_removes_the_worktrees_when_a_benchmark_fails(repo: Repo) -> None:
    cfg = repo.config(repo.group("x", code=3))
    with pytest.raises(subprocess.CalledProcessError) as error:
        cb.measure(cfg)
    assert error.value.returncode == 3
    assert len(repo.worktrees()) == 1


def test_report_fails_on_a_finding(
    repo: Repo, capsys: pytest.CaptureFixture[str]
) -> None:
    cfg = repo.config(repo.group("x"))
    cb.measure(cfg)
    repo.csv.write_text(SLOWER)
    assert not cb.report(cfg)
    assert f"FAIL slower {PACKAGE} X +100.00%" in capsys.readouterr().out
    assert (cfg.out / "report.txt").read_text() == "table\n"


def test_report_fails_when_benchstat_fails(repo: Repo) -> None:
    cfg = repo.config(repo.group("x"))
    cb.measure(cfg)
    (repo.path / "benchstat.py").write_text("import sys; sys.exit('no such file')")
    with pytest.raises(RuntimeError, match=r"^benchstat failed: no such file$"):
        cb.report(cfg)


def test_report_passes_with_no_finding(repo: Repo) -> None:
    cfg = repo.config(repo.group("x"))
    cb.measure(cfg)
    assert cb.report(cfg)


def test_report_passes_on_a_new_benchmark(
    repo: Repo, capsys: pytest.CaptureFixture[str]
) -> None:
    cfg = repo.config(repo.group("x"))
    cb.measure(cfg)
    repo.csv.write_text(table("sec/op", "New-8,,,5e-09,0%"))
    assert cb.report(cfg)
    assert f"new {PACKAGE} New-8" in capsys.readouterr().out


def test_report_never_fails_on_a_control(
    repo: Repo, capsys: pytest.CaptureFixture[str]
) -> None:
    cfg = repo.config(repo.group("floor", control=True))
    cb.measure(cfg)
    (repo.path / "benchstat.py").write_text(BENCHSTAT.replace("not in", "in"))
    repo.csv.write_text(SLOWER)
    assert cb.report(cfg)
    assert f"noisy control {PACKAGE} X +100.00%" in capsys.readouterr().out


def test_parse_selects_the_groups_of_the_suite(tmp_path: Path) -> None:
    cfg = cb.parse(["--base", "a", "--head", "b", "--out", str(tmp_path / "out")])
    assert (cfg.base, cfg.head, cfg.rounds) == ("a", "b", 10)
    assert cfg.selected()
    assert {g.suite for g in cfg.selected()} == {"cpu"}
    only = cb.parse(["--base", "a", "--head", "b", "--only", "^cesium/"])
    assert [g.name for g in only.selected()] == ["cesium/write", "cesium/read"]


@pytest.mark.parametrize(
    ("args", "message"),
    [
        (["--suite", "io"], "the io suite needs --io-dir"),
        (["--only", "nothing"], "no cpu group matches 'nothing'"),
    ],
)
def test_parse_rejects(
    args: list[str], message: str, capsys: pytest.CaptureFixture[str]
) -> None:
    with pytest.raises(SystemExit) as error:
        cb.parse(["--base", "a", "--head", "b", *args])
    assert error.value.code == 2
    assert message in capsys.readouterr().err


def test_parse_refuses_an_existing_out_directory(tmp_path: Path) -> None:
    with pytest.raises(FileExistsError):
        cb.parse(["--base", "a", "--head", "b", "--out", str(tmp_path)])


def test_vitest_writes_a_json_report() -> None:
    group = cb.vitest("ts/codec", "src/framer/codec.bench.ts")
    assert (group.suite, group.format, group.run.directory) == (
        "cpu",
        "vitest",
        "client/ts",
    )
    assert group.run.argv[-3:] == (
        "--reporter=json",
        "--outputFile={json}",
        "src/framer/codec.bench.ts",
    )


def test_gbench_runs_the_binary_that_bazel_builds() -> None:
    group = cb.gbench("cpp/codec", "client/cpp/framer", "codec_bench")
    assert group.build == (
        cb.Step(
            ".", ("bazel", "build", "-c", "opt", "//client/cpp/framer:codec_bench")
        ),
    )
    assert group.format == "gbench"
    assert group.run == cb.Step(
        ".",
        (
            "bazel-bin/client/cpp/framer/codec_bench",
            "--benchmark_out={json}",
            "--benchmark_out_format=json",
        ),
    )


def test_go_builds_one_binary_per_package() -> None:
    group = cb.go("cesium/write", "cpu", "cesium", "^BenchmarkWrite$")
    assert group.build == (
        cb.Step("cesium", ("go", "test", "-c", "-o", "{bin}/cesium.test", ".")),
    )
    assert group.run.argv[:3] == (
        "{bin}/cesium.test",
        "-test.run=^$",
        "-test.bench=^BenchmarkWrite$",
    )

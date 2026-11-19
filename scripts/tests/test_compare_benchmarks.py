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


class TestJudge:
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
            ("allocs/op", "Write-8,0,?,1,?,~,p=0.070 n=10", []),
            ("allocs/op", "Write-8,40,9%,42,9%,~,p=0.400 n=10", []),
            ("allocs/op", "Write-8,4,0%,3,0%,-25.00%,p=0.000 n=10", []),
            ("sec/op", "Gone-8,5e-09,0%", [cb.Finding("missing", PACKAGE, "Gone-8")]),
            ("sec/op", "New-8,,,5e-09,0%", [cb.Finding("new", PACKAGE, "New-8")]),
            ("allocs/op", "Gone-8,3,0%", []),
            ("B/op", "Write-8,512,0%,1024,0%,+100.00%,p=0.000 n=10", []),
            ("samples/sec", "Write-8,100,0%,200,0%,+100.00%,p=0.000 n=10", []),
        ],
    )
    def test_should_apply_the_rule(
        self, unit: str, row: str, want: list[cb.Finding]
    ) -> None:
        assert cb.judge(table(unit, row)) == want

    def test_should_read_a_table_that_only_one_side_has(self) -> None:
        # benchstat prints one column when the other side has no result for a package.
        rows = ",sec/op,CI\nCommit-8,1e-05,0%\ngeomean,1e-05,\n\n"
        assert cb.judge(f"pkg: {PACKAGE}\n,head,\n{rows}") == [
            cb.Finding("new", PACKAGE, "Commit-8")
        ]
        assert cb.judge(f"pkg: {PACKAGE}\n,base,\n{rows}") == [
            cb.Finding("missing", PACKAGE, "Commit-8")
        ]

    def test_should_name_the_package_of_each_table(self) -> None:
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


class TestFormats:
    def test_should_keep_gbench_iteration_runs_in_nanoseconds(self) -> None:
        assert cb.from_gbench(GBENCH) == (
            "BenchmarkDecode/100/10 5 1500.0 ns/op 2.0 allocs/op\n"
            "BenchmarkEncode 7 20.0 ns/op\n"
        )

    def test_should_fail_on_a_gbench_benchmark_that_reports_an_error(self) -> None:
        skipped = {
            "name": "BM_Decode/100/10",
            "run_type": "iteration",
            "error_occurred": True,
            "error_message": "no such channel",
            "iterations": 0,
            "real_time": 0.0,
            "time_unit": "ns",
        }
        with pytest.raises(RuntimeError, match=r"^BM_Decode/100/10: no such channel$"):
            cb.from_gbench({"benchmarks": [skipped]})

    def test_should_name_each_vitest_task_by_file_and_test(self) -> None:
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
BUILD = """import shutil, sys
shutil.copy("value.txt", sys.argv[1])
with open(sys.argv[2], "a") as log:
    log.write(open("value.txt").read() + "\\n")
"""
BENCHSTAT = """import sys
gated, control = sys.argv[1:3]
if "-format" in sys.argv:
    print(open(control if "control" in sys.argv[-1] else gated).read())
else:
    print("table")
"""
# Writes the report that the ref holds, but only on the ref with value 100.
REPORT = """import sys
if open("value.txt").read() == "100":
    open(sys.argv[1], "w").write(open("report.json").read())
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
        self.builds = path / "builds.txt"
        self.io_dir = path / "io"
        self.io_dir.mkdir()
        self.csv = path / "benchstat.csv"
        self.csv.write_text("")
        self.control_csv = path / "control.csv"
        self.control_csv.write_text("")
        (path / "benchstat.py").write_text(BENCHSTAT)
        self.git("init", "-q", "-b", "main")
        (self.root / "bench.py").write_text(BENCH)
        (self.root / "report.json").write_text(json.dumps(GBENCH))
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
        build = (sys.executable, "-c", BUILD, "{bin}/built.txt", str(self.builds))
        bench = f"exec(open('bench.py').read()); sys.exit({code})"
        argv = (sys.executable, "-c", bench, "{bin}/built.txt", str(self.log))
        return cb.Group(name, "io", (cb.Step(".", build),), cb.Step(".", argv), control)

    def reporter(self) -> cb.Group:
        """Returns a group whose run writes a Google Benchmark report on base only."""
        argv = (sys.executable, "-c", REPORT, "{json}")
        return cb.Group("gb", "io", (), cb.Step(".", argv), format="gbench")

    def config(self, *groups: cb.Group) -> cb.Config:
        out = self.path / "out"
        out.mkdir()
        benchstat = (sys.executable, str(self.path / "benchstat.py"))
        return cb.Config(
            base="base",
            head="head",
            out=out,
            rounds=3,
            suite="io",
            io_dir=self.io_dir,
            repo=self.root,
            groups=groups,
            benchstat=(*benchstat, str(self.csv), str(self.control_csv)),
            host_facts=((sys.executable, "-c", "print('disk {dir}')"),),
        )

    def worktrees(self) -> list[str]:
        return self.git("worktree", "list", "--porcelain").split("\n\n")[:-1]


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    return Repo(tmp_path)


class TestMeasure:
    def test_should_flip_the_order_on_odd_rounds(self) -> None:
        assert [cb.order(r) for r in range(3)] == [
            ("base", "head"),
            ("head", "base"),
            ("base", "head"),
        ]

    def test_should_run_each_ref_once_per_round_in_flipped_order(
        self, repo: Repo
    ) -> None:
        cfg = repo.config(repo.group("x"))
        cb.measure(cfg)
        assert cfg.results("base", False).read_text() == lines("x", 100) * 3
        assert cfg.results("head", False).read_text() == lines("x", 200) * 3
        values = ("100", "200", "200", "100", "100", "200")
        assert repo.log.read_text().splitlines() == [
            f"{v} {repo.io_dir}" for v in values
        ]
        assert len(repo.worktrees()) == 1

    def test_should_build_each_ref_once_for_groups_that_share_a_step(
        self, repo: Repo
    ) -> None:
        cb.measure(repo.config(repo.group("x"), repo.group("y")))
        assert repo.builds.read_text().splitlines() == ["100", "200"]

    def test_should_accept_one_ref_as_both_sides(self, repo: Repo) -> None:
        cfg = replace(repo.config(repo.group("x")), head="base")
        cb.measure(cfg)
        assert cfg.results("base", False).read_text() == lines("x", 100) * 3
        assert cfg.results("head", False).read_text() == lines("x", 100) * 3

    def test_should_keep_control_results_apart(self, repo: Repo) -> None:
        cfg = repo.config(repo.group("x"), repo.group("floor", control=True))
        cb.measure(cfg)
        assert cfg.results("base", True).read_text() == lines("floor", 100) * 3
        assert cfg.results("base", False).read_text() == lines("x", 100) * 3

    def test_should_record_the_host(self, repo: Repo) -> None:
        cfg = repo.config(repo.group("x"))
        cb.measure(cfg)
        start, disk, end = (cfg.out / "host.txt").read_text().splitlines()
        assert start.startswith("load at start: (")
        assert disk == f"disk {repo.io_dir}"
        assert end.startswith("load at end: (")

    def test_should_accept_a_group_with_no_benchmark_on_base(self, repo: Repo) -> None:
        repo.commit("0", "empty")
        cfg = replace(repo.config(repo.group("x")), base="empty")
        cb.measure(cfg)
        assert cfg.results("base", False).read_text() == "pkg: x\n" * 3
        assert cfg.results("head", False).read_text() == lines("x", 200) * 3

    def test_should_fail_on_a_group_with_no_benchmark_on_head(self, repo: Repo) -> None:
        repo.commit("0", "empty")
        cfg = replace(repo.config(repo.group("x")), head="empty")
        with pytest.raises(RuntimeError, match=r"^x ran no benchmark on head$"):
            cb.measure(cfg)
        assert len(repo.worktrees()) == 1

    def test_should_convert_the_report_of_a_run(self, repo: Repo) -> None:
        cfg = replace(repo.config(repo.reporter()), head="base", rounds=1)
        cb.measure(cfg)
        want = "pkg: gb\n" + cb.from_gbench(GBENCH)
        assert cfg.results("head", False).read_text() == want

    def test_should_fail_on_a_run_that_writes_no_report(self, repo: Repo) -> None:
        cfg = replace(repo.config(repo.reporter()), rounds=1)
        with pytest.raises(FileNotFoundError, match=r"run\.json"):
            cb.measure(cfg)
        assert not cfg.results("head", False).exists()

    def test_should_run_with_a_relative_out_directory(
        self, repo: Repo, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.chdir(repo.path)
        cfg = replace(repo.config(repo.reporter()), out=Path("out"), head="base")
        cb.measure(cfg)
        want = ("pkg: gb\n" + cb.from_gbench(GBENCH)) * 3
        assert cfg.results("head", False).read_text() == want

    def test_should_fail_on_an_unknown_format(self, repo: Repo) -> None:
        group = replace(repo.group("x"), format="tap")
        with pytest.raises(KeyError, match="tap"):
            cb.measure(repo.config(group))
        assert len(repo.worktrees()) == 1

    def test_should_remove_the_worktrees_when_a_benchmark_fails(
        self, repo: Repo
    ) -> None:
        cfg = repo.config(repo.group("x", code=3))
        with pytest.raises(subprocess.CalledProcessError) as error:
            cb.measure(cfg)
        assert error.value.returncode == 3
        assert len(repo.worktrees()) == 1

    def test_should_remove_the_base_worktree_when_the_head_ref_is_bad(
        self, repo: Repo
    ) -> None:
        cfg = replace(repo.config(repo.group("x")), head="no-such-ref")
        with pytest.raises(subprocess.CalledProcessError):
            cb.measure(cfg)
        assert len(repo.worktrees()) == 1


class TestReport:
    def test_should_fail_on_a_finding(
        self, repo: Repo, capsys: pytest.CaptureFixture[str]
    ) -> None:
        cfg = repo.config(repo.group("x"))
        cb.measure(cfg)
        repo.csv.write_text(SLOWER)
        assert not cb.report(cfg)
        assert f"FAIL slower {PACKAGE} X +100.00%" in capsys.readouterr().out
        assert (cfg.out / "report.txt").read_text() == "table\n"

    def test_should_fail_when_benchstat_fails(self, repo: Repo) -> None:
        cfg = repo.config(repo.group("x"))
        cb.measure(cfg)
        (repo.path / "benchstat.py").write_text("import sys; sys.exit('no such file')")
        with pytest.raises(RuntimeError, match=r"^benchstat failed: no such file$"):
            cb.report(cfg)

    def test_should_pass_with_no_finding(self, repo: Repo) -> None:
        cfg = repo.config(repo.group("x"))
        cb.measure(cfg)
        assert cb.report(cfg)

    def test_should_pass_on_a_new_benchmark(
        self, repo: Repo, capsys: pytest.CaptureFixture[str]
    ) -> None:
        cfg = repo.config(repo.group("x"))
        cb.measure(cfg)
        repo.csv.write_text(table("sec/op", "New-8,,,5e-09,0%"))
        assert cb.report(cfg)
        assert f"new {PACKAGE} New-8" in capsys.readouterr().out

    def test_should_pass_on_a_control_that_breaks_the_rule(
        self, repo: Repo, capsys: pytest.CaptureFixture[str]
    ) -> None:
        cfg = repo.config(repo.group("floor", control=True))
        cb.measure(cfg)
        repo.control_csv.write_text(SLOWER)
        assert cb.report(cfg)
        assert f"noisy control {PACKAGE} X +100.00%" in capsys.readouterr().out

    def test_should_fail_on_a_control_that_head_does_not_have(
        self, repo: Repo, capsys: pytest.CaptureFixture[str]
    ) -> None:
        cfg = repo.config(repo.group("floor", control=True))
        cb.measure(cfg)
        repo.control_csv.write_text(table("sec/op", "Gone-8,5e-09,0%"))
        assert not cb.report(cfg)
        assert f"FAIL missing {PACKAGE} Gone-8" in capsys.readouterr().out


class TestParse:
    def test_should_select_the_groups_of_the_suite(self, tmp_path: Path) -> None:
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
            (["--rounds", "0"], "--rounds must be 4 or more to find a change"),
            (["--rounds", "3"], "--rounds must be 4 or more to find a change"),
        ],
    )
    def test_should_reject(
        self, args: list[str], message: str, capsys: pytest.CaptureFixture[str]
    ) -> None:
        with pytest.raises(SystemExit) as error:
            cb.parse(["--base", "a", "--head", "b", *args])
        assert error.value.code == 2
        assert message in capsys.readouterr().err

    def test_should_accept_the_fewest_rounds_that_can_find_a_change(self) -> None:
        assert cb.parse(["--base", "a", "--head", "b", "--rounds", "4"]).rounds == 4

    def test_should_refuse_an_existing_out_directory(self, tmp_path: Path) -> None:
        with pytest.raises(FileExistsError):
            cb.parse(["--base", "a", "--head", "b", "--out", str(tmp_path)])


class TestGroups:
    def test_should_build_one_go_binary_per_package(self) -> None:
        group = cb.go("cesium/write", "cpu", "cesium", "^BenchmarkWrite$")
        assert group.build == (
            cb.Step("cesium", ("go", "test", "-c", "-o", "{bin}/cesium.test", ".")),
        )
        assert group.run.argv[:3] == (
            "{bin}/cesium.test",
            "-test.run=^$",
            "-test.bench=^BenchmarkWrite$",
        )

    def test_should_have_vitest_write_a_json_report(self) -> None:
        group = cb.vitest("ts/codec", "src/framer/codec.bench.ts")
        assert (group.suite, group.format) == ("cpu", "vitest")
        assert group.run.directory == "client/ts"
        assert group.run.argv[-3:] == (
            "--reporter=json",
            "--outputFile={json}",
            "src/framer/codec.bench.ts",
        )

    def test_should_run_the_gbench_binary_that_bazel_builds(self) -> None:
        group = cb.gbench("cpp/codec", "client/cpp/framer", "codec_bench")
        target = "//client/cpp/framer:codec_bench"
        assert group.build == (cb.Step(".", ("bazel", "build", "-c", "opt", target)),)
        assert group.format == "gbench"
        assert group.run == cb.Step(
            ".",
            (
                "bazel-bin/client/cpp/framer/codec_bench",
                "--benchmark_out={json}",
                "--benchmark_out_format=json",
            ),
        )

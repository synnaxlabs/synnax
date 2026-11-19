#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Compares the gate benchmarks of two Git refs and fails on a regression.

Usage: uv run --project scripts scripts/compare_benchmarks.py --base main --head HEAD
    [--rounds 10] [--suite cpu|io] [--io-dir <dir>] [--only <regex>] [--out <dir>]

Builds each ref once in its own worktree. Each round then runs every group once per
ref, back to back, and odd rounds run head first. One process runs at a time.
benchstat compares the two result files.

A benchmark fails on a sec/op rise over 5% at p < 0.05, on an allocs/op rise of any
size at p < 0.05, or when head does not have it. A failure exits with 1. A control
group that breaks the rule does not fail the run: it means the host is noisy.

Pass one ref as both --base and --head for an A/A run, which measures the noise of the
host. The io suite puts its files in --io-dir, the disk to measure.
"""

import argparse
import csv
import io
import json
import os
import platform
import re
import subprocess
import sys
import tempfile
from collections.abc import Callable, Sequence
from dataclasses import dataclass, field, replace
from pathlib import Path
from typing import Any, Literal

ROOT = Path(__file__).resolve().parent.parent
BENCHSTAT = (
    "go",
    "run",
    "golang.org/x/perf/cmd/benchstat@v0.0.0-20261117185222-4febdb082b2c",
)
SIDES = ("base", "head")
# benchstat finds no change at p < 0.05 with fewer samples on each side.
MIN_ROUNDS = 4
type Suite = Literal["cpu", "io"]
BENCHMARK = re.compile(r"^Benchmark", re.MULTILINE)
ALPHA = 0.05
MAX_TIME_RISE = 0.05


@dataclass(frozen=True)
class Step:
    """A command and the repo-relative directory it runs in.

    `{bin}` in an argument is the directory that holds the builds of the ref, and
    `{json}` is the file that a run writes its JSON report to.
    """

    directory: str
    argv: tuple[str, ...]


@dataclass(frozen=True)
class Group:
    """One benchmark process of the gate, and the steps that build it."""

    name: str
    suite: Suite
    build: tuple[Step, ...]
    run: Step
    control: bool = False
    # The key in FORMATS of the results that the run gives.
    format: str = "go"


def go(
    name: str, suite: Suite, package: str, pattern: str, control: bool = False
) -> Group:
    """Returns the group for the Go benchmarks of package that match pattern."""
    binary = "{bin}/" + package.replace("/", "-") + ".test"
    return Group(
        name,
        suite,
        (Step(package, ("go", "test", "-c", "-o", binary, ".")),),
        Step(
            package,
            (
                binary,
                "-test.run=^$",
                f"-test.bench={pattern}",
                "-test.benchmem",
                "-test.timeout=0",
            ),
        ),
        control,
    )


def vitest(name: str, *files: str) -> Group:
    """Returns the group for the vitest benchmark files of the TypeScript client."""
    return Group(
        name,
        "cpu",
        (
            Step(".", ("pnpm", "install", "--frozen-lockfile")),
            Step(".", ("pnpm", "build:client")),
        ),
        Step(
            "client/ts",
            (
                "pnpm",
                "vitest",
                "bench",
                "--run",
                "--config=vitest.bench.config.ts",
                "--reporter=json",
                "--outputFile={json}",
                *files,
            ),
        ),
        format="vitest",
    )


def gbench(name: str, package: str, binary: str) -> Group:
    """Returns the group for a Google Benchmark binary that Bazel builds."""
    return Group(
        name,
        "cpu",
        (Step(".", ("bazel", "build", "-c", "opt", f"//{package}:{binary}")),),
        Step(
            ".",
            (
                f"bazel-bin/{package}/{binary}",
                "--benchmark_out={json}",
                "--benchmark_out_format=json",
            ),
        ),
        format="gbench",
    )


FRAMER = "core/pkg/service/framer"
GROUPS: tuple[Group, ...] = (
    go("cesium/write", "cpu", "cesium", "^BenchmarkWrite$/^fs=mem$"),
    go("cesium/write-os", "io", "cesium", "^BenchmarkWrite$/^fs=os$"),
    go("cesium/commit", "io", "cesium", "^BenchmarkCommit$"),
    go(
        "cesium/read",
        "cpu",
        "cesium",
        "^Benchmark(ReadDomains|ReadSamples|StreamFixedSpan)$",
    ),
    go(
        "framer/writer",
        "cpu",
        "core/pkg/distribution/framer/writer",
        "^Benchmark(Write|Open)$",
    ),
    go(
        "framer/codec",
        "cpu",
        "core/pkg/distribution/framer/codec",
        "^Benchmark(Encode|Decode)$",
    ),
    go("api/stream", "cpu", "core/pkg/api/framer", "^BenchmarkStreamKeyChange$"),
    go("iterator/calc", "cpu", f"{FRAMER}/iterator", "^BenchmarkIteratorCalc_"),
    go(
        "calculator", "cpu", f"{FRAMER}/calculation/calculator", "^BenchmarkCalculator_"
    ),
    vitest("ts/codec", "src/framer/codec.bench.ts"),
    vitest("ts/cache", "src/framer/cache/"),
    Group(
        "py/codec",
        "cpu",
        (Step("client/py", ("uv", "sync")),),
        Step("client/py", ("uv", "run", "python", "benchmarks/frame_codec.py")),
    ),
    gbench("cpp/codec", "client/cpp/framer", "codec_bench"),
    gbench("driver/sample-clock", "driver/common", "sample_clock_bench"),
    go("floors/cpu", "cpu", "cesium", "^BenchmarkFloor(Copy|CRC32C)$", control=True),
    go("floors", "io", "cesium", "^BenchmarkFloor", control=True),
)

HOST_FACTS: dict[str, tuple[tuple[str, ...], ...]] = {
    "Linux": (
        ("uname", "-srm"),
        ("lscpu",),
        ("go", "version"),
        ("findmnt", "-T", "{dir}"),
        ("lsblk", "-o", "NAME,MODEL,SIZE,ROTA"),
    ),
    "Darwin": (
        ("uname", "-srm"),
        ("sysctl", "-n", "machdep.cpu.brand_string"),
        ("go", "version"),
        ("df", "-h", "{dir}"),
    ),
}


@dataclass(frozen=True)
class Config:
    """The inputs of one comparison."""

    base: str
    head: str
    out: Path
    rounds: int = 10
    suite: Suite = "cpu"
    # The directory the benchmarks put their files in. The io suite needs it.
    io_dir: Path | None = None
    # Limits the run to the groups whose name it matches.
    only: str = ""
    repo: Path = ROOT
    groups: tuple[Group, ...] = GROUPS
    benchstat: tuple[str, ...] = BENCHSTAT
    host_facts: tuple[tuple[str, ...], ...] = field(
        default_factory=lambda: HOST_FACTS[platform.system()]
    )

    def __post_init__(self) -> None:
        # Steps run in other directories, so every path under out must be absolute.
        object.__setattr__(self, "out", self.out.resolve())

    def selected(self) -> tuple[Group, ...]:
        """Returns the groups of the suite that match only."""
        return tuple(
            g
            for g in self.groups
            if g.suite == self.suite and re.search(self.only, g.name)
        )

    def source(self, side: str) -> Path:
        return self.out / "src" / side

    def binaries(self, side: str) -> Path:
        return self.out / "bin" / side

    @property
    def report(self) -> Path:
        return self.out / "run.json"

    def results(self, side: str, control: bool) -> Path:
        return self.out / (f"control-{side}.txt" if control else f"{side}.txt")


def from_vitest(report: Any) -> str:
    """Returns the Go benchmark lines for a vitest JSON report."""
    lines = []
    for file in report["testResults"]:
        stem = Path(file["name"]).name.removesuffix(".bench.ts")
        for test in file["assertionResults"]:
            for benchmark in test.get("benchmarks", ()):
                for task in benchmark["tasks"]:
                    name = "/".join((stem, test["fullName"], task["name"]))
                    latency = task["latency"]
                    lines.append(
                        f"BenchmarkTS/{'_'.join(name.split())} "
                        f"{latency['samplesCount']} {latency['mean'] * 1e6} ns/op"
                    )
    return "".join(f"{line}\n" for line in lines)


TIME_UNITS = {"ns": 1, "us": 1e3, "ms": 1e6, "s": 1e9}


def from_gbench(report: Any) -> str:
    """Returns the Go benchmark lines for a Google Benchmark JSON report."""
    lines = []
    for benchmark in report["benchmarks"]:
        if benchmark["run_type"] != "iteration":
            continue
        if benchmark.get("error_occurred"):
            raise RuntimeError(f"{benchmark['name']}: {benchmark['error_message']}")
        name = benchmark["name"].removeprefix("BM_")
        time = benchmark["real_time"] * TIME_UNITS[benchmark["time_unit"]]
        line = f"Benchmark{name} {benchmark['iterations']} {time} ns/op"
        if "allocs/op" in benchmark:
            line += f" {benchmark['allocs/op']} allocs/op"
        lines.append(line)
    return "".join(f"{line}\n" for line in lines)


# Each format turns what a run printed and the JSON report it wrote into Go benchmark
# lines.
FORMATS: dict[str, Callable[[str, Path], str]] = {
    "go": lambda printed, _: printed,
    "vitest": lambda _, report: from_vitest(json.loads(report.read_text())),
    "gbench": lambda _, report: from_gbench(json.loads(report.read_text())),
}


def order(round_: int) -> tuple[str, str]:
    """Returns the sides in the order round_ runs them."""
    return SIDES if round_ % 2 == 0 else (SIDES[1], SIDES[0])


def run(cfg: Config, side: str, step: Step) -> str:
    """Runs step in the worktree of side and returns what it printed."""
    paths = {"{bin}": cfg.binaries(side), "{json}": cfg.report}
    argv = list(step.argv)
    for placeholder, path in paths.items():
        argv = [a.replace(placeholder, str(path)) for a in argv]
    env = dict(os.environ)
    if cfg.io_dir is not None:
        env["TMPDIR"] = str(cfg.io_dir)
    result = subprocess.run(
        argv,
        cwd=cfg.source(side) / step.directory,
        env=env,
        stdout=subprocess.PIPE,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        sys.stderr.write(result.stdout)
        result.check_returncode()
    return result.stdout


def record_host(cfg: Config, moment: str) -> None:
    """Appends the load average and, at the start, the host facts to host.txt."""
    directory = str(cfg.io_dir or tempfile.gettempdir())
    with (cfg.out / "host.txt").open("a") as f:
        f.write(f"load at {moment}: {os.getloadavg()}\n")
        f.flush()
        if moment != "start":
            return
        for fact in cfg.host_facts:
            argv = [a.replace("{dir}", directory) for a in fact]
            subprocess.run(argv, stdout=f, check=True)


def measure(cfg: Config) -> None:
    """Builds both refs, then appends the results of each round to the result files."""
    groups = cfg.selected()
    record_host(cfg, "start")
    added = []
    try:
        for side, ref in zip(SIDES, (cfg.base, cfg.head), strict=True):
            subprocess.run(
                ["git", "worktree", "add", "--detach", str(cfg.source(side)), ref],
                cwd=cfg.repo,
                check=True,
            )
            added.append(side)
        for side in SIDES:
            cfg.binaries(side).mkdir(parents=True)
            for step in dict.fromkeys(s for g in groups for s in g.build):
                run(cfg, side, step)
        for round_ in range(cfg.rounds):
            for group in groups:
                for side in order(round_):
                    print(
                        f"round {round_ + 1}/{cfg.rounds} {group.name} {side}",
                        flush=True,
                    )
                    # A run that writes no report must not read the report before.
                    cfg.report.unlink(missing_ok=True)
                    printed = run(cfg, side, group.run)
                    results = FORMATS[group.format](printed, cfg.report)
                    # Base can lack a benchmark that head adds.
                    if side == SIDES[1] and not BENCHMARK.search(results):
                        raise RuntimeError(f"{group.name} ran no benchmark on head")
                    # A Go test binary prints its own pkg line, which then applies.
                    with cfg.results(side, group.control).open("a") as f:
                        f.write(f"pkg: {group.name}\n{results}")
        record_host(cfg, "end")
    finally:
        for side in added:
            subprocess.run(
                ["git", "worktree", "remove", "--force", str(cfg.source(side))],
                cwd=cfg.repo,
                check=True,
            )


@dataclass(frozen=True)
class Finding:
    """One benchmark that breaks the rule, or that only head has."""

    # One of "slower", "allocations", "missing", and "new".
    kind: str
    package: str
    benchmark: str
    detail: str = ""


def judge(table: str) -> list[Finding]:
    """Applies the rule to the CSV that benchstat prints for a base and a head file."""
    findings = []
    package = unit = ""
    # The column of each side in the current table. A table has one side only when the
    # other side has no result for its package.
    columns: dict[str, int] = {}
    for row in csv.reader(io.StringIO(table)):
        if not row or row[0] == "geomean":
            continue
        if row[0].startswith("pkg: "):
            package = row[0].removeprefix("pkg: ")
        if len(row) == 1:
            continue
        if row[0] == "" and row[1] in SIDES:
            columns = {side: row.index(side) for side in SIDES if side in row}
            continue
        if row[0] == "":
            unit = row[1]
            continue
        cells = [*row, "", "", "", "", ""]
        name, change = cells[0], cells[5]
        base, head = (cells[columns[s]] if s in columns else "" for s in SIDES)
        if base == "" or head == "":
            if unit == "sec/op":
                kind = "missing" if head == "" else "new"
                findings.append(Finding(kind, package, name))
            continue
        before, after = float(base), float(head)
        if change in ("", "~") or after <= before:
            continue
        if unit == "sec/op" and after > before * (1 + MAX_TIME_RISE):
            detail = f"{after / before - 1:+.2%}"
            findings.append(Finding("slower", package, name, detail))
        if unit == "allocs/op":
            detail = f"{before:g} to {after:g} allocs/op"
            findings.append(Finding("allocations", package, name, detail))
    return findings


def benchstat(cfg: Config, control: bool, *flags: str) -> str:
    files = [f"{side}={cfg.results(side, control)}" for side in SIDES]
    # benchstat prints a note on stderr for each cell it cannot compare.
    result = subprocess.run(
        [*cfg.benchstat, "-alpha", str(ALPHA), *flags, *files],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        raise RuntimeError(f"benchstat failed: {result.stderr}")
    return result.stdout


def report(cfg: Config) -> bool:
    """Prints the comparison and each finding. Returns whether the gate passes."""
    passed = True
    for control in (False, True):
        if not cfg.results(SIDES[0], control).exists():
            continue
        text = benchstat(cfg, control)
        with (cfg.out / "report.txt").open("a") as f:
            f.write(text)
        print(text)
        for finding in judge(benchstat(cfg, control, "-format", "csv")):
            label = f"FAIL {finding.kind}"
            if finding.kind == "new":
                label = "new"
            elif control and finding.kind != "missing":
                label = "noisy control"
            else:
                passed = False
            print(label, finding.package, finding.benchmark, finding.detail)
    print(f"verdict: {'pass' if passed else 'fail'}, results in {cfg.out}")
    return passed


def parse(argv: Sequence[str] | None) -> Config:
    parser = argparse.ArgumentParser(description=__doc__.split("\n", maxsplit=1)[0])
    parser.add_argument("--base", required=True, help="the Git ref to compare against")
    parser.add_argument("--head", required=True, help="the Git ref under test")
    parser.add_argument("--rounds", type=int, default=10)
    parser.add_argument("--suite", choices=("cpu", "io"), default="cpu")
    parser.add_argument(
        "--io-dir", type=Path, help="a directory on the disk to measure"
    )
    parser.add_argument("--only", default="", help="a regex on the group names")
    parser.add_argument("--out", type=Path, help="a new directory for the results")
    args = parser.parse_args(argv)
    if args.suite == "io" and args.io_dir is None:
        parser.error("the io suite needs --io-dir")
    if args.rounds < MIN_ROUNDS:
        parser.error(f"--rounds must be {MIN_ROUNDS} or more to find a change")
    cfg = Config(
        args.base, args.head, Path(), args.rounds, args.suite, args.io_dir, args.only
    )
    if not cfg.selected():
        parser.error(f"no {args.suite} group matches {args.only!r}")
    if args.out is None:
        return replace(cfg, out=Path(tempfile.mkdtemp(prefix="benchmarks-")))
    args.out.mkdir(parents=True)
    return replace(cfg, out=args.out)


def main(argv: Sequence[str] | None = None) -> int:
    cfg = parse(argv)
    measure(cfg)
    return 0 if report(cfg) else 1


if __name__ == "__main__":
    sys.exit(main())

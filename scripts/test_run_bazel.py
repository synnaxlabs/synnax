#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parent / "run_bazel.sh"

CORRUPT_REPO = "ERROR: Error loading '@@grpc+//:grpc': no such package"
LD_TRAILER = (
    "Undefined symbols for architecture arm64:\n"
    '  "_AbslInternalSpinLockWake_lts_20260817", referenced from:\n'
    "ld: symbol(s) not found for architecture arm64"
)
EXTERNAL_LINK = (
    "ERROR: external/protobuf+/upb_generator/c/BUILD:43:20: "
    "Linking external/protobuf+/upb_generator/c/protoc-gen-upb_stage0 failed: "
    f"(Exit 1)\n{LD_TRAILER}"
)
WORKSPACE_LINK = (
    f"ERROR: driver/BUILD.bazel:12:10: Linking driver/driver failed: (Exit 1)\n"
    f"{LD_TRAILER}"
)
COMPILE_ERROR = "driver/rack/rack.cpp:14:3: error: no member named 'strt'"

FAKE_BAZEL = """#!/bin/bash
echo "$@" >> "$BAZEL_CALLS"
case "$1" in
    info) echo "$BAZEL_REPO_CACHE"; exit 0 ;;
    clean) exit 0 ;;
esac
builds=$(grep -cE '^(build|test)' "$BAZEL_CALLS")
if [ "$builds" -eq 1 ] && [ -n "$BAZEL_FAIL_OUTPUT" ]; then
    printf '%s\\n' "$BAZEL_FAIL_OUTPUT"
    exit "$BAZEL_FAIL_CODE"
fi
echo "build ok"
exit 0
"""


class Runner:
    """Drives run_bazel.sh against a fake bazel that fails its first build."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.calls = path / "calls.txt"
        self.calls.write_text("")
        self.repo_cache = path / "repocache"
        (self.repo_cache / "contents").mkdir(parents=True)
        bin_dir = path / "bin"
        bin_dir.mkdir()
        bazel = bin_dir / "bazel"
        bazel.write_text(FAKE_BAZEL)
        bazel.chmod(0o755)
        self.bin_dir = bin_dir

    def run(
        self, fail_output: str = "", fail_code: int = 1, args: list[str] | None = None
    ) -> subprocess.CompletedProcess[str]:
        env = os.environ | {
            "PATH": f"{self.bin_dir}:{os.environ['PATH']}",
            "BAZEL_CALLS": str(self.calls),
            "BAZEL_REPO_CACHE": str(self.repo_cache),
            "BAZEL_FAIL_OUTPUT": fail_output,
            "BAZEL_FAIL_CODE": str(fail_code),
        }
        return subprocess.run(
            [str(SCRIPT), *(["build", "//driver"] if args is None else args)],
            capture_output=True,
            text=True,
            check=False,
            env=env,
        )

    @property
    def builds(self) -> int:
        lines = self.calls.read_text().splitlines()
        return len([x for x in lines if x.startswith(("build", "test"))])

    @property
    def expunged(self) -> bool:
        return "clean --expunge" in self.calls.read_text()


@pytest.fixture
def runner(tmp_path: Path) -> Runner:
    return Runner(tmp_path)


class TestSuccess:
    def test_should_pass_through_a_clean_build(self, runner: Runner) -> None:
        result = runner.run()
        assert result.returncode == 0, result.stderr
        assert runner.builds == 1
        assert not runner.expunged

    def test_should_reject_no_arguments(self, runner: Runner) -> None:
        result = runner.run(args=[])
        assert result.returncode == 1
        assert "Usage" in result.stdout


class TestRetries:
    """The two failures worth a second attempt, each costing one expunge."""

    def test_should_retry_an_unloadable_repo(self, runner: Runner) -> None:
        result = runner.run(CORRUPT_REPO)
        assert result.returncode == 0, result.stderr
        assert runner.builds == 2
        assert runner.expunged
        assert not (runner.repo_cache / "contents").exists()

    def test_should_retry_a_stale_external_link(self, runner: Runner) -> None:
        result = runner.run(EXTERNAL_LINK)
        assert result.returncode == 0, result.stderr
        assert runner.builds == 2
        assert runner.expunged


class TestNoRetry:
    """A source defect must fail at once, with the caches left alone."""

    def test_should_not_retry_a_workspace_link_failure(self, runner: Runner) -> None:
        result = runner.run(WORKSPACE_LINK)
        assert result.returncode == 1
        assert runner.builds == 1
        assert not runner.expunged
        assert (runner.repo_cache / "contents").exists()

    def test_should_not_retry_a_compile_error(self, runner: Runner) -> None:
        result = runner.run(COMPILE_ERROR)
        assert result.returncode == 1
        assert runner.builds == 1
        assert not runner.expunged

    def test_should_preserve_the_exit_code(self, runner: Runner) -> None:
        assert runner.run(COMPILE_ERROR, fail_code=3).returncode == 3

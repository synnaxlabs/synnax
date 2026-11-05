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
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "gh_api.sh"
EMPTY = "unexpected end of JSON input"


@pytest.fixture
def gh(tmp_path: Path) -> FakeGH:
    return FakeGH(tmp_path)


def run(gh: FakeGH, *args: str) -> subprocess.CompletedProcess[str]:
    env = {
        **os.environ,
        "PATH": f"{gh.bin_dir}:{os.environ['PATH']}",
        "GH_API_RETRY_DELAY": "0",
    }
    return subprocess.run([str(SCRIPT), *args], env=env, capture_output=True, text=True)


class TestGhAPI:
    def test_passes_a_success_through(self, gh: FakeGH) -> None:
        """Should print the response of the one call it makes."""
        gh.serve("api */pulls/7", {"state": "open"})
        result = run(gh, "repos/a/b/pulls/7", "--jq", ".state")
        assert result.returncode == 0
        assert result.stdout == "open\n"
        assert gh.calls() == [["api", "repos/a/b/pulls/7", "--jq", ".state"]]

    @pytest.mark.parametrize("failures", [1, 2])
    def test_retries_an_empty_response(self, gh: FakeGH, failures: int) -> None:
        """Should print only the response of the attempt that succeeds."""
        gh.serve("api */pulls/7", {"state": "open"})
        gh.fail("api */pulls/7", EMPTY, times=failures)
        result = run(gh, "repos/a/b/pulls/7", "--jq", ".state")
        assert result.returncode == 0
        assert result.stdout == "open\n"
        assert EMPTY not in result.stderr
        assert len(gh.calls()) == failures + 1

    def test_gives_up_after_three_attempts(self, gh: FakeGH) -> None:
        """Should fail with gh's error when every attempt gets an empty response."""
        gh.serve("api */pulls/7", {"state": "open"})
        gh.fail("api */pulls/7", EMPTY, times=3)
        result = run(gh, "repos/a/b/pulls/7")
        assert result.returncode == 1
        assert result.stdout == ""
        assert result.stderr.endswith(f"{EMPTY}\n")
        assert len(gh.calls()) == 3

    def test_keeps_a_real_error_loud(self, gh: FakeGH) -> None:
        """Should fail at once with gh's error when GitHub rejects the request."""
        gh.serve("api */pulls/7", {"state": "open"})
        gh.fail("api */pulls/7", "gh: Not Found (HTTP 404)")
        result = run(gh, "repos/a/b/pulls/7")
        assert result.returncode == 1
        assert result.stderr == "gh: Not Found (HTTP 404)\n"
        assert len(gh.calls()) == 1

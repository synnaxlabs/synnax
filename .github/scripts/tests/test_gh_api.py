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
TEXT = "invalid character 'S' looking for beginning of value"


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

    def test_retries_a_plain_text_response(self, gh: FakeGH) -> None:
        """Should retry a response body that is not JSON."""
        gh.serve("api */pulls/7", {"state": "open"})
        gh.fail("api */pulls/7", TEXT)
        result = run(gh, "repos/a/b/pulls/7", "--jq", ".state")
        assert result.returncode == 0
        assert result.stdout == "open\n"
        assert TEXT not in result.stderr
        assert len(gh.calls()) == 2

    def test_gives_up_after_five_attempts(self, gh: FakeGH) -> None:
        """Should fail with gh's error when every attempt gets an empty response."""
        gh.serve("api */pulls/7", {"state": "open"})
        gh.fail("api */pulls/7", EMPTY, times=5)
        result = run(gh, "repos/a/b/pulls/7")
        assert result.returncode == 1
        assert result.stdout == ""
        assert result.stderr.endswith(f"{EMPTY}\n")
        assert len(gh.calls()) == 5

    def test_keeps_a_real_error_loud(self, gh: FakeGH) -> None:
        """Should fail at once with gh's error when GitHub rejects the request."""
        gh.serve("api */pulls/7", {"state": "open"})
        gh.fail("api */pulls/7", "gh: Not Found (HTTP 404)")
        result = run(gh, "repos/a/b/pulls/7")
        assert result.returncode == 1
        assert result.stderr == "gh: Not Found (HTTP 404)\n"
        assert len(gh.calls()) == 1

    @pytest.mark.parametrize(
        "error",
        [
            "gh: Server Error (HTTP 500)",
            "could not create workflow dispatch event: HTTP 502 (https://x/dispatches)",
            'Post "https://x": dial tcp: lookup api.github.com: no such host',
            "read tcp 10.0.0.1:443: read: connection reset by peer",
            'Post "https://x": dial tcp 140.82.112.6:443: i/o timeout',
            'Post "https://x": net/http: TLS handshake timeout',
            'Post "https://x": unexpected EOF',
        ],
    )
    def test_retries_a_server_or_network_error(self, gh: FakeGH, error: str) -> None:
        """Should retry an HTTP 5xx or network failure and print the success."""
        gh.serve("api */dispatches")
        gh.fail("api */dispatches", error)
        result = run(gh, "repos/a/b/actions/workflows/c.yaml/dispatches", "-X", "POST")
        assert result.returncode == 0
        assert error not in result.stderr
        assert len(gh.calls()) == 2

    @pytest.mark.parametrize(
        "error",
        [
            "gh: Workflow does not have 'workflow_dispatch' trigger (HTTP 422)",
            "gh: Validation Failed (HTTP 422): input is over 500 characters",
        ],
    )
    def test_fails_a_client_error_at_once(self, gh: FakeGH, error: str) -> None:
        """Should not retry a 4xx, even one whose text holds 500."""
        gh.serve("api */dispatches")
        gh.fail("api */dispatches", error)
        result = run(gh, "repos/a/b/actions/workflows/c.yaml/dispatches", "-X", "POST")
        assert result.returncode == 1
        assert result.stderr == f"{error}\n"
        assert len(gh.calls()) == 1

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
import threading
from collections.abc import Iterator
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from typing import Any

import pytest

from security_osv import main, queries, query_osv, sarif, tag

MODULE = """
bazel_dep(name = "cpp-httplib", version = "0.46.0")
bazel_dep(
    name = "curl",
    version = "8.22.0",
)
bazel_dep(name = "grpc", version = "1.84.0")
bazel_dep(name = "openssl", version = "4.0.1.bcr.0")
bazel_dep(name = "rules_cc", version = "0.2.26")
"""

GITLINKS = """100644 aaaa 0\tMODULE.bazel
160000 b1e4008bc4611a422a86704a60c29d390fa486a5 0\tvendor/open62541/open62541
"""


class OSV:
    """A local OSV that answers every query with one vulnerability."""

    def __init__(self) -> None:
        self.queries: list[dict[str, Any]] = []
        received = self.queries

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self) -> None:
                length = int(self.headers["Content-Length"])
                received.append(json.loads(self.rfile.read(length)))
                body = json.dumps({"vulns": [{"id": "X-1", "summary": "Bad"}]})
                self.send_response(200)
                self.end_headers()
                self.wfile.write(body.encode())

            def log_message(self, *_: Any) -> None:
                pass

        self.server = HTTPServer(("127.0.0.1", 0), Handler)
        self.url = f"http://127.0.0.1:{self.server.server_port}"


@pytest.fixture
def osv() -> Iterator[OSV]:
    server = OSV()
    thread = threading.Thread(target=server.server.serve_forever)
    thread.start()
    yield server
    server.server.shutdown()
    thread.join()
    server.server.server_close()


class TestTag:
    """Tests for naming the upstream tag of a Bazel module version."""

    def test_should_drop_the_registry_suffix(self) -> None:
        """Should strip the .bcr.N suffix the Bazel registry adds."""
        assert tag("openssl", "4.0.1.bcr.0") == "openssl-4.0.1"

    def test_should_write_curl_versions_with_underscores(self) -> None:
        """Should match curl's tag form."""
        assert tag("curl", "8.22.0") == "curl-8_22_0"


class TestQueries:
    """Tests for choosing the dependencies to check."""

    def test_should_check_known_modules_and_submodules(self) -> None:
        """Should skip Bazel modules OSV has no Git data for, and plain files."""
        found = queries(MODULE, GITLINKS)
        assert [dependency for _, dependency, _ in found] == [
            "cpp-httplib 0.46.0",
            "curl 8.22.0",
            "grpc 1.84.0",
            "openssl 4.0.1.bcr.0",
            "vendor/open62541/open62541 b1e4008bc461",
        ]
        curl = {"name": "https://github.com/curl/curl.git", "ecosystem": "GIT"}
        assert found[1][2] == {"package": curl, "version": "curl-8_22_0"}
        assert found[4][2] == {"commit": "b1e4008bc4611a422a86704a60c29d390fa486a5"}

    def test_should_reject_a_module_it_cannot_find(self) -> None:
        """Should fail when a checked Bazel module has no readable version."""
        with pytest.raises(ValueError, match="no version for grpc"):
            queries(MODULE.replace('"grpc"', '"grpc_old"'), GITLINKS)


class TestSarif:
    """Tests for writing the vulnerabilities as SARIF."""

    def test_should_write_one_error_per_vulnerability(self) -> None:
        """Should name the dependency in each message."""
        log = sarif(
            [("MODULE.bazel", "curl 8.22.0", [{"id": "X-1", "summary": "Bad"}])]
        )
        assert log["runs"][0]["results"] == [
            {
                "ruleId": "X-1",
                "level": "error",
                "message": {"text": "curl 8.22.0: Bad"},
                "locations": [
                    {"physicalLocation": {"artifactLocation": {"uri": "MODULE.bazel"}}}
                ],
            }
        ]


class TestQueryOsv:
    """Tests for querying OSV."""

    def test_should_send_the_query_and_return_the_vulnerabilities(
        self, osv: OSV
    ) -> None:
        """Should post the query as JSON and read the vulns field."""
        assert query_osv({"commit": "abc"}, osv.url) == [
            {"id": "X-1", "summary": "Bad"}
        ]
        assert osv.queries == [{"commit": "abc"}]


class TestMain:
    """Tests for checking a repository."""

    def test_should_print_the_vulnerabilities_as_sarif(
        self,
        osv: OSV,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        """Should check each Bazel module in MODULE.bazel."""
        (tmp_path / "MODULE.bazel").write_text(MODULE)
        subprocess.run(["git", "init", "-q"], cwd=tmp_path, check=True)
        subprocess.run(["git", "add", "MODULE.bazel"], cwd=tmp_path, check=True)
        monkeypatch.chdir(tmp_path)
        assert main(osv.url) == 0
        results = json.loads(capsys.readouterr().out)["runs"][0]["results"]
        assert [r["message"]["text"] for r in results] == [
            "cpp-httplib 0.46.0: Bad",
            "curl 8.22.0: Bad",
            "grpc 1.84.0: Bad",
            "openssl 4.0.1.bcr.0: Bad",
        ]

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from security_osv import queries, sarif, tag

MODULE = """
bazel_dep(name = "curl", version = "8.22.0")
bazel_dep(name = "openssl", version = "4.0.1.bcr.0")
bazel_dep(name = "rules_cc", version = "0.2.26")
"""

GITLINKS = """100644 aaaa 0\tMODULE.bazel
160000 b1e4008bc4611a422a86704a60c29d390fa486a5 0\tvendor/open62541/open62541
"""


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
        assert queries(MODULE, GITLINKS) == [
            (
                "MODULE.bazel",
                "curl 8.22.0",
                {
                    "package": {
                        "name": "https://github.com/curl/curl.git",
                        "ecosystem": "GIT",
                    },
                    "version": "curl-8_22_0",
                },
            ),
            (
                "MODULE.bazel",
                "openssl 4.0.1.bcr.0",
                {
                    "package": {
                        "name": "https://github.com/openssl/openssl.git",
                        "ecosystem": "GIT",
                    },
                    "version": "openssl-4.0.1",
                },
            ),
            (
                "vendor/open62541/open62541",
                "vendor/open62541/open62541 b1e4008bc461",
                {"commit": "b1e4008bc4611a422a86704a60c29d390fa486a5"},
            ),
        ]


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

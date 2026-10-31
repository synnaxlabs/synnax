#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import shutil
import subprocess
import sys
import tempfile
from collections.abc import Iterator
from pathlib import Path
from typing import Any

import pytest

from security_report import Finding, Send, main, read_findings


def result(rule: str, level: str, uri: str, line: int) -> dict[str, Any]:
    return {
        "ruleId": rule,
        "level": level,
        "message": {"text": f"{rule}\n  at {uri}"},
        "locations": [
            {
                "physicalLocation": {
                    "artifactLocation": {"uri": uri},
                    "region": {"startLine": line},
                }
            }
        ],
    }


def write_scan(out: Path, name: str, code: int, *results: dict[str, Any]) -> None:
    (out / f"{name}.exit").write_text(f"{code}\n")
    (out / f"{name}.log").write_text("scanner said no\n")
    (out / f"{name}.sarif").write_text(json.dumps({"runs": [{"results": results}]}))


class TestReadFindings:
    """Tests for reading findings out of a SARIF log."""

    def test_should_skip_notes_and_duplicates(self) -> None:
        """Should count errors and warnings once each."""
        sarif = {
            "runs": [
                {
                    "results": [
                        result("a", "error", "x.go", 3),
                        result("a", "error", "x.go", 3),
                        result("b", "note", "y.go", 1),
                    ]
                }
            ]
        }
        assert read_findings(sarif) == [Finding("a", "error", "x.go:3", "a at x.go")]


class TestMain:
    """Tests for building the report from a scan directory."""

    def test_should_group_findings_by_rule(self, tmp_path: Path) -> None:
        """Should write each scan's findings under their rule."""
        write_scan(tmp_path, "gitleaks", 0, result("private-key", "error", "k.pem", 1))
        write_scan(tmp_path, "zizmor", 0)
        assert main([str(tmp_path)]) == 0
        report = (tmp_path / "report.md").read_text()
        assert report.startswith("Findings: 1.")
        assert "## gitleaks: 1 findings" in report
        assert "### `private-key` (error): 1" in report
        assert "- `k.pem:1`: private-key at k.pem" in report
        assert "## zizmor: 0 findings" in report

    def test_should_cap_the_findings_listed_per_rule(self, tmp_path: Path) -> None:
        """Should list the first findings of a rule and count the rest."""
        results = [result("a", "error", "x.yaml", n) for n in range(1, 31)]
        write_scan(tmp_path, "zizmor", 0, *results)
        assert main([str(tmp_path)]) == 0
        report = (tmp_path / "report.md").read_text()
        assert "- `x.yaml:25`" in report
        assert "- `x.yaml:26`" not in report
        assert "- And 5 more." in report

    def test_should_cut_a_report_at_the_size_limit(self, tmp_path: Path) -> None:
        """Should keep the report within the limit and count what it omits."""
        for n in range(200):
            results = [result(f"r{n}", "error", f"{n}.go", m) for m in range(1, 26)]
            write_scan(tmp_path, f"scan{n:03}", 0, *results)
        assert main([str(tmp_path)]) == 0
        report = (tmp_path / "report.md").read_text()
        assert report.startswith("Findings: 5000.")
        assert "## scan199" not in report
        assert report.endswith("listed findings.")

    def test_should_reject_a_directory_with_no_scan(self, tmp_path: Path) -> None:
        """Should not report a clean audit when no scan ran."""
        with pytest.raises(SystemExit, match="holds no scan"):
            main([str(tmp_path)])

    def test_should_fail_and_keep_the_log_when_a_scan_fails(
        self, tmp_path: Path, capsys: pytest.CaptureFixture[str]
    ) -> None:
        """Should put a failed scan's log in the report, and only its name in stderr."""
        write_scan(tmp_path, "checkov", 2)
        assert main([str(tmp_path)]) == 1
        assert "scanner said no" in (tmp_path / "report.md").read_text()
        assert capsys.readouterr().err == "Failed scans: checkov\n"


class TestPublish:
    """Tests for filing the report and posting the count."""

    @pytest.fixture
    def posts(self, monkeypatch: pytest.MonkeyPatch) -> list[tuple[str, Any]]:
        monkeypatch.setenv("LINEAR_API_KEY", "key")
        monkeypatch.setenv("SLACK_WEBHOOK_URL", "https://slack")
        return []

    @staticmethod
    def sender(sent: list[tuple[str, Any]]) -> Send:
        """Records each post and answers as Linear does."""

        def send(url: str, payload: dict[str, Any], _: dict[str, str]) -> bytes:
            sent.append((url, payload))
            if "teams" in payload.get("query", ""):
                return b'{"data": {"teams": {"nodes": [{"id": "team"}]}}}'
            url_json = '{"url": "https://linear.app/synnax/issue/SY-1"}'
            return f'{{"data": {{"issueCreate": {{"issue": {url_json}}}}}}}'.encode()

        return send

    def test_should_file_an_issue_and_post_only_the_count(
        self, tmp_path: Path, posts: list[tuple[str, Any]]
    ) -> None:
        """Should send the report to Linear and only the count and link to Slack."""
        write_scan(tmp_path, "zizmor", 0, result("unpinned-uses", "error", "a.yaml", 1))
        assert main([str(tmp_path), "--publish"], self.sender(posts)) == 0
        create = posts[1][1]["variables"]["input"]
        assert create["teamId"] == "team"
        assert "unpinned-uses" in create["description"]
        assert posts[2] == (
            "https://slack",
            {
                "text": "Weekly security audit: 1 findings, 0 failed scans. "
                "https://linear.app/synnax/issue/SY-1"
            },
        )

    def test_should_file_nothing_without_findings(
        self, tmp_path: Path, posts: list[tuple[str, Any]]
    ) -> None:
        """Should post the zero count to Slack and file no issue."""
        write_scan(tmp_path, "zizmor", 0)
        assert main([str(tmp_path), "--publish"], self.sender(posts)) == 0
        assert posts == [
            (
                "https://slack",
                {"text": "Weekly security audit: 0 findings, 0 failed scans."},
            )
        ]


@pytest.fixture
def key() -> Iterator[tuple[list[str], str]]:
    """A gpg command with a fresh key pair, and the armored private key."""
    # A short home, since the gpg-agent socket path has a length limit.
    with tempfile.TemporaryDirectory(dir="/tmp") as home:
        gpg = ["gpg", "--homedir", home, "--batch", "--quiet", "--passphrase", ""]
        uid = "Report <report@example.com>"
        generate = [*gpg, "--quick-gen-key", uid, "default", "default", "never"]
        subprocess.run(generate, check=True, capture_output=True)
        export = [*gpg, "--pinentry-mode", "loopback", "--armor"]
        yield gpg, subprocess.check_output([*export, "--export-secret-keys"], text=True)
        subprocess.run(["gpgconf", "--homedir", home, "--kill", "all"], check=True)


@pytest.mark.skipif(shutil.which("gpg") is None, reason="needs gpg")
@pytest.mark.skipif(sys.platform == "win32", reason="needs a POSIX /tmp")
class TestAgents:
    """Tests for adding the encrypted agent reports."""

    def test_should_decrypt_and_count_each_report(
        self,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
        key: tuple[list[str], str],
    ) -> None:
        """Should add and count each report, and fail on a failed job."""
        gpg, private = key
        report = "### High: Token never expires\n\n- Location: `a.go:1`\n"
        encrypt = [*gpg, "--armor", "--trust-model", "always", "--encrypt", "-r"]
        armor = subprocess.check_output(
            [*encrypt, "report@example.com"], input=report, text=True
        )
        jobs = {
            "authn": {"result": "success", "outputs": {"report": armor}},
            "tls": {"result": "failure", "outputs": {}},
            # A report gpg cannot decrypt, as after a key rotation.
            "ci": {"result": "success", "outputs": {"report": "not a message"}},
        }
        monkeypatch.setenv("AGENT_REPORTS", json.dumps(jobs))
        monkeypatch.setenv("REPORT_PRIVATE_KEY", private)
        write_scan(tmp_path, "zizmor", 0)
        assert main([str(tmp_path), "--agents"]) == 1
        text = (tmp_path / "report.md").read_text()
        assert text.startswith("Findings: 1.")
        assert "## agent-authn: 1 findings\n\n### High: Token never expires" in text
        assert "## agent-tls: failed" in text
        assert "## agent-ci: failed" in text
        assert "## zizmor: 0 findings" in text

    def test_should_report_every_agent_failed_when_the_key_is_bad(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should keep the scanner results when the private key does not import."""
        jobs = {"authn": {"result": "success", "outputs": {"report": "x"}}}
        monkeypatch.setenv("AGENT_REPORTS", json.dumps(jobs))
        monkeypatch.setenv("REPORT_PRIVATE_KEY", "not a key")
        write_scan(tmp_path, "gitleaks", 0, result("private-key", "error", "k.pem", 1))
        assert main([str(tmp_path), "--agents"]) == 1
        text = (tmp_path / "report.md").read_text()
        assert "## agent-authn: failed" in text
        assert "## gitleaks: 1 findings" in text

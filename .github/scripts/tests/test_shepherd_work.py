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

SCRIPT = Path(__file__).resolve().parents[1] / "shepherd_work.sh"


FINDINGS = "<!-- sedosaur_outside_diff -->\n\n- **Bug** `a.go:1`\n"
COMMITTED = "2026-01-01T12:00:00Z"
QUEUE_BRANCH = "gh-readonly-queue/main/pr-7-base"


class Harness:
    """A fake gh serving one pull request, its CI runs, alerts, threads, queue
    removals, and comments."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
            "SHEPHERD_RETRY_SECONDS": "0",
        }
        self.thread_count = 0
        self.queue_events: list[dict[str, str]] = []
        self.pull()
        self.ok()
        self.runs()
        self.codeql(check="success")
        self.graphql()
        self.comments()

    def pull(self, state: str = "clean") -> None:
        self.gh.serve(
            "api */pulls/7", {"head": {"sha": "head"}, "mergeable_state": state}
        )

    def runs(
        self,
        *conclusions: str | None,
        groups: tuple[str, ...] = (),
        status: str = "completed",
    ) -> None:
        """Serves the CI and CodeQL runs on the head, then the merge group runs."""
        runs = [
            {"id": i + 1, "conclusion": c, "status": status, "head_branch": "branch"}
            for i, c in enumerate(conclusions)
        ]
        runs += [
            {"id": 9 + i, "conclusion": c, "head_branch": QUEUE_BRANCH}
            for i, c in enumerate(groups)
        ]
        self.gh.serve("api */actions/*", {"workflow_runs": runs})

    def codeql(
        self,
        *lines: int,
        severity: str = "error",
        security: str | None = "high",
        check: str = "failure",
    ) -> None:
        """Serves the CodeQL check on the head and the open alerts on the merge ref."""
        self.gh.serve("api */commits/*", {"check_runs": [{"conclusion": check}]})
        rule = {"id": "go/x", "severity": severity, "security_severity_level": security}
        alerts = [
            {
                "rule": rule,
                "most_recent_instance": {
                    "location": {"path": "a.go", "start_line": line},
                    "message": {"text": "Bad\n  cast."},
                },
            }
            for line in lines
        ]
        self.gh.serve("api */code-scanning/*", alerts)

    def ok(self, conclusion: str | None = None) -> None:
        jobs = [{"name": "OK", "conclusion": conclusion}]
        self.gh.serve("api */jobs*", {"jobs": jobs})

    def threads(self, count: int) -> None:
        self.thread_count = count
        self.graphql()

    def removed(self, at: str, reason: str = "failed_checks") -> None:
        self.queue_events = [{"createdAt": at, "reason": reason}]
        self.graphql()

    def enqueued(self) -> None:
        """An added event, which the query selects no fields of."""
        self.queue_events = [{}]
        self.graphql()

    def graphql(self) -> None:
        last = {"author": {"login": "someone", "__typename": "User"}}
        node = {"id": "t", "isResolved": False, "comments": {"nodes": [last]}}
        pull = {
            "author": {"login": "agent"},
            "reviewThreads": {"nodes": [node] * self.thread_count},
            "commits": {"nodes": [{"commit": {"committedDate": COMMITTED}}]},
            "timelineItems": {"nodes": self.queue_events},
        }
        self.gh.serve("api graphql", {"data": {"repository": {"pullRequest": pull}}})

    def comments(self, outside: bool = False) -> None:
        user = {"login": "sedosaur[bot]", "type": "Bot"}
        at = "2026-01-01T00:00:00Z"
        found = [{"user": user, "body": FINDINGS, "created_at": at, "updated_at": at}]
        self.gh.serve("api */comments*", found if outside else [])

    def run(self) -> list[str]:
        result = subprocess.run(
            [str(SCRIPT), "7"], env=self.env, capture_output=True, text=True, check=True
        )
        return result.stdout.splitlines()


@pytest.fixture
def gh(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestShepherdWork:
    """Lists the work a pull request needs before it is done."""

    def test_should_print_nothing_for_a_done_pull_request(self, gh: Harness) -> None:
        gh.runs("success")
        assert gh.run() == []

    def test_should_report_a_conflict(self, gh: Harness) -> None:
        gh.pull(state="dirty")
        assert gh.run() == ["conflict"]

    def test_should_report_no_conflict_while_github_computes_it(
        self, gh: Harness
    ) -> None:
        gh.pull(state="unknown")
        assert gh.run() == []

    def test_should_report_a_failed_ci_run_on_the_head(self, gh: Harness) -> None:
        gh.runs("failure")
        assert gh.run() == ["ci 1"]

    def test_should_ignore_a_ci_run_still_in_progress(self, gh: Harness) -> None:
        gh.runs("")
        assert gh.run() == []

    def test_should_report_a_failed_ok_job_in_a_run_in_progress(
        self, gh: Harness
    ) -> None:
        gh.runs(None)
        gh.ok("failure")
        assert gh.run() == ["ci 1"]

    def test_should_skip_the_ok_job_without_a_ci_run(self, gh: Harness) -> None:
        gh.pull(state="dirty")
        gh.runs()
        gh.ok("failure")
        assert gh.run() == ["conflict"]

    def test_should_report_each_open_codeql_alert(self, gh: Harness) -> None:
        gh.runs("success")
        gh.codeql(3, 9)
        assert gh.run() == [
            "codeql a.go:3 go/x Bad cast.",
            "codeql a.go:9 go/x Bad cast.",
        ]

    def test_should_report_a_queue_removal_after_the_head_commit(
        self, gh: Harness
    ) -> None:
        gh.runs("success", groups=("cancelled",))
        gh.removed(at="2026-01-01T13:00:00Z")
        assert gh.run() == ["queue 9"]
        span = "created=2026-01-01T12:00:00Z..2026-01-01T13:00:00Z"
        assert any(span in call[1] for call in gh.gh.calls())

    @pytest.mark.parametrize(
        ("severity", "security", "blocks"),
        [
            ("warning", "medium", True),
            ("warning", "critical", True),
            ("error", "low", False),
            ("error", None, True),
            ("warning", None, False),
        ],
    )
    def test_should_report_only_alerts_the_ruleset_blocks(
        self, gh: Harness, severity: str, security: str | None, blocks: bool
    ) -> None:
        gh.runs("success")
        gh.codeql(3, severity=severity, security=security)
        assert gh.run() == (["codeql a.go:3 go/x Bad cast."] if blocks else [])

    def test_should_ignore_alerts_while_the_codeql_check_passes(
        self, gh: Harness
    ) -> None:
        gh.runs("success")
        gh.codeql(3, check="success")
        assert gh.run() == []

    def test_should_wait_for_the_codeql_scan_of_the_head(self, gh: Harness) -> None:
        gh.runs(None, status="in_progress")
        gh.codeql(3)
        assert gh.run() == []

    def test_should_prefer_a_failed_merge_group_run(self, gh: Harness) -> None:
        gh.runs("success", groups=("cancelled", "failure"))
        gh.removed(at="2026-01-01T13:00:00Z")
        assert gh.run() == ["queue 10"]

    def test_should_report_a_queue_removal_without_a_merge_group_run(
        self, gh: Harness
    ) -> None:
        gh.runs("success")
        gh.removed(at="2026-01-01T13:00:00Z")
        assert gh.run() == ["queue"]

    def test_should_ignore_a_pull_request_enqueued_after_a_removal(
        self, gh: Harness
    ) -> None:
        gh.runs("success", groups=("failure",))
        gh.enqueued()
        assert gh.run() == []

    def test_should_ignore_a_queue_removal_before_the_head_commit(
        self, gh: Harness
    ) -> None:
        gh.runs("success", groups=("failure",))
        gh.removed(at="2026-01-01T11:00:00Z")
        assert gh.run() == []

    def test_should_ignore_a_queue_removal_for_another_reason(
        self, gh: Harness
    ) -> None:
        gh.runs("success", groups=("failure",))
        gh.removed(at="2026-01-01T13:00:00Z", reason="merged")
        assert gh.run() == []

    def test_should_report_threads_that_need_action(self, gh: Harness) -> None:
        gh.threads(count=2)
        assert gh.run() == ["threads 2"]

    def test_should_report_findings_outside_the_diff(self, gh: Harness) -> None:
        gh.comments(outside=True)
        assert gh.run() == ["outside-diff"]

    def test_should_report_every_kind_of_work_in_order(self, gh: Harness) -> None:
        gh.pull(state="dirty")
        gh.runs("failure", groups=("failure",))
        gh.codeql(3)
        gh.removed(at="2026-01-01T13:00:00Z")
        gh.threads(count=1)
        gh.comments(outside=True)
        assert gh.run() == [
            "conflict",
            "ci 1",
            "codeql a.go:3 go/x Bad cast.",
            "queue 9",
            "threads 1",
            "outside-diff",
        ]

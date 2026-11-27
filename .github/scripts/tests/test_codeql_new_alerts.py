#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from pathlib import Path
from typing import Any

import pytest

import codeql_new_alerts
from codeql_new_alerts import POLL, TIMEOUT, Alert, check, main, new_alerts

REPO = "acme/widgets"
MAIN = "refs/heads/main"
PULL = "refs/pull/7/merge"
SHA = "a" * 40
PARENT = "b" * 40
SCANNED = "c" * 40
FIX = "d" * 40
OLD = "e" * 40
GO = "/language:go"
PYTHON = "/language:python"
CATEGORIES = {GO, PYTHON}
URL = f"https://github.com/{REPO}/security/code-scanning/"

Raw = dict[str, Any]
Pages = list[list[Raw]]


def raw(
    number: int,
    state: str = "open",
    commit: str = SHA,
    category: str = GO,
    path: str = "pkg/parse.go",
    line: int = 10,
) -> Raw:
    """Returns an alert as the API lists it for one ref, with the fields the script
    reads."""
    return {
        "number": number,
        "state": state,
        "html_url": f"{URL}{number}",
        "rule": {
            "id": "go/incorrect-integer-conversion",
            "severity": "warning",
            "security_severity_level": "high",
        },
        "most_recent_instance": {
            "state": state,
            "commit_sha": commit,
            "category": category,
            "location": {"path": path, "start_line": line},
        },
    }


def analysis(category: str, commit: str) -> Raw:
    return {"category": category, "commit_sha": commit}


# The merge ref has an analysis of SHA in each category. Python is an empty analysis of
# a language the pull request leaves alone, so it opens no alert.
ANALYSES = [analysis(GO, SHA), analysis(PYTHON, SHA)]
# The newest analyses of main are of a commit that contains PARENT.
MAIN_ANALYSES = [analysis(GO, SCANNED), analysis(PYTHON, SCANNED)]
# Main has alert 1 open, alert 2 fixed in FIX, alert 3 dismissed, and an open Python
# alert 4 that the merge ref does not carry.
MAIN_ALERTS: Pages = [
    [raw(1, commit=SCANNED), raw(2, "fixed", FIX)],
    [raw(3, "dismissed", SCANNED), raw(4, commit=SCANNED, category=PYTHON)],
]


def alert(number: int, state: str = "open", commit: str = SHA) -> Alert:
    return Alert(
        number=number,
        rule="go/incorrect-integer-conversion",
        severity="high",
        path="pkg/parse.go",
        line=10,
        url=f"{URL}{number}",
        category=GO,
        state=state,
        commit=commit,
    )


class FakeAPI:
    """Answers the `gh api` calls of a check and records them. Each list of analyses
    of the checked ref or of main answers one poll, and the last answers every poll
    after it. compare maps "<base>...<head>" to the status of that comparison."""

    def __init__(
        self,
        pull: Pages,
        analyses: list[list[Raw]] | None = None,
        compare: dict[str, str] | None = None,
        main: list[list[Raw]] | None = None,
    ) -> None:
        self.alerts = {PULL: pull, MAIN: MAIN_ALERTS}
        self.analyses = analyses or [ANALYSES]
        self.main = main or [MAIN_ANALYSES]
        self.compare = {f"{PARENT}...{SCANNED}": "ahead", **(compare or {})}
        self.calls: list[tuple[str, ...]] = []

    def __call__(self, *args: str) -> str:
        self.calls.append(args)
        path = args[0]
        if "/code-scanning/analyses?" in path:
            polls = self.main if f"ref={MAIN}&" in path else self.analyses
            return json.dumps(polls.pop(0) if len(polls) > 1 else polls[0])
        if "/commits/" in path:
            return PARENT + "\n"
        if "/compare/" in path:
            return self.compare[path.split("/compare/")[1]] + "\n"
        ref = path.split("ref=")[1].split("&")[0]
        pages = self.alerts[ref]
        if "&state=open" in path:
            pages = [[a for a in page if a["state"] == "open"] for page in pages]
        return json.dumps(pages)


class Clock:
    """A clock that moves only when the check sleeps."""

    def __init__(self) -> None:
        self.now = 0.0
        self.sleeps: list[float] = []

    def __call__(self) -> float:
        return self.now

    def sleep(self, seconds: float) -> None:
        self.sleeps.append(seconds)
        self.now += seconds


def run(api: FakeAPI, clock: Clock | None = None) -> tuple[int, str]:
    clock = clock or Clock()
    return check(api, REPO, PULL, MAIN, SHA, CATEGORIES, clock=clock, sleep=clock.sleep)


def never(commit: str) -> bool:
    raise AssertionError(f"looked up {commit}")


class TestNewAlerts:
    """Tests the comparison of the alerts on a ref with the alerts on its base."""

    def test_should_keep_an_alert_that_is_not_on_the_base(self) -> None:
        assert new_alerts({1: alert(1)}, {}, never) == ([alert(1)], [])

    def test_should_drop_an_alert_open_on_the_base(self) -> None:
        assert new_alerts({1: alert(1)}, {1: alert(1)}, never) == ([], [])

    def test_should_match_alerts_by_number_not_by_line(self) -> None:
        moved = Alert(**{**alert(1).__dict__, "line": 90})
        assert new_alerts({1: moved}, {1: alert(1)}, never) == ([], [])

    def test_should_keep_an_alert_fixed_in_a_commit_the_ref_contains(self) -> None:
        base = {1: alert(1, "fixed", FIX)}
        assert new_alerts({1: alert(1)}, base, lambda c: c == FIX) == ([alert(1)], [])

    def test_should_mark_an_alert_fixed_after_the_ref_as_stale(self) -> None:
        base = {1: alert(1, "fixed", FIX)}
        assert new_alerts({1: alert(1)}, base, lambda _: False) == ([], [alert(1)])

    def test_should_keep_an_alert_dismissed_on_the_base(self) -> None:
        base = {1: alert(1, "dismissed")}
        assert new_alerts({1: alert(1)}, base, never) == ([alert(1)], [])

    def test_should_order_alerts_by_number(self) -> None:
        ref = {3: alert(3), 1: alert(1), 2: alert(2)}
        assert new_alerts(ref, {}, never)[0] == [alert(1), alert(2), alert(3)]


class TestCheck:
    """Tests a check against a fake API."""

    def test_should_pass_when_the_ref_carries_only_base_alerts(self) -> None:
        assert run(FakeAPI([[raw(1)]])) == (
            0,
            f"No CodeQL alert is open on `{PULL}` that is not open on `{MAIN}`.\n",
        )

    def test_should_fail_on_an_alert_on_an_unchanged_line(
        self, capsys: pytest.CaptureFixture[str]
    ) -> None:
        status, text = run(FakeAPI([[raw(1)], [raw(5, line=71)]]))
        assert status == 1
        assert capsys.readouterr().out == (
            "::error file=pkg/parse.go,line=71::#5 go/incorrect-integer-conversion "
            f"(high) pkg/parse.go:71 {URL}5\n"
        )
        assert text == (
            "## New CodeQL alerts\n\n"
            f"These alerts are open on `{PULL}` and not open on `{MAIN}`. Fix each "
            "one, also on lines the pull request did not change.\n\n"
            "| Alert | Rule | Severity | Location |\n| --- | --- | --- | --- |\n"
            f"| [#5]({URL}5) | `go/incorrect-integer-conversion` | high "
            "| `pkg/parse.go:71` |\n"
        )

    def test_should_fail_on_an_alert_fixed_in_a_commit_the_ref_contains(
        self,
    ) -> None:
        api = FakeAPI([[raw(2)]], compare={f"{FIX}...{SHA}": "ahead"})
        assert run(api)[0] == 1
        assert (f"repos/{REPO}/compare/{FIX}...{SHA}", "--jq", ".status") in api.calls

    def test_should_warn_on_an_alert_fixed_after_the_ref_was_built(
        self, capsys: pytest.CaptureFixture[str]
    ) -> None:
        api = FakeAPI([[raw(2)]], compare={f"{FIX}...{SHA}": "diverged"})
        status, text = run(api)
        assert status == 0
        assert capsys.readouterr().out == (
            "::warning file=pkg/parse.go,line=10::Stale: #2 "
            f"go/incorrect-integer-conversion (high) pkg/parse.go:10 {URL}2\n"
        )
        assert text.startswith(
            f"## Stale CodeQL alerts\n\n`{MAIN}` fixed these alerts after GitHub "
            f"built `{PULL}`. Merge `{MAIN}` to check them.\n\n"
        )

    def test_should_list_the_open_alerts_of_the_ref_and_all_of_the_base(self) -> None:
        api = FakeAPI([[]])
        run(api)
        assert [c for c in api.calls if "/alerts?" in c[0]] == [
            (
                f"repos/{REPO}/code-scanning/alerts?ref={PULL}&state=open"
                "&tool_name=CodeQL&per_page=100",
                "--paginate",
                "--slurp",
            ),
            (
                f"repos/{REPO}/code-scanning/alerts?ref={MAIN}&tool_name=CodeQL"
                "&per_page=100",
                "--paginate",
                "--slurp",
            ),
        ]

    def test_should_wait_for_an_analysis_of_each_category(self) -> None:
        api = FakeAPI([[]], analyses=[[analysis(PYTHON, SHA)], ANALYSES])
        clock = Clock()
        assert run(api, clock)[0] == 0
        assert clock.sleeps == [POLL]

    def test_should_not_count_an_analysis_of_an_older_commit(self) -> None:
        older = [analysis(GO, OLD), analysis(PYTHON, OLD)]
        api = FakeAPI([[]], analyses=[older, ANALYSES])
        clock = Clock()
        assert run(api, clock)[0] == 0
        assert clock.sleeps == [POLL]

    def test_should_wait_for_the_base_to_analyze_a_new_alert(self) -> None:
        lagging = [analysis(GO, OLD), analysis(PYTHON, SCANNED)]
        api = FakeAPI(
            [[raw(5)]],
            compare={f"{PARENT}...{OLD}": "behind"},
            main=[lagging, MAIN_ANALYSES],
        )
        clock = Clock()
        assert run(api, clock)[0] == 1
        assert clock.sleeps == [POLL]
        assert api.calls[0] == (
            f"repos/{REPO}/commits/{SHA}",
            "--jq",
            ".parents[0].sha",
        )

    def test_should_not_wait_for_the_base_without_a_new_alert(self) -> None:
        lagging = [analysis(GO, OLD), analysis(PYTHON, OLD)]
        api = FakeAPI([[raw(1)]], main=[lagging])
        clock = Clock()
        assert run(api, clock)[0] == 0
        assert clock.sleeps == []

    def test_should_fail_when_the_base_does_not_analyze_in_time(self) -> None:
        lagging = [analysis(GO, OLD), analysis(PYTHON, SCANNED)]
        api = FakeAPI(
            [[raw(5)]], compare={f"{PARENT}...{OLD}": "behind"}, main=[lagging]
        )
        assert run(api) == (
            1,
            f"No CodeQL analysis of {SHA} or its base after 3600 s for {GO} on "
            f"{MAIN}. Run the check again when the analyses finish.\n",
        )

    def test_should_fail_when_an_analysis_does_not_arrive_in_time(
        self, capsys: pytest.CaptureFixture[str]
    ) -> None:
        api = FakeAPI([[raw(5)]], analyses=[[analysis(PYTHON, SHA)]])
        clock = Clock()
        status, text = run(api, clock)
        assert status == 1
        assert clock.now == TIMEOUT
        message = (
            f"No CodeQL analysis of {SHA} or its base after 3600 s for {GO} on "
            f"{PULL}. Run the check again when the analyses finish."
        )
        assert text == f"{message}\n"
        assert capsys.readouterr().out == f"::error::{message}\n"
        assert all("/alerts?" not in c[0] for c in api.calls)


class TestMain:
    """Tests the command line."""

    def test_should_append_the_summary_and_exit_with_the_status(
        self, monkeypatch: pytest.MonkeyPatch, tmp_path: Path
    ) -> None:
        summary = tmp_path / "summary.md"
        summary.write_text("before\n")
        monkeypatch.setattr(codeql_new_alerts, "_gh_api", FakeAPI([[raw(5)]]))
        monkeypatch.setenv("GITHUB_REPOSITORY", REPO)
        monkeypatch.setenv("GITHUB_STEP_SUMMARY", str(summary))
        with pytest.raises(SystemExit) as exit:
            main(["codeql_new_alerts.py", PULL, MAIN, SHA, *sorted(CATEGORIES)])
        assert exit.value.code == 1
        assert summary.read_text().startswith("before\n## New CodeQL alerts\n")

    def test_should_print_the_shepherd_work_without_waiting(
        self, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
    ) -> None:
        api = FakeAPI([[raw(1), raw(5)]])
        monkeypatch.setattr(codeql_new_alerts, "_gh_api", api)
        monkeypatch.setenv("GITHUB_REPOSITORY", REPO)
        main(["codeql_new_alerts.py", "--work", PULL, MAIN, SHA])
        assert capsys.readouterr().out == (
            "codeql pkg/parse.go:10 go/incorrect-integer-conversion new alert "
            f"#5 {URL}5\n"
        )
        assert all("/analyses?" not in c[0] for c in api.calls)

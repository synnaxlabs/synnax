#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fails when a CodeQL analysis opens an alert that is not open on a base branch.

    python3 codeql_new_alerts.py <ref> <base> <sha> <category>...
    python3 codeql_new_alerts.py --work <ref> <base> <sha>

The CodeQL check fails a pull request only on alerts in the lines it changes. This
script compares all open CodeQL alerts on <ref>, such as refs/pull/<number>/merge, with
the open alerts on <base>, such as refs/heads/main. It matches alerts by number, because
GitHub gives an alert the same number on each ref. It first waits until <ref> has an
analysis of <sha> for each category. For each category with a new alert, it then waits
until the newest analysis of <base> is of a commit that contains the first parent of
<sha>, so a base that has not analyzed its own new alert yet does not fail <ref>. An
empty analysis of a language that a pull request does not change opens no alert, so it
cannot fail the check.

An alert that <base> fixed in a commit that <sha> does not contain is stale, not new:
the fix landed after GitHub built <sha>. The script warns about it and does not fail.
It prints each new alert, and writes them to the job summary when GITHUB_STEP_SUMMARY
is set. It exits with status 1 when there is a new alert, or when an analysis does not
arrive in time. With --work, it does not wait, and prints a line of shepherd work for
each new alert instead. Environment: GH_TOKEN and GITHUB_REPOSITORY.
"""

import json
import os
import subprocess
import sys
import time
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path
from typing import Any

# A push to main can take 60 minutes to analyze.
TIMEOUT = 3600.0
POLL = 30.0

Api = Callable[..., str]


@dataclass(frozen=True)
class Alert:
    number: int
    rule: str
    severity: str
    path: str
    line: int
    url: str
    category: str
    # The state on the listed ref, and the commit of its last analysis there.
    state: str
    commit: str

    def __str__(self) -> str:
        return (
            f"#{self.number} {self.rule} ({self.severity}) "
            f"{self.path}:{self.line} {self.url}"
        )


def _alert(raw: dict[str, Any]) -> Alert:
    rule = raw["rule"]
    instance = raw["most_recent_instance"]
    location = instance["location"]
    return Alert(
        number=raw["number"],
        rule=rule["id"],
        severity=rule.get("security_severity_level") or rule["severity"],
        path=location["path"],
        line=location["start_line"],
        url=raw["html_url"],
        category=instance["category"],
        state=instance["state"],
        commit=instance["commit_sha"],
    )


def alerts(api: Api, repo: str, ref: str, query: str = "") -> dict[int, Alert]:
    """Returns the CodeQL alerts on ref by number, filtered by the URL query."""
    pages = json.loads(
        api(
            f"repos/{repo}/code-scanning/alerts?ref={ref}{query}&tool_name=CodeQL"
            "&per_page=100",
            *("--paginate", "--slurp"),
        )
    )
    return {a.number: a for a in (_alert(raw) for page in pages for raw in page)}


def contains(api: Api, repo: str, sha: str, commit: str) -> bool:
    """Returns True if commit is sha or an ancestor of it."""
    status = api(f"repos/{repo}/compare/{commit}...{sha}", "--jq", ".status")
    return status.strip() in ("ahead", "identical")


def new_alerts(
    ref: dict[int, Alert], base: dict[int, Alert], fixed_in: Callable[[str], bool]
) -> tuple[list[Alert], list[Alert]]:
    """Returns the open alerts of ref that are not open on base, in order of number,
    split into new and stale ones. An alert is stale when base fixed it in a commit
    for which fixed_in returns False."""
    new: list[Alert] = []
    stale: list[Alert] = []
    for n in sorted(ref):
        b = base.get(n)
        if b is not None and b.state == "open":
            continue
        if b is not None and b.state == "fixed" and not fixed_in(b.commit):
            stale.append(ref[n])
        else:
            new.append(ref[n])
    return new, stale


def analyses(api: Api, repo: str, ref: str) -> list[dict[str, Any]]:
    """Returns the newest CodeQL analyses on ref, newest first."""
    # GitHub ignores a sha filter.
    out: list[dict[str, Any]] = json.loads(
        api(
            f"repos/{repo}/code-scanning/analyses?ref={ref}&tool_name=CodeQL"
            "&per_page=100"
        )
    )
    return out


def lagging(
    api: Api,
    repo: str,
    base: str,
    parent: str,
    categories: set[str],
    contained: Callable[[str, str], bool],
) -> list[str]:
    """Returns the categories whose newest analysis on base is of a commit that does
    not contain parent. contained(commit, head) returns True if head contains commit."""
    newest: dict[str, str] = {}
    for a in analyses(api, repo, base):
        newest.setdefault(a["category"], a["commit_sha"])
    return [
        c
        for c in sorted(categories)
        if c not in newest or not contained(parent, newest[c])
    ]


def _table(alerts: list[Alert]) -> str:
    return (
        "| Alert | Rule | Severity | Location |\n| --- | --- | --- | --- |\n"
        + "".join(
            f"| [#{a.number}]({a.url}) | `{a.rule}` | {a.severity} "
            f"| `{a.path}:{a.line}` |\n"
            for a in alerts
        )
    )


def summary(ref: str, base: str, new: list[Alert], stale: list[Alert]) -> str:
    """Returns the Markdown job summary for the new and stale alerts on ref."""
    text = ""
    if new:
        text += (
            "## New CodeQL alerts\n\n"
            f"These alerts are open on `{ref}` and not open on `{base}`. Fix each "
            "one, also on lines the pull request did not change.\n\n" + _table(new)
        )
    if stale:
        text += (
            "## Stale CodeQL alerts\n\n"
            f"`{base}` fixed these alerts after GitHub built `{ref}`. Merge `{base}` "
            "to check them.\n\n" + _table(stale)
        )
    return text or f"No CodeQL alert is open on `{ref}` that is not open on `{base}`.\n"


def compare(
    api: Api, repo: str, ref: str, base: str, sha: str
) -> tuple[list[Alert], list[Alert]]:
    """Returns the new and the stale alerts on ref, whose analyses are of sha."""
    return new_alerts(
        alerts(api, repo, ref, "&state=open"),
        alerts(api, repo, base),
        lambda commit: contains(api, repo, sha, commit),
    )


def work(api: Api, repo: str, ref: str, base: str, sha: str) -> list[str]:
    """Returns a line of shepherd work for each new alert on ref."""
    new, _ = compare(api, repo, ref, base, sha)
    return [
        f"codeql {a.path}:{a.line} {a.rule} new alert #{a.number} {a.url}" for a in new
    ]


def check(
    api: Api,
    repo: str,
    ref: str,
    base: str,
    sha: str,
    categories: set[str],
    *,
    clock: Callable[[], float] = time.monotonic,
    sleep: Callable[[float], None] = time.sleep,
) -> tuple[int, str]:
    """Returns the exit status and the job summary for ref. The api argument calls
    `gh api` with its arguments and returns the output."""
    deadline = clock() + TIMEOUT
    known: dict[tuple[str, str], bool] = {}

    def contained(commit: str, head: str) -> bool:
        if (commit, head) not in known:
            known[commit, head] = contains(api, repo, head, commit)
        return known[commit, head]

    parent = api(f"repos/{repo}/commits/{sha}", "--jq", ".parents[0].sha").strip()
    while True:
        analyzed = {
            a["category"] for a in analyses(api, repo, ref) if a["commit_sha"] == sha
        }
        waits = [f"{c} on {ref}" for c in sorted(categories - analyzed)]
        if not waits:
            new, stale = compare(api, repo, ref, base, sha)
            lag = lagging(api, repo, base, parent, {a.category for a in new}, contained)
            waits = [f"{c} on {base}" for c in lag]
        if not waits:
            break
        if clock() >= deadline:
            message = (
                f"No CodeQL analysis of {sha} or its base after {TIMEOUT:.0f} s for "
                f"{', '.join(waits)}. Run the check again when the analyses finish."
            )
            print(f"::error::{message}")
            return 1, f"{message}\n"
        sleep(POLL)
    for alert in new:
        print(f"::error file={alert.path},line={alert.line}::{alert}")
    for alert in stale:
        print(f"::warning file={alert.path},line={alert.line}::Stale: {alert}")
    if not new and not stale:
        print(f"No CodeQL alert is open on {ref} that is not open on {base}.")
    return (1 if new else 0), summary(ref, base, new, stale)


def _gh_api(*args: str) -> str:
    return subprocess.run(
        [str(Path(__file__).parent / "gh_api.sh"), *args],
        check=True,
        stdout=subprocess.PIPE,
        text=True,
    ).stdout


def main(argv: list[str]) -> None:
    repo = os.environ["GITHUB_REPOSITORY"]
    if argv[1] == "--work":
        for line in work(_gh_api, repo, *argv[2:5]):
            print(line)
        return
    ref, base, sha, *categories = argv[1:]
    status, text = check(_gh_api, repo, ref, base, sha, set(categories))
    if step_summary := os.environ.get("GITHUB_STEP_SUMMARY"):
        with Path(step_summary).open("a") as f:
            f.write(text)
    sys.exit(status)


if __name__ == "__main__":
    main(sys.argv)

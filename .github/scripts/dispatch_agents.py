#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Selects the agent-ready Linear issues an agent can start now.

An issue is ready when it is in Backlog or Todo, every blocker is Done, no open pull
request names it, and its project has fewer than the cap in flight. In flight means In
Progress or In Review, or named by an open pull request in any state. Without --dry-run,
moves each selected issue to In Progress and writes the agent workflow matrix to
--output.

Needs LINEAR_CLIENT_ID and LINEAR_CLIENT_SECRET for a Linear OAuth app with client
credentials on, and gh authenticated for the repository. Linear shows its changes as the
app.

Usage: dispatch_agents.py [--dry-run] [--cap N] [--output FILE]
"""

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.parse
import urllib.request
from collections import Counter
from collections.abc import Iterator
from contextlib import contextmanager
from dataclasses import dataclass
from typing import Any

TEAM = "SY"
LABEL = "agent-ready"
STARTED_STATE = "In Progress"
LINEAR_OAUTH = "https://api.linear.app/oauth"
# Linear revokes every app token when a request asks for other scopes.
LINEAR_SCOPE = "read,write"
_KEY = re.compile(r"\bSY-(\d+)\b", re.IGNORECASE)
_FIELD = re.compile(r"^[-*]?\s*(RFC|Phase):\s*`?([^`\s]+)`?\s*$", re.MULTILINE)

ISSUES_QUERY = """
query($after: String, $team: String!, $label: String!, $numbers: [Float!]!) {
  issues(first: 100, after: $after, filter: {
    team: {key: {eq: $team}},
    labels: {some: {name: {eq: $label}}},
    or: [
      {state: {type: {in: ["backlog", "unstarted", "started"]}}},
      {number: {in: $numbers}}
    ]
  }) {
    nodes {
      id identifier number title description
      state { type }
      project { name }
      inverseRelations(first: 250) {
        nodes { type issue { identifier state { type } } }
        pageInfo { hasNextPage }
      }
    }
    pageInfo { hasNextPage endCursor }
  }
}
"""

PROMPT = """Implement Linear issue {key}: {title}

This is phase {phase} of the RFC at `{rfc}`. Read the RFC and the AGENTS.md files
before you write code. Do only this phase, as one pull request into `main` that merges
on its own. Title it `{key}: {title}` and fill `.github/pull_request_template.md`. If
the phase cannot land as the RFC plans it, stop and explain why. Do not change the
plan."""


@dataclass(frozen=True, kw_only=True)
class Issue:
    id: str
    key: str
    number: int
    title: str
    description: str
    state: str
    project: str | None
    # None when the issue has more relations than one page holds.
    blocker_states: tuple[str, ...] | None


@dataclass(frozen=True, kw_only=True)
class Dispatch:
    issue: Issue
    rfc: str
    phase: str

    def matrix_entry(self) -> dict[str, str]:
        key, title = self.issue.key, self.issue.title
        return {
            "issue": key,
            "title": f"{key}: {title}",
            "prompt": PROMPT.format(
                key=key, title=title, phase=self.phase, rfc=self.rfc
            ),
        }


def parse_issue(node: dict[str, Any]) -> Issue:
    """Converts one Linear GraphQL issue node into an Issue."""
    relations = node["inverseRelations"]
    blockers = (
        None
        if relations["pageInfo"]["hasNextPage"]
        else tuple(
            r["issue"]["state"]["type"]
            for r in relations["nodes"]
            if r["type"] == "blocks"
        )
    )
    return Issue(
        id=node["id"],
        key=node["identifier"],
        number=node["number"],
        title=node["title"],
        description=node["description"] or "",
        state=node["state"]["type"],
        project=(node["project"] or {}).get("name"),
        blocker_states=blockers,
    )


def open_pr_keys(prs: list[dict[str, str]]) -> set[str]:
    """Returns the SY-#### keys named in the titles or branches of open PRs."""
    return {
        f"SY-{m.group(1)}"
        for pr in prs
        for text in (pr["title"], pr["headRefName"])
        for m in _KEY.finditer(text)
    }


def agent_fields(description: str) -> dict[str, str]:
    """Returns the RFC and Phase items of the `## Agent` section, if any."""
    sections = re.split(r"^## ", description, flags=re.MULTILINE)
    agent = next((s for s in sections if s.split("\n", 1)[0].strip() == "Agent"), "")
    return dict(_FIELD.findall(agent))


def select(
    issues: list[Issue], *, open_prs: set[str], cap: int
) -> tuple[list[Dispatch], list[str]]:
    """Returns the issues to dispatch, oldest first, and a reason for each skip."""
    in_flight = Counter(
        i.project for i in issues if i.state == "started" or i.key in open_prs
    )
    dispatches: list[Dispatch] = []
    skipped: list[str] = []
    for issue in sorted(issues, key=lambda i: i.number):
        if issue.state not in ("backlog", "unstarted") or issue.key in open_prs:
            continue
        fields = agent_fields(issue.description)
        if issue.project is None:
            skipped.append(f"{issue.key}: no project")
        elif issue.blocker_states is None:
            skipped.append(f"{issue.key}: too many relations to check its blockers")
        elif any(s != "completed" for s in issue.blocker_states):
            skipped.append(f"{issue.key}: blocked")
        elif not {"RFC", "Phase"} <= fields.keys():
            skipped.append(f"{issue.key}: no RFC and Phase under ## Agent")
        elif in_flight[issue.project] >= cap:
            skipped.append(f"{issue.key}: {issue.project} has {cap} in flight")
        else:
            in_flight[issue.project] += 1
            dispatches.append(
                Dispatch(issue=issue, rfc=fields["RFC"], phase=fields["Phase"])
            )
    return dispatches, skipped


@contextmanager
def linear_token() -> Iterator[str]:
    """Yields a short-lived Linear app token and revokes it on exit, so runs do not
    pile up toward Linear's limit of active tokens per app."""
    form = {
        "grant_type": "client_credentials",
        "client_id": os.environ["LINEAR_CLIENT_ID"],
        "client_secret": os.environ["LINEAR_CLIENT_SECRET"],
        "scope": LINEAR_SCOPE,
    }
    with urllib.request.urlopen(
        f"{LINEAR_OAUTH}/token", urllib.parse.urlencode(form).encode(), timeout=30
    ) as resp:
        token: str = json.load(resp)["access_token"]
    if os.environ.get("GITHUB_ACTIONS") == "true":
        print(f"::add-mask::{token}", flush=True)
    try:
        yield token
    finally:
        revoke = {"token": token, "token_type_hint": "access_token"}
        urllib.request.urlopen(
            f"{LINEAR_OAUTH}/revoke",
            urllib.parse.urlencode(revoke).encode(),
            timeout=30,
        ).close()


def _linear(token: str, query: str, variables: dict[str, Any]) -> dict[str, Any]:
    req = urllib.request.Request(
        "https://api.linear.app/graphql",
        data=json.dumps({"query": query, "variables": variables}).encode(),
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        body: dict[str, Any] = json.load(resp)
    if body.get("errors"):
        raise RuntimeError(f"Linear API error: {body['errors']}")
    data: dict[str, Any] = body["data"]
    return data


def fetch_issues(token: str, *, open_prs: set[str]) -> list[Issue]:
    """Returns the open agent-ready issues, plus those in any state that an open PR
    names, so they count toward their project's cap."""
    numbers = sorted(int(key.removeprefix("SY-")) for key in open_prs)
    issues: list[Issue] = []
    after: str | None = None
    while True:
        variables = {"after": after, "team": TEAM, "label": LABEL, "numbers": numbers}
        page = _linear(token, ISSUES_QUERY, variables)["issues"]
        issues.extend(parse_issue(n) for n in page["nodes"])
        if not page["pageInfo"]["hasNextPage"]:
            return issues
        after = page["pageInfo"]["endCursor"]


def fetch_open_prs() -> list[dict[str, str]]:
    out = subprocess.run(
        [
            *("gh", "pr", "list", "--state", "open", "--limit", "1000"),
            *("--json", "title,headRefName"),
        ],
        check=True,
        capture_output=True,
        text=True,
    ).stdout
    prs: list[dict[str, str]] = json.loads(out)
    return prs


def start(token: str, *, issues: list[Issue]) -> tuple[list[Issue], list[str]]:
    """Moves each issue to In Progress. Returns the moved issues and an error for
    each issue that did not move, so the moved ones still get their agents."""
    states = _linear(
        token,
        "query($team: String!) { teams(filter: {key: {eq: $team}}) "
        "{ nodes { states { nodes { id name } } } } }",
        {"team": TEAM},
    )["teams"]["nodes"][0]["states"]["nodes"]
    state_id = next(s["id"] for s in states if s["name"] == STARTED_STATE)
    moved: list[Issue] = []
    errors: list[str] = []
    for issue in issues:
        try:
            result = _linear(
                token,
                "mutation($id: String!, $state: String!) "
                "{ issueUpdate(id: $id, input: {stateId: $state}) { success } }",
                {"id": issue.id, "state": state_id},
            )
        except (OSError, RuntimeError) as e:
            errors.append(f"{issue.key}: {e}")
            continue
        if result["issueUpdate"]["success"]:
            moved.append(issue)
        else:
            errors.append(f"{issue.key}: Linear did not apply the update")
    return moved, errors


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--cap", type=int, default=3)
    parser.add_argument("--output")
    args = parser.parse_args()
    open_prs = open_pr_keys(fetch_open_prs())
    with linear_token() as token:
        dispatches, skipped = select(
            fetch_issues(token, open_prs=open_prs), open_prs=open_prs, cap=args.cap
        )
        for reason in skipped:
            print(f"skip {reason}", file=sys.stderr)
        for d in dispatches:
            verb = "would dispatch" if args.dry_run else "dispatch"
            print(f"{verb} {d.issue.key}: {d.issue.title} ({d.rfc} phase {d.phase})")
        if args.dry_run:
            return
        moved, errors = start(token, issues=[d.issue for d in dispatches])
        started = [d for d in dispatches if d.issue in moved]
        if args.output:
            matrix = {"include": [d.matrix_entry() for d in started]}
            with open(args.output, "a") as f:
                f.write(f"count={len(started)}\nmatrix={json.dumps(matrix)}\n")
        if errors:
            sys.exit("failed to move to In Progress:\n" + "\n".join(errors))


if __name__ == "__main__":
    main()

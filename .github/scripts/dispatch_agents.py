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
request names it, and its project has fewer than the cap in flight. In flight means
In Progress or In Review, or named by an open pull request. Without --dry-run, moves
each selected issue to In Progress and writes the agent workflow matrix to --output.

Needs LINEAR_API_KEY, and gh authenticated for the repository.

Usage: dispatch_agents.py [--dry-run] [--cap N] [--output FILE]
"""

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.request
from collections import Counter
from dataclasses import dataclass
from typing import Any

TEAM = "SY"
LABEL = "agent-ready"
STARTED_STATE = "In Progress"
_KEY = re.compile(r"\bSY-(\d+)\b", re.IGNORECASE)
_FIELD = re.compile(r"^[-*]?\s*(RFC|Phase|Tier):\s*`?([^`\s]+)`?\s*$", re.MULTILINE)

ISSUES_QUERY = """
query($after: String, $team: String!, $label: String!) {
  issues(first: 100, after: $after, filter: {
    team: {key: {eq: $team}},
    labels: {some: {name: {eq: $label}}},
    state: {type: {in: ["backlog", "unstarted", "started"]}}
  }) {
    nodes {
      id identifier number title description
      state { type }
      project { name }
      inverseRelations(first: 50) {
        nodes { type issue { identifier state { type } } }
      }
    }
    pageInfo { hasNextPage endCursor }
  }
}
"""

PROMPT = """Implement Linear issue {key}: {title}

This is phase {phase} of the RFC at `{rfc}`. Read the RFC and the CLAUDE.md files
before you write code. Do only this phase, as one pull request into `main` that merges
on its own. Title it `{key}: {title}` and fill `.github/pull_request_template.md`. If
the phase cannot land as the RFC plans it, stop and explain why. Do not change the
plan."""


@dataclass(frozen=True)
class Issue:
    id: str
    key: str
    number: int
    title: str
    description: str
    state: str
    project: str | None
    blocker_states: tuple[str, ...]


@dataclass(frozen=True)
class Dispatch:
    issue: Issue
    rfc: str
    phase: str
    tier: str

    def matrix_entry(self) -> dict[str, str]:
        key, title = self.issue.key, self.issue.title
        return {
            "issue": key,
            "title": f"{key}: {title}",
            "labels": self.tier,
            "prompt": PROMPT.format(
                key=key, title=title, phase=self.phase, rfc=self.rfc
            ),
        }


def parse_issue(node: dict[str, Any]) -> Issue:
    """Converts one Linear GraphQL issue node into an Issue."""
    blockers = tuple(
        r["issue"]["state"]["type"]
        for r in node["inverseRelations"]["nodes"]
        if r["type"] == "blocks"
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


def select(
    issues: list[Issue], open_prs: set[str], cap: int
) -> tuple[list[Dispatch], list[str]]:
    """Returns the issues to dispatch, oldest first, and a reason for each skip."""
    in_flight = Counter(
        i.project for i in issues if i.state == "started" or i.key in open_prs
    )
    dispatches: list[Dispatch] = []
    skipped: list[str] = []
    for issue in sorted(issues, key=lambda i: i.number):
        if issue.state == "started" or issue.key in open_prs:
            continue
        fields = dict(_FIELD.findall(issue.description))
        if issue.project is None:
            skipped.append(f"{issue.key}: no project")
        elif any(s != "completed" for s in issue.blocker_states):
            skipped.append(f"{issue.key}: blocked")
        elif not {"RFC", "Phase", "Tier"} <= fields.keys():
            skipped.append(f"{issue.key}: no RFC, Phase, and Tier under ## Agent")
        elif in_flight[issue.project] >= cap:
            skipped.append(f"{issue.key}: {issue.project} has {cap} in flight")
        else:
            in_flight[issue.project] += 1
            dispatches.append(
                Dispatch(issue, fields["RFC"], fields["Phase"], fields["Tier"])
            )
    return dispatches, skipped


def _linear(query: str, variables: dict[str, Any]) -> dict[str, Any]:
    req = urllib.request.Request(
        "https://api.linear.app/graphql",
        data=json.dumps({"query": query, "variables": variables}).encode(),
        headers={
            "Authorization": os.environ["LINEAR_API_KEY"],
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        body: dict[str, Any] = json.load(resp)
    if body.get("errors"):
        raise RuntimeError(f"Linear API error: {body['errors']}")
    data: dict[str, Any] = body["data"]
    return data


def fetch_issues() -> list[Issue]:
    issues: list[Issue] = []
    after: str | None = None
    while True:
        variables = {"after": after, "team": TEAM, "label": LABEL}
        page = _linear(ISSUES_QUERY, variables)["issues"]
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


def start(issues: list[Issue]) -> None:
    states = _linear(
        "query($team: String!) { teams(filter: {key: {eq: $team}}) "
        "{ nodes { states { nodes { id name } } } } }",
        {"team": TEAM},
    )["teams"]["nodes"][0]["states"]["nodes"]
    state_id = next(s["id"] for s in states if s["name"] == STARTED_STATE)
    for issue in issues:
        result = _linear(
            "mutation($id: String!, $state: String!) "
            "{ issueUpdate(id: $id, input: {stateId: $state}) { success } }",
            {"id": issue.id, "state": state_id},
        )
        if not result["issueUpdate"]["success"]:
            raise RuntimeError(f"failed to move {issue.key} to {STARTED_STATE}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--cap", type=int, default=3)
    parser.add_argument("--output")
    args = parser.parse_args()
    dispatches, skipped = select(
        fetch_issues(), open_pr_keys(fetch_open_prs()), args.cap
    )
    for reason in skipped:
        print(f"skip {reason}", file=sys.stderr)
    for d in dispatches:
        verb = "would dispatch" if args.dry_run else "dispatch"
        print(f"{verb} {d.issue.key}: {d.issue.title} ({d.rfc} phase {d.phase})")
    if args.dry_run:
        return
    start([d.issue for d in dispatches])
    if args.output:
        matrix = {"include": [d.matrix_entry() for d in dispatches]}
        with open(args.output, "a") as f:
            f.write(f"count={len(dispatches)}\nmatrix={json.dumps(matrix)}\n")


if __name__ == "__main__":
    main()

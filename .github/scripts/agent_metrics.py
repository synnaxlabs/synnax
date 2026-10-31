#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints a report on agent and human pull requests as Slack mrkdwn.

Usage: agent_metrics.py [--days 7] [--agents '["synnax-codex[bot]"]']

Reads GitHub through the gh CLI, which needs GH_TOKEN in CI.
"""

import argparse
import json
import math
import re
import subprocess
from collections.abc import Iterable
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta

REPO = "synnaxlabs/synnax"
AGENTS = '["synnax-claude-code[bot]", "synnax-codex[bot]"]'
# A revert lands days after the merge, so the revert rate looks further back.
REVERT_DAYS = 30
# GitHub search returns no more than this many results.
LIMIT = 1000


@dataclass(frozen=True, kw_only=True)
class PR:
    number: int
    author: str
    bot: bool
    labels: frozenset[str]
    created: datetime
    merged: datetime | None
    closed: datetime | None


@dataclass(frozen=True, kw_only=True)
class Run:
    event: str
    started: datetime
    finished: datetime


def parse_time(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def login(author: dict[str, str]) -> str:
    """Returns the app slug for an app login such as app/x or x[bot]."""
    return author["login"].removeprefix("app/").removesuffix("[bot]")


def parse_pr(raw: dict[str, object]) -> PR:
    author = raw["author"]
    assert isinstance(author, dict)
    labels = raw["labels"]
    assert isinstance(labels, list)
    merged, closed = raw.get("mergedAt"), raw.get("closedAt")
    return PR(
        number=int(str(raw["number"])),
        author=login(author),
        bot=bool(author.get("is_bot")),
        labels=frozenset(label["name"] for label in labels),
        created=parse_time(str(raw["createdAt"])),
        merged=parse_time(merged) if isinstance(merged, str) else None,
        closed=parse_time(closed) if isinstance(closed, str) else None,
    )


def group(pr: PR, *, agents: frozenset[str]) -> str:
    if pr.author in agents:
        return "agent"
    return "other bot" if pr.bot else "human"


def percentile(values: Iterable[float], *, q: float) -> float:
    """Returns the nearest-rank percentile, so the result is always a sample."""
    ordered = sorted(values)
    return ordered[max(math.ceil(q * len(ordered)) - 1, 0)]


def reverted(messages: Iterable[str]) -> set[int]:
    """Returns the PR numbers that revert commits reference. A squash merge ends
    its subject with its own PR number, which is left out."""
    numbers: set[int] = set()
    for message in messages:
        subject = message.split("\n", 1)[0]
        if not subject.startswith("Revert"):
            continue
        refs = {int(n) for n in re.findall(r"#(\d+)", message)}
        own = re.search(r"\(#(\d+)\)$", subject)
        numbers |= refs - ({int(own.group(1))} if own else set())
    return numbers


def tier(pr: PR) -> str:
    tiers = sorted(label for label in pr.labels if label.startswith("review/"))
    return tiers[0] if tiers else "no tier"


def spread(values: list[float], *, unit: str) -> str:
    median = percentile(values, q=0.5)
    p90 = percentile(values, q=0.9)
    return f"{median:.1f} {unit} / {p90:.1f} {unit}"


def report(
    *,
    now: datetime,
    days: int,
    agents: frozenset[str],
    prs: list[PR],
    runs: list[Run],
    reverts: set[int],
) -> str:
    since = now - timedelta(days=days)
    lines = [f"*Agent report, {since:%Y-%m-%d} to {now:%Y-%m-%d}*"]
    groups = ["agent", "human", "other bot"]
    opened = [group(p, agents=agents) for p in prs if p.created >= since]
    lines.append("Opened: " + ", ".join(f"{g} {opened.count(g)}" for g in groups))
    closed = [p for p in prs if p.closed is not None and p.closed >= since]
    for g in groups[:2]:
        mine = [p for p in closed if group(p, agents=agents) == g]
        merged = sum(p.merged is not None for p in mine)
        lines.append(
            f"{g.capitalize()} PRs: merged {merged}, closed {len(mine) - merged}"
        )
    hours: dict[str, list[float]] = {}
    for p in closed:
        if p.merged is not None:
            elapsed = (p.merged - p.created).total_seconds() / 3600
            hours.setdefault(tier(p), []).append(elapsed)
    lines.append("Time to merge (median / p90):")
    lines.extend(
        f"• `{t}` (n={len(v)}): {spread(v, unit='h')}" for t, v in sorted(hours.items())
    )
    revert_since = now - timedelta(days=REVERT_DAYS)
    bot = [
        p.number
        for p in prs
        if p.merged is not None and p.merged >= revert_since and tier(p) == "review/bot"
    ]
    hit = len(reverts.intersection(bot))
    rate = f"{hit / len(bot):.0%}" if bot else "n/a"
    lines.append(
        f"Reverted `review/bot` merges, last {REVERT_DAYS} days: "
        f"{hit} of {len(bot)} ({rate})"
    )
    minutes: dict[str, list[float]] = {}
    for r in runs:
        if r.started >= since:
            elapsed = (r.finished - r.started).total_seconds() / 60
            minutes.setdefault(r.event, []).append(elapsed)
    lines.append("Successful CI duration (median / p90):")
    lines.extend(
        f"• {e} (n={len(v)}): {spread(v, unit='min')}"
        for e, v in sorted(minutes.items())
    )
    return "\n".join(lines)


def gh(*args: str) -> list[str]:
    # gh's stderr goes to the log, so a failed call shows its cause.
    out = subprocess.run(["gh", *args], check=True, stdout=subprocess.PIPE, text=True)
    return out.stdout.splitlines()


def search_prs(query: str) -> list[PR]:
    out = gh(
        "pr", "list", "--repo", REPO, "--state", "all", "--limit", str(LIMIT),
        "--search", query,
        "--json", "number,author,labels,createdAt,mergedAt,closedAt",
    )  # fmt: skip
    raw = json.loads("".join(out))
    if len(raw) >= LIMIT:
        raise SystemExit(f"{query!r} hit the search limit of {LIMIT} PRs")
    return [parse_pr(r) for r in raw]


def fetch_prs(since: datetime, *, revert_since: datetime) -> list[PR]:
    queries = [
        f"created:>={since:%Y-%m-%d}",
        f"closed:>={since:%Y-%m-%d}",
        f"merged:>={revert_since:%Y-%m-%d} label:review/bot",
    ]
    return list({p.number: p for q in queries for p in search_prs(q)}.values())


def fetch_reverts(since: datetime) -> set[int]:
    out = gh(
        "api", "--paginate", f"repos/{REPO}/commits?sha=main&since={since:%FT%TZ}",
        "--jq", ".[].commit.message | @json",
    )  # fmt: skip
    return reverted(json.loads(line) for line in out)


def fetch_runs(since: datetime) -> list[Run]:
    out = gh(
        "api", "--paginate",
        f"repos/{REPO}/actions/workflows/ci.yaml/runs"
        f"?status=success&per_page=100&created=>={since:%Y-%m-%d}",
        "--jq", ".workflow_runs[] | [.event, .run_started_at, .updated_at] | @json",
    )  # fmt: skip
    runs = (json.loads(line) for line in out)
    return [
        Run(event=e, started=parse_time(s), finished=parse_time(f)) for e, s, f in runs
    ]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--days", type=int, default=7)
    parser.add_argument("--agents", default=AGENTS, help="JSON array of bot logins")
    args = parser.parse_args()
    now = datetime.now(UTC).replace(microsecond=0)
    since = now - timedelta(days=args.days)
    revert_since = now - timedelta(days=REVERT_DAYS)
    agents = frozenset(login({"login": a}) for a in json.loads(args.agents))
    prs = fetch_prs(since, revert_since=revert_since)
    print(
        report(
            now=now,
            days=args.days,
            agents=agents,
            prs=prs,
            runs=fetch_runs(since),
            reverts=fetch_reverts(revert_since),
        )
    )


if __name__ == "__main__":
    main()

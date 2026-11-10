#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import sys
from datetime import UTC, datetime, timedelta

import pytest

import agent_metrics
from agent_metrics import PR, Run, login, parse_pr, percentile, report, reverted

NOW = datetime(2026, 10, 26, tzinfo=UTC)
AGENTS = frozenset({"synnax-claude-code"})


def pr(
    number: int,
    *,
    author: str = "someone",
    bot: bool = False,
    labels: tuple[str, ...] = (),
    age_days: float = 1,
    merge_hours: float | None = None,
    closed: bool = False,
) -> PR:
    created = NOW - timedelta(days=age_days)
    merged = None if merge_hours is None else created + timedelta(hours=merge_hours)
    return PR(
        number=number,
        author=author,
        bot=bot,
        labels=frozenset(labels),
        created=created,
        merged=merged,
        closed=merged or (created + timedelta(hours=1) if closed else None),
    )


class TestLogin:
    def test_strips_app_prefixes_and_bot_suffixes(self) -> None:
        assert login({"login": "app/synnax-codex"}) == "synnax-codex"
        assert login({"login": "synnax-codex[bot]"}) == "synnax-codex"
        assert login({"login": "pjdotson"}) == "pjdotson"

    def test_parses_a_gh_pr_list_row(self) -> None:
        parsed = parse_pr(
            {
                "number": 7,
                "author": {"login": "app/dependabot", "is_bot": True},
                "labels": [{"name": "review/bot"}],
                "createdAt": "2026-10-20T00:00:00Z",
                "mergedAt": None,
                "closedAt": "2026-10-21T00:00:00Z",
            }
        )
        assert parsed.author == "dependabot"
        assert parsed.bot
        assert parsed.labels == {"review/bot"}
        assert parsed.merged is None
        assert parsed.closed == datetime(2026, 10, 21, tzinfo=UTC)


class TestPercentile:
    def test_returns_the_nearest_rank_sample(self) -> None:
        values = [float(v) for v in range(1, 11)]
        assert percentile(values, q=0.5) == 5
        assert percentile(values, q=0.9) == 9
        assert percentile([3.0], q=0.9) == 3


class TestReverted:
    def test_finds_the_reverted_pr_and_skips_the_revert_itself(self) -> None:
        messages = [
            'Revert "SY-1: Add a thing (#120)" (#130)\n\nThis reverts commit abc.',
            "Revert something\n\nReverts synnaxlabs/synnax#121",
            "Revert the broken change (#132)\n\nReverts synnaxlabs/synnax#123",
            "SY-2: Fix #122 in a normal commit (#131)",
        ]
        assert reverted(messages) == {120, 121, 123}


class TestReport:
    def test_counts_groups_tiers_reverts_and_ci(self) -> None:
        prs = [
            pr(
                1,
                author="synnax-claude-code",
                bot=True,
                labels=("review/bot",),
                merge_hours=2,
            ),
            pr(
                2,
                author="synnax-claude-code",
                bot=True,
                labels=("review/bot",),
                merge_hours=4,
            ),
            pr(3, author="synnax-claude-code", bot=True, closed=True),
            pr(4, labels=("review/human",), merge_hours=10),
            pr(5, author="dependabot", bot=True),
            pr(6, labels=("review/bot",), age_days=20, merge_hours=1),
        ]
        runs = [
            Run(event="pull_request", started=NOW - timedelta(hours=1), finished=NOW),
            Run(event="merge_group", started=NOW - timedelta(minutes=30), finished=NOW),
            Run(event="pull_request", started=NOW - timedelta(days=9), finished=NOW),
        ]
        out = report(
            now=NOW, days=7, agents=AGENTS, prs=prs, runs=runs, reverts={2, 99}
        )
        assert out.splitlines() == [
            "*Agent report, 2026-10-19 to 2026-10-26*",
            "Opened: agent 3, human 1, other bot 1",
            "Agent PRs: merged 2, closed 1",
            "Human PRs: merged 1, closed 0",
            "Time to merge (median / p90):",
            "• `review/bot` (n=2): 2.0 h / 4.0 h",
            "• `review/human` (n=1): 10.0 h / 10.0 h",
            "Reverted `review/bot` merges, last 30 days: 1 of 3 (33%)",
            "Successful CI duration (median / p90):",
            "• merge_group (n=1): 30.0 min / 30.0 min",
            "• pull_request (n=1): 60.0 min / 60.0 min",
        ]


class TestMain:
    def test_reports_the_prs_runs_and_reverts_that_gh_returns(
        self,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        """Should count a pull request that more than one search finds once."""
        created = datetime.now(UTC) - timedelta(days=1)
        row = {
            "number": 1,
            "author": {"login": "app/synnax-codex", "is_bot": True},
            "labels": [{"name": "review/bot"}],
            "createdAt": f"{created:%FT%TZ}",
            "mergedAt": f"{created + timedelta(hours=2):%FT%TZ}",
            "closedAt": f"{created + timedelta(hours=2):%FT%TZ}",
        }
        run = [
            "merge_group",
            f"{created:%FT%TZ}",
            f"{created + timedelta(minutes=6):%FT%TZ}",
        ]

        def gh(*args: str) -> list[str]:
            if args[0] == "pr":
                return [json.dumps([row])]
            if "--jq" in args and args[-1].startswith(".workflow_runs"):
                return [json.dumps(run)]
            return [json.dumps('Revert "Add x (#1)" (#2)')]

        monkeypatch.setattr(agent_metrics, "gh", gh)
        monkeypatch.setattr(sys, "argv", ["agent_metrics.py", "--days", "7"])
        agent_metrics.main()
        lines = capsys.readouterr().out.splitlines()
        assert lines[1:4] == [
            "Opened: agent 1, human 0, other bot 0",
            "Agent PRs: merged 1, closed 0",
            "Human PRs: merged 0, closed 0",
        ]
        assert "• `review/bot` (n=1): 2.0 h / 2.0 h" in lines
        assert "Reverted `review/bot` merges, last 30 days: 1 of 1 (100%)" in lines
        assert "• merge_group (n=1): 6.0 min / 6.0 min" in lines

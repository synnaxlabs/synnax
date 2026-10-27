#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from datetime import UTC, datetime, timedelta

from agent_metrics import PR, Run, login, parse_pr, percentile, report, reverted

NOW = datetime(2026, 10, 26, tzinfo=UTC)
AGENTS = frozenset({"synnax-claude-code"})


def pr(
    number: int,
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
        assert percentile(values, 0.5) == 5
        assert percentile(values, 0.9) == 9
        assert percentile([3.0], 0.9) == 3


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
            pr(1, "synnax-claude-code", True, ("review/bot",), merge_hours=2),
            pr(2, "synnax-claude-code", True, ("review/bot",), merge_hours=4),
            pr(3, "synnax-claude-code", True, closed=True),
            pr(4, labels=("review/human",), merge_hours=10),
            pr(5, "dependabot", True),
            pr(6, labels=("review/bot",), age_days=20, merge_hours=1),
        ]
        runs = [
            Run("pull_request", NOW - timedelta(hours=1), NOW),
            Run("merge_group", NOW - timedelta(minutes=30), NOW),
            Run("pull_request", NOW - timedelta(days=9), NOW),
        ]
        out = report(NOW, 7, AGENTS, prs, runs, {2, 99})
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

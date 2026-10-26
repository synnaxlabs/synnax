#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from dispatch_agents import Issue, agent_fields, open_pr_keys, parse_issue, select

AGENT = """## Why

Phase 2 of the RFC.

## Agent

- RFC: `docs/tech/rfc/0090-foo.md`
- Phase: 2
"""


def issue(
    number: int,
    state: str = "unstarted",
    project: str | None = "Foo",
    blockers: tuple[str, ...] | None = (),
    description: str = AGENT,
) -> Issue:
    return Issue(
        id=f"uuid-{number}",
        key=f"SY-{number}",
        number=number,
        title=f"Add part {number}",
        description=description,
        state=state,
        project=project,
        blocker_states=blockers,
    )


def keys(
    issues: list[Issue], open_prs: set[str] | None = None, cap: int = 3
) -> list[str]:
    dispatches, _ = select(issues, open_prs or set(), cap)
    return [d.issue.key for d in dispatches]


def node(more_relations: bool = False) -> dict[str, Any]:
    return {
        "id": "uuid",
        "identifier": "SY-7",
        "number": 7,
        "title": "Add the codec",
        "description": None,
        "state": {"type": "backlog"},
        "project": None,
        "inverseRelations": {
            "nodes": [
                {"type": "blocks", "issue": {"state": {"type": "completed"}}},
                {"type": "related", "issue": {"state": {"type": "started"}}},
            ],
            "pageInfo": {"hasNextPage": more_relations},
        },
    }


class TestParseIssue:
    """Tests for reading a Linear GraphQL issue node."""

    def test_should_keep_only_blocking_relations(self) -> None:
        """Should read blockers from inverse relations of type blocks."""
        assert parse_issue(node()) == Issue(
            "uuid", "SY-7", 7, "Add the codec", "", "backlog", None, ("completed",)
        )

    def test_should_mark_blockers_unknown_on_a_partial_page(self) -> None:
        """Should not judge blockers from one page of relations."""
        assert parse_issue(node(more_relations=True)).blocker_states is None


class TestAgentFields:
    """Tests for reading the agent block of an issue description."""

    def test_should_read_only_the_agent_section(self) -> None:
        """Should ignore RFC and Phase items under other headings."""
        description = "## Scope\n\n- RFC: `docs/a.md`\n- Phase: 9\n\n" + AGENT
        assert agent_fields(description) == {
            "RFC": "docs/tech/rfc/0090-foo.md",
            "Phase": "2",
        }

    def test_should_return_nothing_without_an_agent_section(self) -> None:
        """Should not read items that sit outside an agent section."""
        assert agent_fields("## Scope\n\n- RFC: `docs/a.md`\n- Phase: 9\n") == {}


class TestOpenPRKeys:
    """Tests for finding the issues that open pull requests name."""

    def test_should_read_titles_and_branches(self) -> None:
        """Should find keys in titles and lowercase branch names."""
        prs = [
            {"title": "SY-1: Add the codec", "headRefName": "x"},
            {"title": "[docs] Fix a typo", "headRefName": "sy-22-fix-typo"},
            {"title": "Bump deps", "headRefName": "dependabot/npm"},
        ]
        assert open_pr_keys(prs) == {"SY-1", "SY-22"}


class TestSelect:
    """Tests for choosing the issues an agent can start now."""

    def test_should_dispatch_unblocked_issues_oldest_first(self) -> None:
        """Should dispatch Backlog and Todo issues whose blockers are Done."""
        issues = [issue(3, blockers=("completed",)), issue(1, state="backlog")]
        assert keys(issues) == ["SY-1", "SY-3"]

    def test_should_wait_for_every_blocker_to_be_done(self) -> None:
        """Should skip an issue with a blocker that is open or canceled."""
        issues = [
            issue(1, blockers=("completed", "started")),
            issue(2, blockers=("canceled",)),
        ]
        dispatches, skipped = select(issues, set(), 3)
        assert dispatches == []
        assert skipped == ["SY-1: blocked", "SY-2: blocked"]

    def test_should_count_a_done_issue_with_an_open_pr(self) -> None:
        """Should count a Done issue toward the cap while an open PR names it."""
        issues = [issue(1, state="completed"), issue(2, state="started"), issue(3)]
        assert keys(issues, open_prs={"SY-1"}, cap=2) == []

    def test_should_not_count_or_dispatch_a_done_issue_without_a_pr(self) -> None:
        """Should neither dispatch a Done issue nor count it toward the cap."""
        issues = [issue(1, state="completed"), issue(2, state="started"), issue(3)]
        assert keys(issues, cap=2) == ["SY-3"]

    def test_should_skip_only_the_issue_with_unknown_blockers(self) -> None:
        """Should skip an issue whose blockers are unknown and dispatch the rest."""
        dispatches, skipped = select([issue(1, blockers=None), issue(2)], set(), 3)
        assert [d.issue.key for d in dispatches] == ["SY-2"]
        assert skipped == ["SY-1: too many relations to check its blockers"]

    def test_should_skip_issues_already_in_flight(self) -> None:
        """Should never dispatch a started issue or one an open PR names."""
        assert keys([issue(1, state="started"), issue(2)], open_prs={"SY-2"}) == []

    def test_should_cap_in_flight_issues_per_project(self) -> None:
        """Should count started issues and open PRs against the project cap."""
        issues = [
            issue(1, state="started"),
            issue(2),
            issue(3),
            issue(4),
            issue(5, project="Bar"),
        ]
        assert keys(issues, open_prs={"SY-2"}, cap=3) == ["SY-3", "SY-5"]

    def test_should_skip_issues_without_a_project(self) -> None:
        """Should report an issue with no project instead of dispatching it."""
        dispatches, skipped = select([issue(1, project=None)], set(), 3)
        assert dispatches == []
        assert skipped == ["SY-1: no project"]

    def test_should_skip_issues_without_an_agent_block(self) -> None:
        """Should report an issue whose description lacks the agent fields."""
        dispatches, skipped = select([issue(1, description="## Why\n")], set(), 3)
        assert dispatches == []
        assert skipped == ["SY-1: no RFC and Phase under ## Agent"]

    def test_should_build_the_agent_workflow_inputs(self) -> None:
        """Should name the issue and the RFC in the prompt and title the PR."""
        dispatches, _ = select([issue(1)], set(), 3)
        entry = dispatches[0].matrix_entry()
        assert entry["issue"] == "SY-1"
        assert entry["title"] == "SY-1: Add part 1"
        assert "Implement Linear issue SY-1: Add part 1" in entry["prompt"]
        assert "phase 2 of the RFC at `docs/tech/rfc/0090-foo.md`" in entry["prompt"]

    def test_should_accept_star_bullets(self) -> None:
        """Should read the agent fields when Linear stores bullets as stars."""
        description = AGENT.replace("- ", "* ")
        assert keys([issue(1, description=description)]) == ["SY-1"]

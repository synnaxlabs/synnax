#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io
import json
import sys
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

import pytest

import dispatch_agents
from dispatch_agents import (
    Issue,
    agent_fields,
    linear_token,
    open_pr_keys,
    parse_issue,
    select,
)

AGENT = """## Why

Phase 2 of the RFC.

## Agent

- RFC: `docs/tech/rfc/0090-foo.md`
- Phase: 2
"""


def issue(
    number: int,
    *,
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
    issues: list[Issue], *, open_prs: set[str] | None = None, cap: int = 3
) -> list[str]:
    dispatches, _ = select(issues, open_prs=open_prs or set(), cap=cap)
    return [d.issue.key for d in dispatches]


def node(*, more_relations: bool = False) -> dict[str, Any]:
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
            id="uuid",
            key="SY-7",
            number=7,
            title="Add the codec",
            description="",
            state="backlog",
            project=None,
            blocker_states=("completed",),
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
        dispatches, skipped = select(issues, open_prs=set(), cap=3)
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
        dispatches, skipped = select(
            [issue(1, blockers=None), issue(2)], open_prs=set(), cap=3
        )
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
        dispatches, skipped = select([issue(1, project=None)], open_prs=set(), cap=3)
        assert dispatches == []
        assert skipped == ["SY-1: no project"]

    def test_should_skip_issues_without_an_agent_block(self) -> None:
        """Should report an issue whose description lacks the agent fields."""
        dispatches, skipped = select(
            [issue(1, description="## Why\n")], open_prs=set(), cap=3
        )
        assert dispatches == []
        assert skipped == ["SY-1: no RFC and Phase under ## Agent"]

    def test_should_build_the_agent_workflow_inputs(self) -> None:
        """Should name the issue and the RFC in the prompt and title the PR."""
        dispatches, _ = select([issue(1)], open_prs=set(), cap=3)
        entry = dispatches[0].matrix_entry()
        assert entry["issue"] == "SY-1"
        assert entry["title"] == "SY-1: Add part 1"
        assert "Implement Linear issue SY-1: Add part 1" in entry["prompt"]
        assert "phase 2 of the RFC at `docs/tech/rfc/0090-foo.md`" in entry["prompt"]

    def test_should_accept_star_bullets(self) -> None:
        """Should read the agent fields when Linear stores bullets as stars."""
        description = AGENT.replace("- ", "* ")
        assert keys([issue(1, description=description)]) == ["SY-1"]


Request = tuple[str, dict[str, str]]


class TestLinearToken:
    @pytest.fixture
    def requests(self, monkeypatch: pytest.MonkeyPatch) -> list[Request]:
        sent: list[Request] = []

        def urlopen(url: str, data: bytes, timeout: float) -> io.BytesIO:
            sent.append((url, dict(urllib.parse.parse_qsl(data.decode()))))
            return io.BytesIO(b'{"access_token": "tok"}')

        monkeypatch.setattr(urllib.request, "urlopen", urlopen)
        monkeypatch.setenv("LINEAR_CLIENT_ID", "id")
        monkeypatch.setenv("LINEAR_CLIENT_SECRET", "secret")
        monkeypatch.delenv("GITHUB_ACTIONS", raising=False)
        return sent

    def test_should_request_a_client_credentials_token_and_revoke_it(
        self, requests: list[Request]
    ) -> None:
        with linear_token() as token:
            assert token == "tok"
            assert requests == [
                (
                    "https://api.linear.app/oauth/token",
                    {
                        "grant_type": "client_credentials",
                        "client_id": "id",
                        "client_secret": "secret",
                        "scope": "read,write",
                    },
                )
            ]
        assert requests[1] == (
            "https://api.linear.app/oauth/revoke",
            {"token": "tok", "token_type_hint": "access_token"},
        )

    def test_should_revoke_the_token_when_the_block_raises(
        self, requests: list[Request]
    ) -> None:
        with pytest.raises(RuntimeError), linear_token():
            raise RuntimeError
        assert requests[-1][0] == "https://api.linear.app/oauth/revoke"

    def test_should_mask_the_token_in_github_actions(
        self,
        requests: list[Request],
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        monkeypatch.setenv("GITHUB_ACTIONS", "true")
        with linear_token():
            pass
        assert capsys.readouterr().out == "::add-mask::tok\n"

    def test_should_print_nothing_outside_github_actions(
        self, requests: list[Request], capsys: pytest.CaptureFixture[str]
    ) -> None:
        with linear_token():
            pass
        assert capsys.readouterr().out == ""


class FakeLinear:
    """Serves the Linear OAuth and GraphQL requests the dispatcher sends."""

    def __init__(self) -> None:
        self.update_succeeds = True
        self.updated: list[str] = []

    def __call__(self, req: Any, *args: Any, **kwargs: Any) -> io.BytesIO:
        if isinstance(req, str):
            return io.BytesIO(b'{"access_token": "tok"}')
        body = json.loads(req.data)
        query, variables = body["query"], body["variables"]
        if "issueUpdate" in query:
            self.updated.append(variables["id"])
            data: dict[str, Any] = {"issueUpdate": {"success": self.update_succeeds}}
        elif "teams" in query:
            states = {"nodes": [{"id": "started", "name": "In Progress"}]}
            data = {"teams": {"nodes": [{"states": states}]}}
        else:
            nodes = [self.node(1), self.node(2)]
            page = {"hasNextPage": False, "endCursor": None}
            data = {"issues": {"nodes": nodes, "pageInfo": page}}
        return io.BytesIO(json.dumps({"data": data}).encode())

    @staticmethod
    def node(number: int) -> dict[str, Any]:
        return {
            "id": f"uuid-{number}",
            "identifier": f"SY-{number}",
            "number": number,
            "title": f"Add part {number}",
            "description": AGENT,
            "state": {"type": "unstarted"},
            "project": {"name": "Foo"},
            "inverseRelations": {"nodes": [], "pageInfo": {"hasNextPage": False}},
        }


class TestMain:
    """Tests a dispatch run end to end against a fake Linear and gh."""

    @pytest.fixture
    def linear(self, monkeypatch: pytest.MonkeyPatch) -> FakeLinear:
        fake = FakeLinear()
        monkeypatch.setattr(urllib.request, "urlopen", fake)
        monkeypatch.setattr(
            dispatch_agents,
            "fetch_open_prs",
            lambda: [{"title": "SY-2: Add part 2", "headRefName": "sy-2"}],
        )
        monkeypatch.setenv("LINEAR_CLIENT_ID", "id")
        monkeypatch.setenv("LINEAR_CLIENT_SECRET", "secret")
        monkeypatch.delenv("GITHUB_ACTIONS", raising=False)
        return fake

    def test_should_start_the_ready_issue_and_write_its_matrix(
        self, linear: FakeLinear, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        output = tmp_path / "output"
        argv = ["dispatch_agents.py", "--output", str(output)]
        monkeypatch.setattr(sys, "argv", argv)
        dispatch_agents.main()
        assert linear.updated == ["uuid-1"]
        count, matrix = output.read_text().splitlines()
        assert count == "count=1"
        include = json.loads(matrix.removeprefix("matrix="))["include"]
        assert [e["issue"] for e in include] == ["SY-1"]

    def test_should_move_nothing_on_a_dry_run(
        self,
        linear: FakeLinear,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        monkeypatch.setattr(sys, "argv", ["dispatch_agents.py", "--dry-run"])
        dispatch_agents.main()
        assert linear.updated == []
        assert capsys.readouterr().out == (
            "would dispatch SY-1: Add part 1 (docs/tech/rfc/0090-foo.md phase 2)\n"
        )

    def test_should_exit_when_linear_does_not_apply_an_update(
        self, linear: FakeLinear, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        linear.update_succeeds = False
        output = tmp_path / "output"
        argv = ["dispatch_agents.py", "--output", str(output)]
        monkeypatch.setattr(sys, "argv", argv)
        with pytest.raises(SystemExit, match="SY-1: Linear did not apply the update"):
            dispatch_agents.main()
        assert output.read_text().startswith("count=0\n")

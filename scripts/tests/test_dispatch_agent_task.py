#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from conftest import FakeGitHub

FUTURE = "2999-01-01T00:00:00Z"
PAST = "2000-01-01T00:00:00Z"


class TestDispatchAgentTask:
    """Tests for `dispatch_agent_task.sh`."""

    @pytest.mark.parametrize(
        "args",
        [
            [],
            ["SY-1; rm -rf /"],
            ["sy-1"],
            ["--agent", "gpt", "SY-1"],
            ["--force", "SY-1"],
        ],
    )
    def test_should_reject_bad_input(
        self, fake_github: FakeGitHub, args: list[str]
    ) -> None:
        result = fake_github.run(*args)
        assert result.returncode == 2
        assert fake_github.gh_calls() == []

    def test_should_print_payload_and_send_nothing_on_dry_run(
        self, fake_github: FakeGitHub
    ) -> None:
        result = fake_github.run("--dry-run", "--agent", "codex", "SY-42")
        assert result.returncode == 0
        assert "issue=SY-42" in result.stdout
        assert "agent=codex" in result.stdout
        assert "prompt=Work on Linear issue SY-42." in result.stdout
        assert fake_github.gh_calls() == []

    def test_should_send_dispatch_and_print_matching_run(
        self, fake_github: FakeGitHub
    ) -> None:
        fake_github.runs(
            ("agent-task SY-42", PAST, "old"),
            ("agent-task SY-420", FUTURE, "other-issue"),
            ("agent-task SY-42", FUTURE, "match"),
        )
        result = fake_github.run("SY-42")
        assert result.returncode == 0
        assert "Run: match" in result.stdout
        send = fake_github.gh_calls()[0]
        assert send[:4] == [
            "api",
            "repos/synnaxlabs/synnax/dispatches",
            "--method",
            "POST",
        ]
        assert "event_type=agent-task" in send
        assert "client_payload[issue]=SY-42" in send
        assert "client_payload[agent]=claude" in send
        assert "client_payload[output]=pr" in send

    def test_should_print_actions_link_when_no_run_matches(
        self, fake_github: FakeGitHub
    ) -> None:
        fake_github.runs(("agent-task SY-7", FUTURE, "other-issue"))
        result = fake_github.run("SY-42")
        assert result.returncode == 0
        assert "No run found yet" in result.stdout
        assert len(fake_github.gh_calls()) == 13

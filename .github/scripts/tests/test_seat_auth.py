#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from datetime import UTC, datetime
from typing import Any

from seat_auth import check

NOW = datetime(2026, 11, 15, 10, 17, tzinfo=UTC)


def instance(
    iid: str, seat: str, *, agent: str = "codex", state: str = "running", **tags: str
) -> dict[str, Any]:
    tags |= {"seat": seat, "agent": agent}
    return {
        "InstanceId": iid,
        "State": {"Name": state},
        "Tags": [{"Key": k.replace("_", "-"), "Value": v} for k, v in tags.items()],
    }


class FakeAws:
    """Answers the AWS CLI calls of the check from fixed instances and answers. Each
    command reports its invocations only on the second poll."""

    def __init__(
        self,
        instances: list[dict[str, Any]],
        answers: dict[str, tuple[str, ...]] | None = None,
        *,
        offline: tuple[str, ...] = (),
    ) -> None:
        self.instances = instances
        self.answers = answers or {}
        self.offline = offline
        self.calls: list[list[str]] = []
        self.commands: dict[str, list[str]] = {}
        self.polls: dict[str, int] = {}

    def __call__(self, args: list[str]) -> Any:
        self.calls.append(args)
        match args[:2]:
            case ["ec2", "describe-instances"]:
                return {"Reservations": [{"Instances": self.instances}]}
            case ["ssm", "describe-instance-information"]:
                ids = [i["InstanceId"] for i in self.instances]
                return {
                    "InstanceInformationList": [
                        {"InstanceId": iid} for iid in ids if iid not in self.offline
                    ]
                }
            case ["ssm", "send-command"]:
                command = f"command-{len(self.commands)}"
                ids = args[args.index("--instance-ids") + 1 :]
                self.commands[command] = ids
                return {"Command": {"CommandId": command}}
            case ["ssm", "list-command-invocations"]:
                command = args[args.index("--command-id") + 1]
                self.polls[command] = self.polls.get(command, 0) + 1
                if self.polls[command] == 1:
                    return {"CommandInvocations": []}
                return {
                    "CommandInvocations": [
                        {
                            "InstanceId": iid,
                            "Status": self.answers[iid][0],
                            "CommandPlugins": [
                                {"Output": out} for out in self.answers[iid][1:]
                            ],
                        }
                        for iid in self.commands[command]
                    ]
                }
            case ["ec2", "create-tags"]:
                return None
        raise AssertionError(f"unexpected call {args}")

    def tagged(self) -> list[list[str]]:
        return [c[2:] for c in self.calls if c[:2] == ["ec2", "create-tags"]]


def run(aws: FakeAws) -> tuple[bool, list[str]]:
    return check(aws, now=NOW, sleep=lambda _: None)


def test_signed_out_running_seat_fails_and_is_tagged() -> None:
    aws = FakeAws(
        [instance("i-1", "seat-1", auth="signed-in")],
        {"i-1": ("Success", "signed-out\n")},
    )
    ok, lines = run(aws)
    assert not ok
    assert lines == ["seat-1: signed-out as of 2026-11-15T10:17Z"]
    assert aws.tagged() == [
        [
            "--resources",
            "i-1",
            "--tags",
            "Key=auth,Value=signed-out",
            "Key=auth-checked,Value=2026-11-15T10:17Z",
        ]
    ]


def test_stopped_seat_reports_its_recorded_state() -> None:
    aws = FakeAws(
        [
            instance(
                "i-1",
                "seat-1",
                state="stopped",
                auth="signed-out",
                auth_checked="2026-11-14T08:17Z",
            )
        ]
    )
    ok, lines = run(aws)
    assert not ok
    assert lines == ["seat-1: signed-out as of 2026-11-14T08:17Z"]
    assert not any(c[:2] == ["ssm", "send-command"] for c in aws.calls)


def test_signed_in_seats_pass() -> None:
    aws = FakeAws(
        [
            instance("i-2", "review-1"),
            instance("i-1", "seat-1"),
            instance("i-3", "seat-3", state="stopped"),
        ],
        {"i-1": ("Success", "signed-in"), "i-2": ("Success", "signed-in")},
    )
    ok, lines = run(aws)
    assert ok
    assert lines == [
        "review-1: signed-in as of 2026-11-15T10:17Z",
        "seat-1: signed-in as of 2026-11-15T10:17Z",
        "seat-3: not checked yet",
    ]
    sends = [c for c in aws.calls if c[:2] == ["ssm", "send-command"]]
    assert len(sends) == 1
    assert "agent=codex" in sends[0]
    assert sends[0][-2:] == ["i-2", "i-1"]


def test_claude_seats_are_not_checked() -> None:
    aws = FakeAws(
        [instance("i-1", "claude-1", agent="claude"), instance("i-2", "codex-1")],
        {"i-2": ("Success", "signed-out")},
    )
    ok, lines = run(aws)
    assert not ok
    assert lines == ["codex-1: signed-out as of 2026-11-15T10:17Z"]
    sends = [c for c in aws.calls if c[:2] == ["ssm", "send-command"]]
    assert [c[c.index("--parameters") + 1] for c in sends] == ["agent=codex"]


def test_seat_without_an_answer_keeps_its_recorded_state() -> None:
    aws = FakeAws(
        [
            instance("i-1", "seat-1", auth="signed-in", auth_checked="earlier"),
            instance("i-2", "seat-2"),
        ],
        # An undelivered invocation has no plugins.
        {"i-1": ("Failed",), "i-2": ("Success", "")},
    )
    ok, lines = run(aws)
    assert ok
    assert lines == [
        "seat-1: running, but gave no answer",
        "seat-1: signed-in as of earlier",
        "seat-2: running, but gave no answer",
        "seat-2: not checked yet",
    ]
    assert aws.tagged() == []


def test_seat_with_ssm_offline_is_not_asked() -> None:
    aws = FakeAws(
        [instance("i-1", "seat-1"), instance("i-2", "seat-2")],
        {"i-2": ("Success", "signed-in")},
        offline=("i-1",),
    )
    ok, lines = run(aws)
    assert ok
    assert lines == [
        "seat-1: running, but gave no answer",
        "seat-1: not checked yet",
        "seat-2: signed-in as of 2026-11-15T10:17Z",
    ]
    sends = [c for c in aws.calls if c[:2] == ["ssm", "send-command"]]
    assert sends[0][-1:] == ["i-2"]
    asked = [c for c in aws.calls if c[1] == "describe-instance-information"]
    assert len(asked) == 1

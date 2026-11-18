#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fails when the agent CLI of an agent seat is signed out.

Claude seats run on a shared login from `seat.sh token`, so only Codex seats have a
CLI login to check. Asks each running Codex seat for its sign-in state through the
synnax-agent-seat-auth SSM document, and records the answer in the seat's auth and
auth-checked tags. A stopped seat is not started: its tags give the state from when it
last ran.

Usage: seat_auth.py. Needs AWS credentials and AWS_REGION.
"""

import json
import subprocess
import sys
import time
from collections.abc import Callable
from dataclasses import dataclass, replace
from datetime import UTC, datetime
from typing import Any

DOCUMENT = "synnax-agent-seat-auth"
ANSWERS = ("signed-in", "signed-out")
WAITING = ("Pending", "InProgress", "Delayed", "Cancelling")
POLL_SECONDS = 5

# Runs the AWS CLI with the given arguments and returns its decoded JSON output.
Aws = Callable[[list[str]], Any]


def cli(args: list[str]) -> Any:
    out = subprocess.run(
        ["aws", *args, "--output", "json"],
        check=True,
        stdout=subprocess.PIPE,
        text=True,
    ).stdout
    return json.loads(out) if out.strip() else None


@dataclass(frozen=True)
class Seat:
    id: str
    name: str
    agent: str
    running: bool
    auth: str | None
    checked: str | None


def seats(aws: Aws) -> list[Seat]:
    """The running and stopped Codex seats, by name."""
    found = aws(
        [
            "ec2",
            "describe-instances",
            "--filters",
            "Name=tag:project,Values=synnax-agents",
            "Name=tag-key,Values=seat",
            "Name=instance-state-name,Values=running,stopped",
        ]
    )
    result = []
    for reservation in found["Reservations"]:
        for i in reservation["Instances"]:
            tags = {t["Key"]: t["Value"] for t in i.get("Tags", [])}
            if tags["agent"] == "claude":
                continue
            result.append(
                Seat(
                    id=i["InstanceId"],
                    name=tags["seat"],
                    agent=tags["agent"],
                    running=i["State"]["Name"] == "running",
                    auth=tags.get("auth"),
                    checked=tags.get("auth-checked"),
                )
            )
    return sorted(result, key=lambda s: s.name)


def ask(
    aws: Aws, *, agent: str, ids: list[str], sleep: Callable[[float], None]
) -> dict[str, str]:
    """Each instance's answer, leaving out instances that gave none. SSM ends the
    command at the document's timeouts, so the wait ends."""
    command = aws(
        [
            "ssm",
            "send-command",
            "--document-name",
            DOCUMENT,
            "--parameters",
            f"agent={agent}",
            "--timeout-seconds",
            "60",
            "--instance-ids",
            *ids,
        ]
    )["Command"]["CommandId"]
    while True:
        invocations = aws(
            [
                "ssm",
                "list-command-invocations",
                "--command-id",
                command,
                "--details",
            ]
        )["CommandInvocations"]
        if len(invocations) == len(ids) and not any(
            i["Status"] in WAITING for i in invocations
        ):
            break
        sleep(POLL_SECONDS)
    answers = {}
    for i in invocations:
        # An undelivered invocation has no plugins.
        if i["Status"] != "Success":
            continue
        answer = i["CommandPlugins"][0]["Output"].strip()
        if answer in ANSWERS:
            answers[i["InstanceId"]] = answer
    return answers


def online(aws: Aws) -> set[str]:
    """The instances whose SSM agent is online. send-command rejects the whole call
    when it names any other instance, such as a seat that has just started."""
    found = aws(
        [
            "ssm",
            "describe-instance-information",
            "--filters",
            "Key=PingStatus,Values=Online",
        ]
    )
    return {i["InstanceId"] for i in found["InstanceInformationList"]}


def check(
    aws: Aws, *, now: datetime, sleep: Callable[[float], None] = time.sleep
) -> tuple[bool, list[str]]:
    """Returns whether no seat is signed out, and one line for each seat."""
    found = seats(aws)
    checked = f"{now:%Y-%m-%dT%H:%MZ}"
    up = online(aws)
    reachable = [s for s in found if s.running and s.id in up]
    answers: dict[str, str] = {}
    for agent in sorted({s.agent for s in reachable}):
        ids = [s.id for s in reachable if s.agent == agent]
        answers |= ask(aws, agent=agent, ids=ids, sleep=sleep)
    lines = []
    signed_out = False
    for seat in found:
        if seat.id in answers:
            seat = replace(seat, auth=answers[seat.id], checked=checked)
            aws(
                [
                    "ec2",
                    "create-tags",
                    "--resources",
                    seat.id,
                    "--tags",
                    f"Key=auth,Value={seat.auth}",
                    f"Key=auth-checked,Value={seat.checked}",
                ]
            )
        elif seat.running:
            lines.append(f"{seat.name}: running, but gave no answer")
        if seat.auth is None:
            lines.append(f"{seat.name}: not checked yet")
            continue
        signed_out |= seat.auth == "signed-out"
        lines.append(f"{seat.name}: {seat.auth} as of {seat.checked}")
    return not signed_out, lines


def main() -> None:
    ok, lines = check(cli, now=datetime.now(UTC))
    print("\n".join(lines))
    if not ok:
        sys.exit(1)


if __name__ == "__main__":
    main()

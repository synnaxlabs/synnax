#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import cast

import synnax as sy

from aws import AWS
from common import RECOMMAND_DELAY, Channels
from conftest import Client, core
from log import Log
from seats import Seats, seat_labels

JOB = {"self-hosted", "agent-claude"}
CLAUDE = "i-0aaa"
CLAUDE_2 = "i-0bbb"
CODEX = "i-0ccc"


def seat(state: str, *, name: str = "claude-1", pool: str = "agent") -> dict:
    agent = name.split("-")[0]
    tags = {"project": "synnax-agents", "pool": pool, "agent": agent, "seat": name}
    return {"state": state, "region": "us-east-1", "tags": tags}


class Instances:
    """Stands in for AWS, recording what is started. Starts fail while failing is
    set."""

    def __init__(self) -> None:
        self.started: list[str] = []
        self.failing = False
        self.ec2 = {"us-east-1": self}

    def start_instances(self, InstanceIds: list[str]) -> None:
        if self.failing:
            raise PermissionError("not authorized")
        self.started.extend(InstanceIds)


def seats(client: Client, aws: Instances) -> Seats:
    log = Log(
        client=core(client),
        channels=Channels(client=core(client), enabled=False),
        channel="log",
    )
    return Seats(aws=cast(AWS, aws), log=log)


class TestSeatLabels:
    """Tests for mapping instance tags to the labels of a seat's runners."""

    def test_should_map_an_agent_seat_to_its_runner_labels(self) -> None:
        labels = seat_labels(seat("stopped")["tags"])
        assert labels == {"agent", "agent-claude", "seat-claude-1"}

    def test_should_map_other_instances_to_no_labels(self) -> None:
        assert seat_labels(seat("stopped", pool="review")["tags"]) == set()
        assert seat_labels({"RunnerOS": "ubuntu"}) == set()


class TestSeats:
    """Tests for starting agent seats to meet queued jobs."""

    def test_should_start_a_stopped_seat_for_a_queued_job(self, client: Client) -> None:
        aws = Instances()
        seats(client, aws).step(queued=[JOB], found={CLAUDE: seat("stopped")})
        assert aws.started == [CLAUDE]
        assert client.written("log") == [["seat claude-1: start requested"]]

    def test_should_start_one_seat_for_several_queued_jobs(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("stopped"), CLAUDE_2: seat("stopped", name="claude-2")}
        seats(client, aws).step(queued=[JOB, {"self-hosted", "agent"}], found=found)
        assert aws.started == [CLAUDE]

    def test_should_start_only_the_seat_of_the_agent_a_job_asks_for(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("stopped"), CODEX: seat("stopped", name="codex-1")}
        seats(client, aws).step(queued=[{"self-hosted", "agent-codex"}], found=found)
        assert aws.started == [CODEX]

    def test_should_start_the_seat_a_job_names(self, client: Client) -> None:
        aws = Instances()
        found = {CLAUDE: seat("stopped"), CLAUDE_2: seat("stopped", name="claude-2")}
        seats(client, aws).step(
            queued=[{"self-hosted", "Linux", "seat-claude-2"}], found=found
        )
        assert aws.started == [CLAUDE_2]

    def test_should_not_start_a_seat_for_jobs_of_other_pools(
        self, client: Client
    ) -> None:
        aws = Instances()
        found = {CLAUDE: seat("stopped"), CODEX: seat("stopped", pool="review")}
        seats(client, aws).step(
            queued=[
                {"self-hosted", "ubuntu-build-bot"},
                {"self-hosted", "review"},
                {"self-hosted", "linux"},
                {"ubuntu-latest"},
            ],
            found=found,
        )
        assert aws.started == []

    def test_should_not_start_a_seat_while_one_is_up(self, client: Client) -> None:
        aws = Instances()
        s = seats(client, aws)
        for state in ("pending", "running"):
            found = {CLAUDE: seat(state), CLAUDE_2: seat("stopped", name="claude-2")}
            s.step(queued=[JOB], found=found)
        assert aws.started == []

    def test_should_start_a_stopping_seat_once_it_has_stopped(
        self, client: Client
    ) -> None:
        aws = Instances()
        s = seats(client, aws)
        s.step(queued=[JOB], found={CLAUDE: seat("stopping")})
        assert aws.started == []
        s.step(queued=[JOB], found={CLAUDE: seat("stopped")})
        assert aws.started == [CLAUDE]

    def test_should_wait_out_the_delay_before_it_starts_a_seat_again(
        self, client: Client
    ) -> None:
        aws = Instances()
        s = seats(client, aws)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found)
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]
        client.elapsed += RECOMMAND_DELAY
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE, CLAUDE]

    def test_should_log_a_failed_start_and_try_again_after_the_delay(
        self, client: Client
    ) -> None:
        aws = Instances()
        aws.failing = True
        s = seats(client, aws)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found)
        s.step(queued=[JOB], found=found)
        assert client.written("log") == [
            ["[ERROR] seat claude-1 start failed: PermissionError: not authorized"]
        ]
        aws.failing = False
        client.elapsed += RECOMMAND_DELAY + sy.TimeSpan.SECOND
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]

    def test_should_log_each_start_once_while_disabled(self, client: Client) -> None:
        aws = Instances()
        s = seats(client, aws)
        found = {CLAUDE: seat("stopped")}
        s.step(queued=[JOB], found=found, enabled=False)
        s.step(queued=[JOB], found=found, enabled=False)
        assert aws.started == []
        assert client.written("log") == [["[DISABLED] seat claude-1: start"]]
        s.step(queued=[JOB], found=found)
        assert aws.started == [CLAUDE]

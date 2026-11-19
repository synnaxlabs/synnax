#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Iterator

import pytest
import synnax as sy

from aws import AWS
from common import Channels
from conftest import Client, Clock, Server, github
from controller import FAILED_POLLS, Controller, Health, Toggle, wait_for_cmds
from cost import Cost
from gate import Gate
from heartbeat import Heartbeat
from jobs import Jobs
from log import Log
from registry import RunnerRegistry
from scaler import SCALER_CMD, Scaler
from subscription import Subscription
from test_jobs import Held, Responses, runs
from test_seats import CLAUDE, CLAUDE_2, Instances, seat, seats
from webhook import Webhook


class TestWaitForCmds:
    """Tests for reading the switches between polls."""

    def test_should_read_switches_when_no_time_remains(self, client: Client) -> None:
        channels = Channels(client=client, enabled=False)
        scaler = Scaler(
            client=client,
            cooldown=sy.TimeSpan(0),
            boot_grace=sy.TimeSpan(0),
            log=Log(client=client, channels=channels, channel="log"),
            gate=Gate(
                client=client,
                channels=channels,
                log=Log(client=client, channels=channels, channel="gh_log"),
            ),
            github=github(Server(Clock())),
        )
        beat = Heartbeat(
            client=client,
            channels=channels,
            name="enabled",
            value=lambda: scaler.enabled,
        )
        toggles: dict[str, Toggle] = {SCALER_CMD: (scaler, beat, "scaler")}
        client.streamed.append({SCALER_CMD: [False]})
        commands = Subscription(client=client, name="switches", channels=[SCALER_CMD])
        wait_for_cmds(commands, remaining=-sy.TimeSpan.SECOND, toggles=toggles)
        beat.stop()
        assert not scaler.enabled


class TestHealth:
    """Tests for the poll health behind the controller heartbeat."""

    def test_should_stay_healthy_below_the_failure_limit(self) -> None:
        health = Health()
        for _ in range(FAILED_POLLS - 1):
            health.record(failed=True)
        assert health.healthy()

    def test_should_turn_unhealthy_at_the_failure_limit(self) -> None:
        health = Health()
        for _ in range(FAILED_POLLS):
            health.record(failed=True)
        assert not health.healthy()

    def test_should_count_only_consecutive_failures(self) -> None:
        health = Health()
        for _ in range(FAILED_POLLS - 1):
            health.record(failed=True)
        health.record(failed=False)
        health.record(failed=True)
        assert health.healthy()

    def test_should_recover_after_a_successful_poll(self) -> None:
        health = Health()
        for _ in range(FAILED_POLLS):
            health.record(failed=True)
        health.record(failed=False)
        assert health.healthy()


@pytest.fixture
def webhook() -> Iterator[Webhook]:
    w = Webhook(secret="secret", port=0)
    yield w
    w.close()


class TestStartSeats:
    """Tests for starting seats from the jobs the controller holds."""

    def controller(
        self, client: Client, *, aws: Instances, webhook: Webhook, gate: bool
    ) -> Controller:
        channels = Channels(client=client, enabled=False)
        log = Log(client=client, channels=channels, channel="log")
        gh = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(7),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(),
                "/actions/runs/7/jobs?per_page=100&page=1": {
                    "jobs": [
                        {
                            "id": 1,
                            "status": "queued",
                            "labels": ["self-hosted", "agent-claude"],
                            "created_at": "2026-11-11T00:00:00Z",
                        }
                    ]
                },
            }
        )
        jobs = Jobs(github=gh, log=log, pool=Held())
        jobs.resync(sy.TimeStamp.now())
        g = Gate(client=client, channels=channels, log=log, enabled=gate)
        real = AWS()
        return Controller(
            client=client,
            channels=channels,
            github=gh,
            aws=real,
            runners=RunnerRegistry(),
            jobs=jobs,
            cost=Cost(log=log, aws=real),
            scaler=Scaler(
                client=client,
                cooldown=sy.TimeSpan(0),
                boot_grace=sy.TimeSpan(0),
                log=log,
                gate=g,
                github=github(Server(Clock())),
            ),
            seats=seats(client, aws),
            gate=g,
            log=log,
            webhook=webhook,
        )

    def test_should_start_another_seat_for_a_job_stuck_beside_a_busy_one(
        self, client: Client, webhook: Webhook
    ) -> None:
        aws = Instances()
        c = self.controller(client, aws=aws, webhook=webhook, gate=True)
        found = {CLAUDE: seat("running"), CLAUDE_2: seat("stopped", name="claude-2")}
        c.start_seats(found)
        assert aws.started == [CLAUDE_2]

    def test_should_hold_the_start_while_the_gate_is_closed(
        self, client: Client, webhook: Webhook
    ) -> None:
        aws = Instances()
        c = self.controller(client, aws=aws, webhook=webhook, gate=False)
        found = {CLAUDE: seat("running"), CLAUDE_2: seat("stopped", name="claude-2")}
        c.start_seats(found)
        assert aws.started == []

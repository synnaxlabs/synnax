#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy

from common import IDLE, OFFLINE, Channels, channel_name
from conftest import Client, Writer, core
from log import Log
from publisher import OPEN_LEEWAY
from scaler import Scaler, job_pool, pool_of

POOL = "ubuntu-build-bot"
LABELS = {"self-hosted", POOL}
A = "ubuntu-build-bot-i-0aaa"
B = "ubuntu-build-bot-i-0bbb"
LONG_AGO = sy.TimeStamp.now() - sy.TimeSpan.HOUR


def instance(state: str) -> dict:
    return {"state": state, "running": state == "running", "launched": LONG_AGO}


def scaler(client: Client) -> Scaler:
    log = Log(
        client=core(client),
        channels=Channels(client=core(client), enabled=False),
        channel="log",
    )
    return Scaler(
        client=core(client),
        cooldown=sy.TimeSpan(0),
        boot_grace=sy.TimeSpan.MINUTE * 2,
        log=log,
    )


def started(client: Client) -> list[str]:
    return [
        r
        for r in (A, B)
        for v in client.written(channel_name("ec2_cmd", runner=r))
        if v is True
    ]


class TestPools:
    """Tests for mapping runner labels to pools."""

    def test_should_map_runner_labels_to_a_pool(self) -> None:
        assert pool_of({"self-hosted", "Linux", POOL}) == POOL
        assert pool_of({"self-hosted", "Linux"}) is None

    def test_should_only_claim_jobs_for_self_hosted_pools(self) -> None:
        assert job_pool({POOL}) == POOL
        assert job_pool({"ubuntu-latest"}) is None


class TestScaler:
    """Tests for starting and stopping instances to meet demand."""

    def test_should_start_one_instance_per_queued_job(self, client: Client) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("stopped"), "i-0bbb": instance("stopped")}
        labels = {A: LABELS, B: LABELS}
        s.step(
            known={A, B},
            status={A: OFFLINE, B: OFFLINE},
            labels=labels,
            queued={POOL: 1},
            found=found,
            running=set(),
        )
        assert started(client) == [A]

    def test_should_count_a_commanded_instance_before_it_moves(
        self, client: Client
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("stopped"), "i-0bbb": instance("stopped")}
        labels = {A: LABELS, B: LABELS}
        status = {A: OFFLINE, B: OFFLINE}
        s.step(
            known={A, B},
            status=status,
            labels=labels,
            queued={POOL: 1},
            found=found,
            running=set(),
        )
        s.step(
            known={A, B},
            status=status,
            labels=labels,
            queued={POOL: 1},
            found=found,
            running=set(),
        )
        assert started(client) == [A]

    def test_should_count_booting_instances_toward_demand(self, client: Client) -> None:
        s = scaler(client)
        booting = instance("running") | {"launched": sy.TimeStamp.now()}
        found = {"i-0aaa": booting, "i-0bbb": instance("stopped")}
        labels = {A: LABELS, B: LABELS}
        s.step(
            known={A, B},
            status={A: OFFLINE, B: OFFLINE},
            labels=labels,
            queued={POOL: 1},
            found=found,
            running=set(),
        )
        assert started(client) == []

    def test_should_stop_idle_runners_once_the_queue_is_empty(
        self, client: Client
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step(
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={},
            found=found,
            running=set(),
        )
        assert client.written(channel_name("ec2_cmd", runner=A)) == [False]

    def test_should_keep_idle_runners_while_stops_are_held(
        self, client: Client
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step(
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={},
            found=found,
            running=set(),
            stops=False,
        )
        assert client.written(channel_name("ec2_cmd", runner=A)) == []

    def test_should_keep_idle_runners_while_their_pool_has_jobs(
        self, client: Client
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step(
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={POOL: 1},
            found=found,
            running=set(),
        )
        assert client.written(channel_name("ec2_cmd", runner=A)) == []

    def test_should_keep_runners_with_a_job_in_progress(self, client: Client) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step(
            known={A},
            status={A: OFFLINE},
            labels={A: LABELS},
            queued={},
            found=found,
            running={A},
        )
        assert client.written(channel_name("ec2_cmd", runner=A)) == []

    def test_should_clear_the_idle_timer_when_a_job_starts(
        self, client: Client
    ) -> None:
        log = Log(
            client=core(client),
            channels=Channels(client=core(client), enabled=False),
            channel="log",
        )
        s = Scaler(
            client=core(client),
            cooldown=sy.TimeSpan.HOUR,
            boot_grace=sy.TimeSpan.MINUTE * 2,
            log=log,
        )
        found = {"i-0aaa": instance("running")}
        s.step(
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={},
            found=found,
            running=set(),
        )
        assert A in s.idle_since
        s.step(
            known={A},
            status={A: OFFLINE},
            labels={A: LABELS},
            queued={},
            found=found,
            running={A},
        )
        assert A not in s.idle_since

    def test_should_stop_a_runner_once_its_job_completes(self, client: Client) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step(
            known={A},
            status={A: OFFLINE},
            labels={A: LABELS},
            queued={},
            found=found,
            running={A},
        )
        s.step(
            known={A},
            status={A: OFFLINE},
            labels={A: LABELS},
            queued={},
            found=found,
            running=set(),
        )
        assert client.written(channel_name("ec2_cmd", runner=A)) == [False]

    def test_should_start_an_instance_after_a_slow_writer_close(
        self, client: Client, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("stopped"), "i-0bbb": instance("stopped")}
        labels = {A: LABELS, B: LABELS}
        s.step(
            known={A},
            status={A: OFFLINE},
            labels=labels,
            queued={},
            found=found,
            running=set(),
        )

        def close(_: Writer) -> None:
            client.elapsed += 2 * OPEN_LEEWAY

        monkeypatch.setattr(Writer, "close", close)
        s.step(
            known={A, B},
            status={A: OFFLINE, B: OFFLINE},
            labels=labels,
            queued={POOL: 1},
            found=found,
            running=set(),
        )
        assert started(client) == [A]


WIN = "windows-build-bot-i-0aaa"
WIN_LABELS = {"self-hosted", "windows-build-bot"}
RUNNING = {"i-0aaa": instance("running")}


def windows_scaler(client: Client) -> Scaler:
    log = Log(
        client=core(client),
        channels=Channels(client=core(client), enabled=False),
        channel="log",
    )
    return Scaler(
        client=core(client),
        cooldown=sy.TimeSpan(0),
        boot_grace=sy.TimeSpan.MINUTE * 2,
        log=log,
        pool_cooldowns={"windows-build-bot": sy.TimeSpan.MINUTE * 15},
    )


class TestPoolCooldowns:
    """Tests for the idle cooldown a pool sets over the default."""

    def test_should_keep_a_runner_idle_less_than_its_pool_cooldown(
        self, client: Client
    ) -> None:
        s = windows_scaler(client)
        s.idle_since[WIN] = sy.TimeStamp.now() - sy.TimeSpan.MINUTE * 14
        s.step(
            known={WIN},
            status={WIN: IDLE},
            labels={WIN: WIN_LABELS},
            queued={},
            found=RUNNING,
            running=set(),
        )
        assert client.written(channel_name("ec2_cmd", runner=WIN)) == []

    def test_should_stop_a_runner_idle_past_its_pool_cooldown(
        self, client: Client
    ) -> None:
        s = windows_scaler(client)
        s.idle_since[WIN] = sy.TimeStamp.now() - sy.TimeSpan.MINUTE * 16
        s.step(
            known={WIN},
            status={WIN: IDLE},
            labels={WIN: WIN_LABELS},
            queued={},
            found=RUNNING,
            running=set(),
        )
        assert client.written(channel_name("ec2_cmd", runner=WIN)) == [False]

    def test_should_use_the_default_cooldown_for_other_pools(
        self, client: Client
    ) -> None:
        s = windows_scaler(client)
        s.step(
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={},
            found=RUNNING,
            running=set(),
        )
        assert client.written(channel_name("ec2_cmd", runner=A)) == [False]

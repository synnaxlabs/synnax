#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import pytest
import synnax as sy

from common import BUSY, IDLE, OFFLINE, RECOMMAND_DELAY, Channels, channel_name
from conftest import Client, Writer, core
from gate import Gate
from log import Log
from publisher import OPEN_LEEWAY
from scaler import Scaler

POOL = "ubuntu-build-bot"
LABELS = {"self-hosted", POOL}
A = "ubuntu-build-bot-i-0aaa"
B = "ubuntu-build-bot-i-0bbb"
LONG_AGO = sy.TimeStamp.now() - sy.TimeSpan.HOUR
NO_COOLDOWN = sy.TimeSpan(0)


def instance(state: str) -> dict:
    return {"state": state, "running": state == "running", "launched": LONG_AGO}


def scaler(client: Client, *, cooldown: sy.TimeSpan = NO_COOLDOWN) -> Scaler:
    channels = Channels(client=core(client), enabled=False)
    return Scaler(
        client=core(client),
        cooldown=cooldown,
        boot_grace=sy.TimeSpan.MINUTE * 2,
        log=Log(client=core(client), channels=channels, channel="log"),
        gate=Gate(
            client=core(client),
            channels=channels,
            log=Log(client=core(client), channels=channels, channel="gh_log"),
        ),
    )


def cycle(s: Scaler, **poll: Any) -> set[str]:
    """One controller poll: the gate steps, then the scaler."""
    s.gate.step(poll["known"])
    return s.step(**poll)


def runner_commands(client: Client, runner: str = A) -> list[object]:
    return client.written(channel_name("gh_cmd", runner=runner))


def instance_commands(client: Client, runner: str = A) -> list[object]:
    return client.written(channel_name("ec2_cmd", runner=runner))


def started(client: Client) -> list[str]:
    return [
        r
        for r in (A, B)
        for v in client.written(channel_name("ec2_cmd", runner=r))
        if v is True
    ]


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

    def test_should_return_the_booting_runners(self, client: Client) -> None:
        s = scaler(client)
        booting = instance("running") | {"launched": sy.TimeStamp.now()}
        found = {"i-0aaa": booting, "i-0bbb": instance("stopped")}
        assert s.step(
            known={A, B},
            status={A: OFFLINE, B: OFFLINE},
            labels={A: LABELS, B: LABELS},
            queued={},
            found=found,
            running=set(),
        ) == {A}

    def test_should_ignore_the_queue_of_an_unscaled_pool(self, client: Client) -> None:
        s = scaler(client)
        s.step(
            known={A},
            status={A: OFFLINE},
            labels={A: LABELS},
            queued={"macos-build-bot": 2},
            found={"i-0aaa": instance("stopped")},
            running=set(),
        )
        assert started(client) == []
        assert client.written("log") == []

    def test_should_drain_idle_runners_once_the_queue_is_empty(
        self, client: Client
    ) -> None:
        s = scaler(client)
        cycle(
            s,
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={},
            found={"i-0aaa": instance("running")},
            running=set(),
        )
        assert runner_commands(client) == [True, False]
        assert instance_commands(client) == []

    def test_should_stop_the_instance_of_a_drained_runner_once_it_is_offline(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A},
            "labels": {A: LABELS},
            "queued": {},
            "found": {"i-0aaa": instance("running")},
            "running": set(),
        }
        cycle(s, status={A: IDLE}, **poll)
        cycle(s, status={A: IDLE}, **poll)
        assert instance_commands(client) == []
        cycle(s, status={A: OFFLINE}, **poll)
        assert runner_commands(client) == [True, False]
        assert instance_commands(client) == [False]

    def test_should_keep_the_instance_of_a_runner_that_took_a_job_after_the_drain(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A},
            "labels": {A: LABELS},
            "queued": {},
            "found": {"i-0aaa": instance("running")},
        }
        cycle(s, status={A: IDLE}, running=set(), **poll)
        cycle(s, status={A: BUSY}, running=set(), **poll)
        cycle(s, status={A: BUSY}, running={A}, **poll)
        assert instance_commands(client) == []

    def test_should_drain_an_offline_runner_before_its_instance_stops(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A},
            "status": {A: OFFLINE},
            "labels": {A: LABELS},
            "queued": {},
            "found": {"i-0aaa": instance("running")},
            "running": set(),
        }
        cycle(s, **poll)
        assert instance_commands(client) == []
        cycle(s, **poll)
        assert runner_commands(client) == [True, False]
        assert instance_commands(client) == [False]

    def test_should_start_another_instance_for_a_job_queued_during_a_drain(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A, B},
            "status": {A: IDLE, B: OFFLINE},
            "labels": {A: LABELS, B: LABELS},
            "found": {"i-0aaa": instance("running"), "i-0bbb": instance("stopped")},
            "running": set(),
        }
        cycle(s, queued={}, **poll)
        cycle(s, queued={POOL: 1}, **poll)
        assert started(client) == [B]

    def test_should_stop_a_drained_runner_while_its_pool_has_jobs(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A},
            "labels": {A: LABELS},
            "found": {"i-0aaa": instance("running")},
            "running": set(),
        }
        cycle(s, status={A: IDLE}, queued={}, **poll)
        cycle(s, status={A: OFFLINE}, queued={POOL: 1}, **poll)
        assert instance_commands(client) == [False]

    def test_should_enable_a_drained_runner_once_its_instance_is_stopped(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A},
            "labels": {A: LABELS},
            "queued": {},
            "running": set(),
        }
        up = {"i-0aaa": instance("running")}
        cycle(s, status={A: IDLE}, found=up, **poll)
        cycle(s, status={A: OFFLINE}, found=up, **poll)
        assert runner_commands(client) == [True, False]
        cycle(s, status={A: OFFLINE}, found={"i-0aaa": instance("stopped")}, **poll)
        assert runner_commands(client) == [True, False, True]

    def test_should_leave_a_stopped_runner_drained_while_the_gate_is_closed(
        self, client: Client
    ) -> None:
        s = scaler(client)
        s.gate.enabled = False
        cycle(
            s,
            known={A},
            status={A: OFFLINE},
            labels={A: LABELS},
            queued={},
            found={"i-0aaa": instance("stopped")},
            running=set(),
        )
        assert runner_commands(client) == [False]

    def test_should_keep_idle_runners_of_an_unscaled_pool(self, client: Client) -> None:
        s = scaler(client)
        cycle(
            s,
            known={A},
            status={A: IDLE},
            labels={A: {"self-hosted", "macos-build-bot"}},
            queued={},
            found={"i-0aaa": instance("running")},
            running=set(),
        )
        assert runner_commands(client) == [True]

    def test_should_keep_idle_runners_while_stops_are_held(
        self, client: Client
    ) -> None:
        s = scaler(client)
        cycle(
            s,
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={},
            found={"i-0aaa": instance("running")},
            running=set(),
            stops=False,
        )
        assert runner_commands(client) == [True]

    def test_should_keep_idle_runners_while_their_pool_has_jobs(
        self, client: Client
    ) -> None:
        s = scaler(client)
        cycle(
            s,
            known={A},
            status={A: IDLE},
            labels={A: LABELS},
            queued={POOL: 1},
            found={"i-0aaa": instance("running")},
            running=set(),
        )
        assert runner_commands(client) == [True]

    def test_should_keep_runners_with_a_job_in_progress(self, client: Client) -> None:
        s = scaler(client)
        cycle(
            s,
            known={A},
            status={A: OFFLINE},
            labels={A: LABELS},
            queued={},
            found={"i-0aaa": instance("running")},
            running={A},
        )
        assert runner_commands(client) == [True]

    def test_should_clear_the_idle_timer_when_a_job_starts(
        self, client: Client
    ) -> None:
        s = scaler(client, cooldown=sy.TimeSpan.HOUR)
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

    def test_should_drain_again_a_runner_that_stays_online(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A},
            "status": {A: IDLE},
            "labels": {A: LABELS},
            "queued": {},
            "found": {"i-0aaa": instance("running")},
            "running": set(),
        }
        cycle(s, **poll)
        cycle(s, **poll)
        assert runner_commands(client) == [True, False]
        client.elapsed += 2 * RECOMMAND_DELAY
        cycle(s, **poll)
        assert runner_commands(client) == [True, False, False]
        assert instance_commands(client) == []

    def test_should_not_count_a_drained_runner_as_booting(self, client: Client) -> None:
        s = scaler(client)
        young = instance("running") | {"launched": sy.TimeStamp.now()}
        poll: dict[str, Any] = {
            "known": {A, B},
            "labels": {A: LABELS, B: LABELS},
            "found": {"i-0aaa": young, "i-0bbb": instance("stopped")},
            "running": set(),
        }
        cycle(s, status={A: IDLE, B: OFFLINE}, queued={}, **poll)
        cycle(s, status={A: OFFLINE, B: OFFLINE}, queued={POOL: 1}, **poll)
        assert started(client) == [B]
        assert instance_commands(client) == [False]

    def test_should_enable_a_runner_it_drained_when_disabled(
        self, client: Client
    ) -> None:
        s = scaler(client)
        poll: dict[str, Any] = {
            "known": {A},
            "labels": {A: LABELS},
            "queued": {},
            "found": {"i-0aaa": instance("running")},
        }
        cycle(s, status={A: IDLE}, running=set(), **poll)
        s.enabled = False
        cycle(s, status={A: BUSY}, running={A}, **poll)
        assert runner_commands(client) == [True, False, True]
        assert instance_commands(client) == []

    def test_should_keep_a_runner_drained_by_hand_while_disabled(
        self, client: Client
    ) -> None:
        s = scaler(client)
        s.enabled = False
        poll: dict[str, Any] = {
            "known": {A},
            "status": {A: OFFLINE},
            "labels": {A: LABELS},
            "queued": {},
            "found": {"i-0aaa": instance("stopped")},
            "running": set(),
        }
        cycle(s, **poll)
        client.streamed.append({channel_name("gh_cmd", runner=A): [False]})
        cycle(s, **poll)
        assert s.drained(A)
        assert runner_commands(client) == [True]

    def test_should_log_a_drain_once_while_disabled(self, client: Client) -> None:
        s = scaler(client)
        s.enabled = False
        poll: dict[str, Any] = {
            "known": {A},
            "status": {A: IDLE},
            "labels": {A: LABELS},
            "queued": {},
            "found": {"i-0aaa": instance("running")},
            "running": set(),
        }
        cycle(s, **poll)
        cycle(s, **poll)
        assert runner_commands(client) == [True]
        assert len(client.written("log")) == 1

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

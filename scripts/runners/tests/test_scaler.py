#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from common import IDLE, OFFLINE, Channels, channel_name
from conftest import Client, core
from log import Log
from scaler import Scaler, job_pool, pool_of

POOL = "ubuntu-build-bot"
LABELS = {"self-hosted", POOL}
A = "ubuntu-build-bot-i-0aaa"
B = "ubuntu-build-bot-i-0bbb"
LONG_AGO = sy.TimeStamp.now() - sy.TimeSpan.HOUR


def instance(state: str) -> dict:
    return {"state": state, "running": state == "running", "launched": LONG_AGO}


def scaler(client: Client) -> Scaler:
    log = Log(core(client), Channels(core(client), False), "log")
    return Scaler(core(client), sy.TimeSpan(0), sy.TimeSpan.MINUTE * 2, log)


def started(client: Client) -> list[str]:
    return [
        r
        for r in (A, B)
        for v in client.written(channel_name("ec2_cmd", r))
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
        s.step({A, B}, {A: OFFLINE, B: OFFLINE}, labels, {POOL: 1}, found, set())
        assert started(client) == [A]

    def test_should_count_a_commanded_instance_before_it_moves(
        self, client: Client
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("stopped"), "i-0bbb": instance("stopped")}
        labels = {A: LABELS, B: LABELS}
        status = {A: OFFLINE, B: OFFLINE}
        s.step({A, B}, status, labels, {POOL: 1}, found, set())
        s.step({A, B}, status, labels, {POOL: 1}, found, set())
        assert started(client) == [A]

    def test_should_count_booting_instances_toward_demand(self, client: Client) -> None:
        s = scaler(client)
        booting = instance("running") | {"launched": sy.TimeStamp.now()}
        found = {"i-0aaa": booting, "i-0bbb": instance("stopped")}
        labels = {A: LABELS, B: LABELS}
        s.step({A, B}, {A: OFFLINE, B: OFFLINE}, labels, {POOL: 1}, found, set())
        assert started(client) == []

    def test_should_stop_idle_runners_once_the_queue_is_empty(
        self, client: Client
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step({A}, {A: IDLE}, {A: LABELS}, {}, found, set())
        assert client.written(channel_name("ec2_cmd", A)) == [False]

    def test_should_keep_idle_runners_while_their_pool_has_jobs(
        self, client: Client
    ) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step({A}, {A: IDLE}, {A: LABELS}, {POOL: 1}, found, set())
        assert client.written(channel_name("ec2_cmd", A)) == []

    def test_should_keep_runners_with_a_job_in_progress(self, client: Client) -> None:
        s = scaler(client)
        found = {"i-0aaa": instance("running")}
        s.step({A}, {A: OFFLINE}, {A: LABELS}, {}, found, {A})
        assert client.written(channel_name("ec2_cmd", A)) == []

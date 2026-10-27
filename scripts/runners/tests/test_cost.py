#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math
from collections.abc import Iterator
from typing import cast

import pytest
import synnax as sy

from aws import AWS
from common import Channels, channel_name
from conftest import Client, core
from cost import FETCHED, LIVE_INTERVAL, Cost
from log import Log

RUNNER = "windows-build-bot-i-0aaa"
RATE = channel_name("cost_per_hour", RUNNER)
EBS = channel_name("ebs_cost_per_hour", RUNNER)


class NoAWS:
    """Stands in for AWS where a test makes no AWS call."""

    pricing = None


def found(running: bool) -> dict[str, dict]:
    return {"i-0aaa": {"running": running}}


@pytest.fixture
def cost(client: Client) -> Iterator[Cost]:
    log = Log(core(client), Channels(core(client), False), "log")
    c = Cost(log, cast(AWS, NoAWS()))
    now = sy.TimeStamp.now()
    # A fetch just ran, so step() only writes the live frame.
    c.attempted = now
    c.known = [RUNNER]
    c.latest = {RATE: 0.6, EBS: 0.1, FETCHED: int(now)}
    c.priced = {RUNNER: ("i-0aaa", 0.5, 0.1)}
    yield c
    c.close()


class TestLiveFrame:
    """Tests for the cost frame written between fetches."""

    def test_should_carry_every_cost_channel(self, cost: Cost) -> None:
        frame = cost.step(sy.TimeStamp.now(), {RUNNER}, found(True))
        assert frame.keys() == cost.channels().keys()

    def test_should_follow_the_instance_state(self, cost: Cost) -> None:
        assert cost.step(sy.TimeStamp.now(), {RUNNER}, found(False))[RATE] == 0.1

    def test_should_keep_the_last_fetch_time(self, cost: Cost) -> None:
        frame = cost.step(sy.TimeStamp.now(), {RUNNER}, found(True))
        assert frame[FETCHED] == cost.latest[FETCHED]

    def test_should_fill_channels_without_a_value_with_nan(self, cost: Cost) -> None:
        frame = cost.step(sy.TimeStamp.now(), {RUNNER}, found(True))
        assert math.isnan(frame[channel_name("cost_mtd_est", RUNNER)])

    def test_should_reprice_only_once_after_a_restart(self, cost: Cost) -> None:
        cost.priced = {}
        cost.step(sy.TimeStamp.now(), {RUNNER}, found(True))
        assert cost.pending is not None
        cost.pending.exception()
        cost.step(sy.TimeStamp.now(), {RUNNER}, found(True))
        assert cost.repriced
        assert cost.pending is None

    def test_should_write_once_per_live_interval(self, cost: Cost) -> None:
        now = sy.TimeStamp.now()
        assert cost.step(now, {RUNNER}, found(True))
        assert cost.step(now + LIVE_INTERVAL / 2, {RUNNER}, found(True)) == {}
        assert cost.step(now + LIVE_INTERVAL, {RUNNER}, found(True))

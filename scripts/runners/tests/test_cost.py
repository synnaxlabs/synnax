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
from cost import FETCHED, LIVE_INTERVAL, Cost, Prices
from log import Log

RUNNER = "windows-build-bot-i-0aaa"
OTHER = "windows-build-bot-i-0bbb"
RATE = channel_name("cost_per_hour", runner=RUNNER)
EBS = channel_name("ebs_cost_per_hour", runner=RUNNER)


class NoAWS:
    """Stands in for AWS where a test makes no AWS call."""

    pricing = None


class Volumes:
    """Stands in for AWS with no attached volumes."""

    pricing = None

    def volumes(self) -> dict[str, int]:
        return {}


class SomePrices:
    """Prices one instance type and fails on any other."""

    def instance(self, instance_type: str, *, region: str, windows: bool) -> float:
        if instance_type != "t3.large":
            raise LookupError(instance_type)
        return 0.5

    def volume(self, region: str) -> float:
        return 0.0


def found(*, running: bool) -> dict[str, dict]:
    return {"i-0aaa": {"running": running}}


@pytest.fixture
def cost(client: Client) -> Iterator[Cost]:
    log = Log(
        client=core(client),
        channels=Channels(client=core(client), enabled=False),
        channel="log",
    )
    c = Cost(log=log, aws=cast(AWS, NoAWS()))
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
        frame = cost.step(sy.TimeStamp.now(), known={RUNNER}, found=found(running=True))
        assert frame.keys() == cost.channels().keys()

    def test_should_follow_the_instance_state(self, cost: Cost) -> None:
        assert (
            cost.step(sy.TimeStamp.now(), known={RUNNER}, found=found(running=False))[
                RATE
            ]
            == 0.1
        )

    def test_should_keep_the_last_fetch_time(self, cost: Cost) -> None:
        frame = cost.step(sy.TimeStamp.now(), known={RUNNER}, found=found(running=True))
        assert frame[FETCHED] == cost.latest[FETCHED]

    def test_should_fill_channels_without_a_value_with_nan(self, cost: Cost) -> None:
        frame = cost.step(sy.TimeStamp.now(), known={RUNNER}, found=found(running=True))
        assert math.isnan(frame[channel_name("cost_mtd_est", runner=RUNNER)])

    def test_should_reprice_only_once_after_a_restart(self, cost: Cost) -> None:
        cost.priced = {}
        cost.step(sy.TimeStamp.now(), known={RUNNER}, found=found(running=True))
        assert cost.pending is not None
        cost.pending.exception()
        cost.step(sy.TimeStamp.now(), known={RUNNER}, found=found(running=True))
        assert cost.repriced
        assert cost.pending is None

    def test_should_write_once_per_live_interval(self, cost: Cost) -> None:
        now = sy.TimeStamp.now()
        assert cost.step(now, known={RUNNER}, found=found(running=True))
        assert (
            cost.step(
                now + LIVE_INTERVAL / 2, known={RUNNER}, found=found(running=True)
            )
            == {}
        )
        assert cost.step(now + LIVE_INTERVAL, known={RUNNER}, found=found(running=True))


class TestPriceTable:
    """Tests for the prices a restart looks up before the next fetch."""

    def test_should_keep_the_prices_that_resolve(self, client: Client) -> None:
        log = Log(
            client=core(client),
            channels=Channels(client=core(client), enabled=False),
            channel="log",
        )
        c = Cost(log=log, aws=cast(AWS, Volumes()))
        c.prices = cast(Prices, SomePrices())
        c.known = [RUNNER, OTHER]
        instances = {
            "i-0aaa": {"type": "t3.large", "region": "us-east-1", "windows": True},
            "i-0bbb": {"type": "c7i.4xlarge", "region": "us-east-1", "windows": True},
        }
        assert c.price_table(instances) == {RUNNER: ("i-0aaa", 0.5, 0.0)}
        c.close()

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
from common import Channels, channel_name
from conftest import Client, core
from cost import Cost
from log import Log

RUNNER = "windows-build-bot-i-0aaa"
RATE = channel_name("cost_per_hour", RUNNER)


class NoAWS:
    """Stands in for AWS where a test makes no AWS call."""

    pricing = None


def cost(client: Client) -> Cost:
    log = Log(core(client), Channels(core(client), False), "log")
    c = Cost(log, cast(AWS, NoAWS()))
    # Not due, so step() only reports rates.
    c.attempted = sy.TimeStamp.now()
    c.priced = {RUNNER: ("i-0aaa", 0.5, 0.1)}
    c.rates = {RUNNER: 0.6}
    return c


def found(running: bool) -> dict[str, dict]:
    return {"i-0aaa": {"running": running}}


class TestRates:
    """Tests for the runner rates between cost fetches."""

    def test_should_report_a_rate_when_its_instance_stops(self, client: Client) -> None:
        c = cost(client)
        assert c.step(sy.TimeStamp.now(), {RUNNER}, found(False)) == {RATE: 0.1}

    def test_should_report_nothing_while_the_state_holds(self, client: Client) -> None:
        c = cost(client)
        assert c.step(sy.TimeStamp.now(), {RUNNER}, found(True)) == {}
        c.close()

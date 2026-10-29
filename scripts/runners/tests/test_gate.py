#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

from common import Channels, channel_name
from conftest import Client, core
from gate import Gate
from log import Log
from publisher import OPEN_LEEWAY

RUNNER = "ubuntu-build-bot-i-0f44"
CHANNEL = channel_name("gh_cmd", runner=RUNNER)


def gate(client: Client) -> Gate:
    channels = Channels(client=core(client), enabled=False)
    return Gate(
        client=core(client),
        channels=channels,
        log=Log(client=core(client), channels=channels, channel="log"),
    )


class TestGate:
    """Tests for fanning the GitHub gate out to each runner."""

    def test_should_fan_out_once_per_change(self, client: Client) -> None:
        g = gate(client)
        g.step({RUNNER})
        g.step({RUNNER})
        g.enabled = False
        g.step({RUNNER})
        g.step({RUNNER})
        assert client.written(CHANNEL) == [True, False]

    def test_should_resend_a_fan_out_whose_write_failed(self, client: Client) -> None:
        g = gate(client)
        g.step({RUNNER})
        g.enabled = False
        client.failing = True
        g.step({RUNNER})
        client.failing = False
        g.step({RUNNER})
        assert client.written(CHANNEL) == [True, False]

    def test_should_publish_the_state_after_a_slow_step(self, client: Client) -> None:
        client.latency = 2 * OPEN_LEEWAY
        gate(client).step({RUNNER})
        assert client.written(channel_name("gh_state", runner=RUNNER)) == [True]

    def test_should_apply_the_start_state_over_saved_states(
        self, client: Client
    ) -> None:
        client.latest[channel_name("gh_state", runner=RUNNER)] = np.array([1], np.uint8)
        g = gate(client)
        g.enabled = False
        g.step({RUNNER})
        assert client.written(CHANNEL) == [False]

    def test_should_resend_nothing_saved_on_restart(self, client: Client) -> None:
        client.latest[channel_name("gh_state", runner=RUNNER)] = np.array([1], np.uint8)
        gate(client).step({RUNNER})
        assert client.written(CHANNEL) == []

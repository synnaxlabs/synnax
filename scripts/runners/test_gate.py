#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from common import Channels, channel_name
from conftest import Client, core
from gate import Gate
from log import Log

RUNNER = "ubuntu-build-bot-i-0f44"
CHANNEL = channel_name("gh_cmd", RUNNER)


def gate(client: Client) -> Gate:
    channels = Channels(core(client), False)
    return Gate(core(client), channels, Log(core(client), channels, "log"))


class TestGate:
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

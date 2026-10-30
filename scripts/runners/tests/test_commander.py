#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from commander import Commander
from common import Channels, channel_name
from conftest import Client, core
from log import Log

RUNNER = "ubuntu-build-bot-i-0f44"
CHANNEL = channel_name("ec2_cmd", runner=RUNNER)


def commander(client: Client) -> Commander:
    log = Log(
        client=core(client),
        channels=Channels(client=core(client), enabled=False),
        channel="log",
    )
    c = Commander(client=core(client), command="ec2_cmd", name="test", log=log)
    c.refresh_writer({RUNNER})
    return c


class TestCommander:
    """Tests for writing a command to a runner and logging it."""

    def test_should_write_and_record_a_command(self, client: Client) -> None:
        c = commander(client)
        now = sy.TimeStamp.now()
        assert c.command({RUNNER: True}, now=now)
        assert client.written(CHANNEL) == [True]
        assert c.commanded[RUNNER] == (now, True)
        assert c.ready({RUNNER}, now=now) == set()

    def test_should_record_nothing_when_the_write_fails(self, client: Client) -> None:
        c = commander(client)
        client.failing = True
        now = sy.TimeStamp.now()
        assert not c.command({RUNNER: True}, now=now)
        assert c.commanded == {}
        assert c.ready({RUNNER}, now=now) == {RUNNER}
        client.failing = False
        assert c.command({RUNNER: True}, now=now)
        assert client.written(CHANNEL) == [True]

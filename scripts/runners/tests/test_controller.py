#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from common import Channels
from conftest import Client, core
from controller import Toggle, wait_for_cmds
from heartbeat import Heartbeat
from log import Log
from scaler import SCALER_CMD, Scaler
from subscription import Subscription


class TestWaitForCmds:
    """Tests for reading the switches between polls."""

    def test_should_read_switches_when_no_time_remains(self, client: Client) -> None:
        channels = Channels(core(client), False)
        scaler = Scaler(
            core(client),
            sy.TimeSpan(0),
            sy.TimeSpan(0),
            Log(core(client), channels, "log"),
        )
        beat = Heartbeat(core(client), channels, "enabled", lambda: scaler.enabled)
        toggles: dict[str, Toggle] = {SCALER_CMD: (scaler, beat, "scaler")}
        client.streamed.append({SCALER_CMD: [False]})
        commands = Subscription(core(client), "switches", [SCALER_CMD])
        wait_for_cmds(commands, -sy.TimeSpan.SECOND, toggles)
        beat.stop()
        assert not scaler.enabled

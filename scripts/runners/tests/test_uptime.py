#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from common import IDLE, Channels, channel_name
from conftest import Client, core
from uptime import Uptime

OLD = "ubuntu-build-bot-i-0aaa"
NEW = "ubuntu-build-bot-i-0bbb"


class TestUptime:
    """Tests for the seconds each runner has spent in its state."""

    def test_should_cover_a_runner_added_before_its_first_update(
        self, client: Client
    ) -> None:
        u = Uptime(
            client=core(client), channels=Channels(client=core(client), enabled=False)
        )
        now = sy.TimeStamp.now()
        u.update({OLD: IDLE}, found={}, now=now - sy.TimeSpan.SECOND * 30)
        u.retarget({OLD, NEW})
        assert u.frame(now) == {
            channel_name("state_uptime", runner=OLD): 30,
            channel_name("state_uptime", runner=NEW): 0,
        }
        u.stop()

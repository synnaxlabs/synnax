#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import pytest


def create_valve_set(client: sy.Synnax) -> tuple[sy.Channel]:
    valve_en_time = client.channels.retrieve("valve_enable_time")
    valve_en_cmd_time = client.channels.retrieve("valve_enable_command_time")
    valve_en_cmd = client.channels.retrieve("valve_enable_command")
    valve_en = client.channels.retrieve("valve_enable")
    data = client.channels.retrieve("data")
    return valve_en_time, valve_en_cmd_time, valve_en_cmd, valve_en, data


@pytest.mark.control
@pytest.mark.focus
class TestController:
    def test_basic_control(self, client: sy.Synnax):
        with client.control.acquire(
            read=[valve_en.key, data.key],
            write=[valve_en_cmd.key, valve_en_cmd_time.key],
        ) as auto:
            while True:
                auto["valve_enable_command"] = True
                auto.wait_until(lambda c: c.data > 2000.0)
                auto["valve_enable_command"] = False
                auto.wait_until(lambda c: c.data < -2000.0)

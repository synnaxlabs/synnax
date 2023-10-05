#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
import time

import synnax as sy
import pytest


def create_valve_set(
    client: sy.Synnax,
) -> tuple[sy.Channel, sy.Channel, sy.Channel, sy.Channel]:
    press_end_cmd_time = client.channels.create(
        name="press_en_cmd_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
    )
    press_en_cmd = client.channels.create(
        name="press_en_cmd",
        data_type=sy.DataType.UINT8,
        index=press_end_cmd_time.key,
    )
    daq_time = client.channels.create(
        name="daq_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
    )
    press_en = client.channels.create(
        name="press_en",
        data_type=sy.DataType.UINT8,
        index=daq_time.key,
    )
    return press_end_cmd_time, press_en_cmd, press_en, daq_time


@pytest.mark.control
class TestController:
    def test_valve_toggle(self, client: sy.Synnax):
        press_end_cmd_time, press_en_cmd, press_en, daq_time = create_valve_set(client)

        def sequence():
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en.key],
                write=[press_en_cmd.key],
            ) as auto:
                time.sleep(0.05)
                auto[press_en_cmd.key] = True
                assert auto.wait_until(
                    lambda c: c[press_en.key],
                    timeout=2 * sy.TimeSpan.SECOND,
                )
                auto[press_en_cmd.key] = False
                assert auto.wait_until(
                    lambda c: not c[press_en.key],
                    timeout=2 * sy.TimeSpan.SECOND,
                )

        def daq():
            with client.control.acquire(
                name="Basic Valve Toggle",
                read=[press_en_cmd.key],
                write=[press_en.key],
            ) as auto:
                auto.wait_until(lambda c: c[press_en_cmd.key])
                auto[press_en.key] = True
                auto.wait_until(lambda c: not c[press_en_cmd.key])
                auto[press_en.key] = False

        t1 = threading.Thread(target=sequence)
        t2 = threading.Thread(target=daq)
        t2.start()
        t1.start()
        t1.join()
        t2.join()

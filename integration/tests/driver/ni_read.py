#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI analog read task integration tests."""

import synnax as sy

from tests.driver.ni_task import NIAnalogReadTaskCase
from tests.driver.task import create_channel, create_index


class NIAnalogReadHS(NIAnalogReadTaskCase):
    """Read "high speed" analog voltage from NI 9229."""

    # TODO: Create a task with a sample rate that is too low and verify status/error
    task_name = "NI Analog Voltage Read"
    device_name = "E101Mod1"  # NI 9229 

    SAMPLE_RATE = 10000 * sy.Rate.HZ  # Min sample rate for NI 9229: 1612.9 Hz
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, "ni_aiv_index")
        return [
            sy.ni.AIVoltageChan(
                port=i,
                channel=create_channel(
                    client,
                    name=f"ni_voltage_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                min_val=-10.0,
                max_val=10.0,
            )
            for i in range(2)
        ]

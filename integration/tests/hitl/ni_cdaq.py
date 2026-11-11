#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI task tests on the real cDAQ-9185 (SY2) of the HITL station.

Slot | Alias   | Module
-----|---------|------------------------------------------
1    | SY2Mod1 | NI 9375, 16 DI on port 0, 16 DO on port 1
2    | SY2Mod2 | NI 9211, 4 thermocouple inputs

The NI 9375 has no supply on Vsup, so its outputs do not drive its inputs. No test uses
this NI 9211 yet. The cRIO has a second NI 9211, and `crio_ni` tests the analog modules
there.
"""

import synnax as sy
from tests.driver.ni_task import NIDigitalReadTaskCase, NIDigitalWriteTaskCase
from tests.driver.task import create_channel, create_index


class CDAQDigitalRead(NIDigitalReadTaskCase):
    """Read two digital inputs of the NI 9375."""

    task_name = "HITL cDAQ Digital Read"
    device_locations = ["SY2Mod1"]
    simulated = False
    # Digital reads are software timed, and an Ethernet chassis answers each sample
    # over the network, so 1 kHz is out of reach.
    SAMPLE_RATE = 100 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.DIChannel]:
        idx = create_index(client, "hitl_cdaq_di_time")
        return [
            sy.ni.DIChannel(
                port=0,
                line=i,
                channel=create_channel(
                    client,
                    name=f"hitl_cdaq_di_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
            )
            for i in range(2)
        ]


class CDAQDigitalWrite(NIDigitalWriteTaskCase):
    """Write two digital outputs of the NI 9375."""

    task_name = "HITL cDAQ Digital Write"
    device_locations = ["SY2Mod1"]
    simulated = False
    command_values: list[list[float]] | None = [[1, 0], [0, 1]]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.DOChannel]:
        cmd_idx = create_index(client, "hitl_cdaq_do_cmd_time")
        state_idx = create_index(client, "hitl_cdaq_do_state_time")
        return [
            sy.ni.DOChannel(
                cmd_channel=create_channel(
                    client,
                    name=f"hitl_cdaq_do_cmd_{i}",
                    data_type=sy.DataType.UINT8,
                    index=cmd_idx.key,
                ),
                state_channel=create_channel(
                    client,
                    name=f"hitl_cdaq_do_state_{i}",
                    data_type=sy.DataType.UINT8,
                    index=state_idx.key,
                ),
                port=1,
                line=i,
            )
            for i in range(2)
        ]

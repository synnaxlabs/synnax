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
-----|---------|---------------------------------------
1    | SY2Mod1 | NI 9205, 32 analog inputs
2    | SY2Mod2 | NI 9211, 4 thermocouple inputs
3    | SY2Mod3 | NI 9263, 4 analog outputs
4    | SY2Mod4 | NI 9375, 16 DI on port 0, 16 DO on port 1
"""

import synnax as sy
from tests.driver.ni_task import (
    NIAnalogReadTaskCase,
    NIAnalogWriteTaskCase,
    NIDigitalReadTaskCase,
    NIDigitalWriteTaskCase,
)
from tests.driver.task import create_channel, create_index


class CDAQAnalogRead(NIAnalogReadTaskCase):
    """Read voltage from four inputs of the NI 9205."""

    task_name = "HITL cDAQ Analog Read"
    device_locations = ["SY2Mod1"]
    simulated = False
    SAMPLE_RATE = 1000 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIChannel]:
        idx = create_index(client, "hitl_cdaq_ai_time")
        return [
            sy.ni.AIVoltageChannel(
                port=i,
                channel=create_channel(
                    client,
                    name=f"hitl_cdaq_ai_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                min_val=-10.0,
                max_val=10.0,
            )
            for i in range(4)
        ]


class CDAQAnalogWrite(NIAnalogWriteTaskCase):
    """Write voltage to two outputs of the NI 9263."""

    task_name = "HITL cDAQ Analog Write"
    device_locations = ["SY2Mod3"]
    simulated = False
    # Low voltages, so an output wired to the station's inputs stays in their range.
    command_values: list[list[float]] | None = [[1, -1], [2, -2]]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AOVoltageChannel]:
        cmd_idx = create_index(client, "hitl_cdaq_ao_cmd_time")
        state_idx = create_index(client, "hitl_cdaq_ao_state_time")
        return [
            sy.ni.AOVoltageChannel(
                cmd_channel=create_channel(
                    client,
                    name=f"hitl_cdaq_ao_cmd_{i}",
                    data_type=sy.DataType.FLOAT64,
                    index=cmd_idx.key,
                ),
                state_channel=create_channel(
                    client,
                    name=f"hitl_cdaq_ao_state_{i}",
                    data_type=sy.DataType.FLOAT64,
                    index=state_idx.key,
                ),
                port=i,
                min_val=-10,
                max_val=10,
            )
            for i in range(2)
        ]


class CDAQDigitalRead(NIDigitalReadTaskCase):
    """Read two digital inputs of the NI 9375."""

    task_name = "HITL cDAQ Digital Read"
    device_locations = ["SY2Mod4"]
    simulated = False
    SAMPLE_RATE = 1000 * sy.Rate.HZ
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
    device_locations = ["SY2Mod4"]
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

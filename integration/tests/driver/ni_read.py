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
    device_locations = ["E101Mod1"]  # NI 9229

    SAMPLE_RATE = 10000 * sy.Rate.HZ  # Min sample rate for NI 9229: 1612.9 Hz
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
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


class NIReadRTD(NIAnalogReadTaskCase):
    """Read RTD sensors across two NI 9219 modules — all 7 RTD types, all 3 wire configs.

    E101Mod7:
        Port 0: Pt3750, 4-wire, DegC, PT100
        Port 1: Pt3851, 3-wire, DegF, PT100
        Port 2: Pt3911, 2-wire, DegC, PT100
        Port 3: Pt3916, 4-wire, Kelvins, PT1000
    E101Mod8:
        Port 0: Pt3920, 3-wire, DegR, PT100
        Port 1: Pt3928, 2-wire, DegC, PT100
        Port 2: Pt3850, 4-wire, DegF, PT1000
    """

    task_name = "NI RTD Read"
    device_locations = ["E101Mod7", "E101Mod8"]  # NI 9219

    SAMPLE_RATE = 20 * sy.Rate.HZ
    STREAM_RATE = 5 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIRTDChan]:
        idx = create_index(client, "ni_rtd_index")
        mod8 = devices["E101Mod8"]
        return [
            # --- E101Mod7 (ports 0-3) — device inherited from task ---
            sy.ni.AIRTDChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3750_4w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                rtd_type="Pt3750",
                resistance_config="4Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=100.0,
                min_val=-50.0,
                max_val=200.0,
            ),
            sy.ni.AIRTDChan(
                port=1,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3851_3w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegF",
                rtd_type="Pt3851",
                resistance_config="3Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=100.0,
                min_val=-58.0,
                max_val=392.0,
            ),
            sy.ni.AIRTDChan(
                port=2,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3911_2w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                rtd_type="Pt3911",
                resistance_config="2Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=100.0,
                min_val=-50.0,
                max_val=200.0,
            ),
            sy.ni.AIRTDChan(
                port=3,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3916_4w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="Kelvins",
                rtd_type="Pt3916",
                resistance_config="4Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=1000.0,
                min_val=223.0,
                max_val=473.0,
            ),
            # --- E101Mod8 (ports 0-2) — device set per channel ---
            sy.ni.AIRTDChan(
                device=mod8.key,
                port=0,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3920_3w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegR",
                rtd_type="Pt3920",
                resistance_config="3Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=100.0,
                min_val=400.0,
                max_val=850.0,
            ),
            sy.ni.AIRTDChan(
                device=mod8.key,
                port=1,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3928_2w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                rtd_type="Pt3928",
                resistance_config="2Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=100.0,
                min_val=-50.0,
                max_val=200.0,
            ),
            sy.ni.AIRTDChan(
                device=mod8.key,
                port=2,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3850_4w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegF",
                rtd_type="Pt3850",
                resistance_config="4Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=1000.0,
                min_val=-58.0,
                max_val=392.0,
            ),
        ]


class NIReadResistance(NIAnalogReadTaskCase):
    """Read resistance on NI 9219 with varied wire configurations.

    Port 0: 4-wire, 0-1 kOhm, 1 mA excitation
    Port 1: 2-wire, 0-10 kOhm, 100 uA excitation
    Port 2: 3-wire, 0-1 kOhm, 1 mA excitation
    """

    task_name = "NI Resistance Read"
    device_locations = ["E101Mod5"]  # NI 9219

    SAMPLE_RATE = 100 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIResistanceChan]:
        idx = create_index(client, "ni_resistance_index")
        return [
            sy.ni.AIResistanceChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_res_4w_1k",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                resistance_config="4Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                min_val=0.0,
                max_val=1000.0,
            ),
            sy.ni.AIResistanceChan(
                port=1,
                channel=create_channel(
                    client,
                    name="ni_res_2w_10k",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                resistance_config="2Wire",
                current_excit_source="Internal",
                current_excit_val=0.0001,
                min_val=0.0,
                max_val=10000.0,
            ),
            sy.ni.AIResistanceChan(
                port=2,
                channel=create_channel(
                    client,
                    name="ni_res_3w_1k",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                resistance_config="3Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                min_val=0.0,
                max_val=1000.0,
            ),
        ]

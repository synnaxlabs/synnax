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
    """Read RTD sensors across two NI 9219 modules — all 7 RTD types, 3/4-wire configs.

    NI 9219 only supports 3-wire and 4-wire RTD configurations.

    E101Mod7:
        Port 0: Pt3750, 4-wire, DegC
        Port 1: Pt3851, 3-wire, DegF
        Port 2: Pt3911, 3-wire, DegC
        Port 3: Pt3916, 4-wire, Kelvins
    E101Mod8:
        Port 0: Pt3920, 3-wire, DegR
        Port 1: Pt3928, 3-wire, DegC
        Port 2: Pt3850, 4-wire, DegF
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
        mod7 = devices["E101Mod7"]
        mod8 = devices["E101Mod8"]
        return [
            # --- E101Mod7 (ports 0-3) ---
            sy.ni.AIRTDChan(
                device=mod7.key,
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
                device=mod7.key,
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
                min_val=32.0,
                max_val=212.0,
            ),
            sy.ni.AIRTDChan(
                device=mod7.key,
                port=2,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3911_3w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                rtd_type="Pt3911",
                resistance_config="3Wire",
                current_excit_source="Internal",
                current_excit_val=0.001,
                r0=100.0,
                min_val=-50.0,
                max_val=200.0,
            ),
            sy.ni.AIRTDChan(
                device=mod7.key,
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
                r0=100.0,
                min_val=273.0,
                max_val=373.0,
            ),
            # --- E101Mod8 (ports 0-2) ---
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
                min_val=491.0,
                max_val=671.0,
            ),
            sy.ni.AIRTDChan(
                device=mod8.key,
                port=1,
                channel=create_channel(
                    client,
                    name="ni_rtd_pt3928_3w",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                rtd_type="Pt3928",
                resistance_config="3Wire",
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
                r0=100.0,
                min_val=32.0,
                max_val=212.0,
            ),
        ]


class NIReadTC(NIAnalogReadTaskCase):
    """Read thermocouples across two NI 9211 modules — all 8 TC types.

    E101Mod5:
        Port 0: J-type, DegC
        Port 1: K-type, DegF
        Port 2: T-type, DegC
        Port 3: E-type, Kelvins
    E101Mod6:
        Port 0: R-type, DegC
        Port 1: S-type, DegF
        Port 2: B-type, DegC
        Port 3: N-type, DegR
    """

    task_name = "NI Thermocouple Read"
    device_locations = ["E101Mod5", "E101Mod6"]  # NI 9211

    SAMPLE_RATE = 20 * sy.Rate.HZ
    STREAM_RATE = 5 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIThermocoupleChan]:
        idx = create_index(client, "ni_tc_index")
        mod5 = devices["E101Mod5"]
        mod6 = devices["E101Mod6"]
        return [
            # --- E101Mod5 (ports 0-3) ---
            sy.ni.AIThermocoupleChan(
                device=mod5.key,
                port=0,
                channel=create_channel(
                    client,
                    name="ni_tc_j",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                thermocouple_type="J",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=-50.0,
                max_val=500.0,
            ),
            sy.ni.AIThermocoupleChan(
                device=mod5.key,
                port=1,
                channel=create_channel(
                    client,
                    name="ni_tc_k",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegF",
                thermocouple_type="K",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=32.0,
                max_val=932.0,
            ),
            sy.ni.AIThermocoupleChan(
                device=mod5.key,
                port=2,
                channel=create_channel(
                    client,
                    name="ni_tc_t",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                thermocouple_type="T",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=-50.0,
                max_val=300.0,
            ),
            sy.ni.AIThermocoupleChan(
                device=mod5.key,
                port=3,
                channel=create_channel(
                    client,
                    name="ni_tc_e",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="Kelvins",
                thermocouple_type="E",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=223.0,
                max_val=773.0,
            ),
            # --- E101Mod6 (ports 0-3) ---
            sy.ni.AIThermocoupleChan(
                device=mod6.key,
                port=0,
                channel=create_channel(
                    client,
                    name="ni_tc_r",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                thermocouple_type="R",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=0.0,
                max_val=1500.0,
            ),
            sy.ni.AIThermocoupleChan(
                device=mod6.key,
                port=1,
                channel=create_channel(
                    client,
                    name="ni_tc_s",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegF",
                thermocouple_type="S",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=32.0,
                max_val=2732.0,
            ),
            sy.ni.AIThermocoupleChan(
                device=mod6.key,
                port=2,
                channel=create_channel(
                    client,
                    name="ni_tc_b",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
                thermocouple_type="B",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=250.0,
                max_val=1700.0,
            ),
            sy.ni.AIThermocoupleChan(
                device=mod6.key,
                port=3,
                channel=create_channel(
                    client,
                    name="ni_tc_n",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegR",
                thermocouple_type="N",
                cjc_source="BuiltIn",
                cjc_val=0.0,
                cjc_port=0,
                min_val=491.0,
                max_val=1851.0,
            ),
        ]


class NIReadCurrentVoltage(NIAnalogReadTaskCase):
    """Read current and voltage across two modules in a single task.

    E101Mod3 (NI 9203): 4-20 mA current input
        Port 0: Current, internal shunt
        Port 1: Current, internal shunt
    E101Mod4 (NI 9205): Voltage input
        Port 0: Voltage, differential
        Port 1: Voltage, RSE
    """

    task_name = "NI Current + Voltage Read"
    device_locations = ["E101Mod3", "E101Mod4"]

    SAMPLE_RATE = 1000 * sy.Rate.HZ
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIChan]:
        idx = create_index(client, "ni_cur_volt_index")
        mod3 = devices["E101Mod3"]
        mod4 = devices["E101Mod4"]
        return [
            # --- E101Mod3 / NI 9203 (current) ---
            sy.ni.AICurrentChan(
                device=mod3.key,
                port=0,
                channel=create_channel(
                    client,
                    name="ni_current_0",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                min_val=0.004,
                max_val=0.02,
                shunt_resistor_loc="Internal",
                ext_shunt_resistor_val=249.0,
            ),
            sy.ni.AICurrentChan(
                device=mod3.key,
                port=1,
                channel=create_channel(
                    client,
                    name="ni_current_1",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                min_val=0.004,
                max_val=0.02,
                shunt_resistor_loc="Internal",
                ext_shunt_resistor_val=249.0,
            ),
            # --- E101Mod4 / NI 9205 (voltage) ---
            sy.ni.AIVoltageChan(
                device=mod4.key,
                port=0,
                channel=create_channel(
                    client,
                    name="ni_voltage_diff",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Diff",
                min_val=-10.0,
                max_val=10.0,
            ),
            sy.ni.AIVoltageChan(
                device=mod4.key,
                port=1,
                channel=create_channel(
                    client,
                    name="ni_voltage_rse",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="RSE",
                min_val=-10.0,
                max_val=10.0,
            ),
        ]


class NIReadVoltageUSB(NIAnalogReadTaskCase):
    """Read voltage on USB-6000.

    The USB-6000 has 8 single-ended (RSE) analog inputs at 10 kS/s max.

    Port 0: RSE, +/- 10V
    Port 1: RSE, +/- 10V
    """

    task_name = "NI Voltage Read (USB-6000)"
    device_locations = ["USB-6000"]

    SAMPLE_RATE = 1000 * sy.Rate.HZ
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, "ni_usb6000_index")
        return [
            sy.ni.AIVoltageChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_usb6000_ai0",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="RSE",
                min_val=-10.0,
                max_val=10.0,
            ),
            sy.ni.AIVoltageChan(
                port=1,
                channel=create_channel(
                    client,
                    name="ni_usb6000_ai1",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="RSE",
                min_val=-10.0,
                max_val=10.0,
            ),
        ]


class NIReadVoltageDiffNRSE(NIAnalogReadTaskCase):
    """Read voltage with differential and NRSE terminal configs on NI 9206.

    NI 9206: 16 differential / 32 single-ended channels, 250 kS/s aggregate,
    +/-200mV to +/-10V programmable ranges, 16-bit.

    Port 0: Differential, +/- 10V
    Port 1: Differential, +/- 5V
    Port 2: NRSE, +/- 10V
    """

    task_name = "NI Voltage Read (Diff + NRSE)"
    device_locations = ["E102Mod1"]  # NI 9206

    SAMPLE_RATE = 10000 * sy.Rate.HZ
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, "ni_9206_index")
        return [
            sy.ni.AIVoltageChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_9206_diff_10v",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Diff",
                min_val=-10.0,
                max_val=10.0,
            ),
            sy.ni.AIVoltageChan(
                port=1,
                channel=create_channel(
                    client,
                    name="ni_9206_diff_5v",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Diff",
                min_val=-5.0,
                max_val=5.0,
            ),
            sy.ni.AIVoltageChan(
                port=2,
                channel=create_channel(
                    client,
                    name="ni_9206_nrse_10v",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="NRSE",
                min_val=-10.0,
                max_val=10.0,
            ),
        ]


class NIReadVoltagePseudoDiff(NIAnalogReadTaskCase):
    """Read voltage with pseudo-differential terminal config on NI 9234.

    NI 9234: 4 channels (BNC), simultaneous sampling at up to 51.2 kS/s
    per channel, +/-5V, 24-bit, designed for IEPE/sound & vibration.

    Port 0: PseudoDiff, +/- 5V
    Port 1: PseudoDiff, +/- 5V
    """

    task_name = "NI Voltage Read (PseudoDiff)"
    device_locations = ["E102Mod2"]  # NI 9234

    SAMPLE_RATE = 25600 * sy.Rate.HZ
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, "ni_9234_index")
        return [
            sy.ni.AIVoltageChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_9234_pdiff_0",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="PseudoDiff",
                min_val=-5.0,
                max_val=5.0,
            ),
            sy.ni.AIVoltageChan(
                port=1,
                channel=create_channel(
                    client,
                    name="ni_9234_pdiff_1",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="PseudoDiff",
                min_val=-5.0,
                max_val=5.0,
            ),
        ]


class NIReadResistance(NIAnalogReadTaskCase):
    """Read resistance on NI 9219 (2-wire and 4-wire only, 500 uA excitation).

    Port 0: 4-wire, 0-1 kOhm
    Port 1: 2-wire, 0-10 kOhm
    Port 2: 4-wire, 0-1 kOhm
    """

    task_name = "NI Resistance Read"
    device_locations = ["E101Mod2"]  # NI 9219

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
                current_excit_val=0.0005,
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
                current_excit_val=0.0005,
                min_val=0.0,
                max_val=10000.0,
            ),
            sy.ni.AIResistanceChan(
                port=2,
                channel=create_channel(
                    client,
                    name="ni_res_4w_1k_b",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                resistance_config="4Wire",
                current_excit_source="Internal",
                current_excit_val=0.0005,
                min_val=0.0,
                max_val=1000.0,
            ),
        ]


class NIReadTempBuiltIn(NIAnalogReadTaskCase):
    """Read built-in temperature sensor on USB-6289.

    The USB-6289 has an onboard temperature sensor accessible via
    AITempBuiltInChan. No external sensor required.

    Port 0: Built-in temp, DegC
    """

    task_name = "NI Built-In Temp Read (USB-6289)"
    device_locations = ["USB-6289"]

    SAMPLE_RATE = 25 * sy.Rate.HZ
    STREAM_RATE = 5 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AITempBuiltInChan]:
        idx = create_index(client, "ni_temp_builtin_index")
        return [
            sy.ni.AITempBuiltInChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_usb6289_temp",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                units="DegC",
            ),
        ]
 
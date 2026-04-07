#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI read task integration tests."""

import numpy as np

import synnax as sy
from tests.driver.ni_task import (
    NIAnalogReadTaskCase,
    NICounterReadTaskCase,
    NIDigitalReadTaskCase,
)
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


class NIAnalogReadScaled(NIAnalogReadTaskCase):
    """Read voltage with custom scales, verify values fall in scaled bounds.

    E101Mod4 (NI 9205) — sim DAQ outputs a sine wave in -10V to +10V.

    Port | Scale    | Params                       | Output Range
    -----|----------|------------------------------|-------------
    0    | MapScale | -10V..+10V → 500..700        | [500, 700]
    1    | LinScale | slope=60, intercept=1200      | [600, 1800]
    2    | None     | nominal volts (control)       | [-10, 10]
    """

    task_name = "NI Analog Voltage Read (Scaled)"
    device_locations = ["E101Mod4"]  # NI 9205

    SAMPLE_RATE = 25 * sy.Rate.HZ
    STREAM_RATE = 5 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, "ni_scaled_index")
        map_scale = sy.ni.MapScale(
            pre_scaled_min=-10,
            pre_scaled_max=10,
            scaled_min=500,
            scaled_max=700,
            pre_scaled_units="Volts",
        )
        # slope=60: each volt = 60 scaled units, intercept=1200: 0V = 1200
        # -10V → 600, 0V → 1200, +10V → 1800
        lin_scale = sy.ni.LinScale(
            slope=60,
            y_intercept=1200,
            pre_scaled_units="Volts",
            scaled_units="Volts",
        )
        return [
            sy.ni.AIVoltageChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_scaled_map",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                min_val=500,
                max_val=700,
                custom_scale=map_scale,
            ),
            sy.ni.AIVoltageChan(
                port=1,
                channel=create_channel(
                    client,
                    name="ni_scaled_lin",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                min_val=600,
                max_val=1800,
                custom_scale=lin_scale,
            ),
            sy.ni.AIVoltageChan(
                port=2,
                channel=create_channel(
                    client,
                    name="ni_scaled_none",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                min_val=-10.0,
                max_val=10.0,
            ),
        ]

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()
        self.test_scaled_values()
        self.test_tare()

    def test_scaled_values(self) -> None:
        """Stream data and verify each channel's values fall within its scaled bounds."""
        assert self.tsk is not None
        self.log("Testing: Verify scaled value ranges")
        channel_keys = self._channel_keys(self.tsk)

        # Collect samples for 2 seconds
        with self.tsk.run():
            with self.client.open_streamer(channel_keys) as streamer:
                streamer.read(timeout=30)  # wait for first frame
            sy.sleep(0.5)
            start = sy.TimeStamp.now()
            sy.sleep(1)

        end = sy.TimeStamp.now()
        tr = sy.TimeRange(start, end)

        # Expected bounds per channel (port order: map, lin, none)
        bounds = [
            ("ni_scaled_map", 500, 700),
            ("ni_scaled_lin", 600, 1800),
            ("ni_scaled_none", -10, 10),
        ]
        for key, (name, lo, hi) in zip(channel_keys, bounds):
            data = np.array(self.client.channels.retrieve(key).read(tr))
            if len(data) == 0:
                self.fail(f"No samples for '{name}'")
            vmin, vmax = float(np.min(data)), float(np.max(data))
            if vmin < lo - 1 or vmax > hi + 1:
                self.fail(
                    f"'{name}' values [{vmin:.2f}, {vmax:.2f}] "
                    f"outside expected [{lo}, {hi}]"
                )
            # Confirm the scaled channels actually produce values outside [-10, 10]
            if lo > 10 and vmax <= 10:
                self.fail(
                    f"'{name}' max={vmax:.2f} — scaling not applied "
                    f"(expected values above 10)"
                )
            self.log(f"  {name}: [{vmin:.2f}, {vmax:.2f}] within [{lo}, {hi}]")

    def test_tare(self) -> None:
        """Tare all channels and verify values shift to ~0."""
        assert self.tsk is not None
        self.log("Testing: Tare")
        channel_keys = self._channel_keys(self.tsk)
        names = ["ni_scaled_map", "ni_scaled_lin", "ni_scaled_none"]

        with self.tsk.run():
            with self.client.open_streamer(channel_keys) as streamer:
                streamer.read(timeout=30)

            # Capture pre-tare averages
            pre_start = sy.TimeStamp.now()
            sy.sleep(0.5)
            pre_end = sy.TimeStamp.now()
            pre_tr = sy.TimeRange(pre_start, pre_end)

            pre_avgs: dict[int, float] = {}
            for key, name in zip(channel_keys, names):
                data = np.array(self.client.channels.retrieve(key).read(pre_tr))
                if len(data) == 0:
                    self.fail(f"No pre-tare samples for '{name}'")
                pre_avgs[key] = float(np.mean(data))
            self.log(
                "  Pre-tare averages: "
                + ", ".join(
                    f"{n}={pre_avgs[k]:.2f}" for k, n in zip(channel_keys, names)
                )
            )

            # Execute tare on all channels (async — tare doesn't send an ACK)
            self.tsk._internal.execute_command("tare", {"keys": channel_keys})
            self.log("  Tare command sent")

            # Capture post-tare values
            sy.sleep(0.5)
            post_start = sy.TimeStamp.now()
            sy.sleep(0.5)
            post_end = sy.TimeStamp.now()

        post_tr = sy.TimeRange(post_start, post_end)

        # Post-tare range should be original bounds shifted by -pre_avg.
        bounds = [
            ("ni_scaled_map", 500, 700),
            ("ni_scaled_lin", 600, 1800),
            ("ni_scaled_none", -10, 10),
        ]
        for key, (name, orig_lo, orig_hi) in zip(channel_keys, bounds):
            data = np.array(self.client.channels.retrieve(key).read(post_tr))
            if len(data) == 0:
                self.fail(f"No post-tare samples for '{name}'")
            avg = float(np.mean(data))
            vmin, vmax = float(np.min(data)), float(np.max(data))

            offset = pre_avgs[key]
            tared_lo = orig_lo - offset
            tared_hi = orig_hi - offset
            tolerance = 2
            if vmin < tared_lo - tolerance or vmax > tared_hi + tolerance:
                self.fail(
                    f"'{name}' post-tare [{vmin:.2f}, {vmax:.2f}] "
                    f"outside expected [{tared_lo:.2f}, {tared_hi:.2f}]"
                )
            if abs(avg) > abs(offset) * 0.5 + tolerance:
                self.fail(
                    f"'{name}' post-tare avg={avg:.2f} not near 0 "
                    f"(pre-tare avg was {offset:.2f})"
                )
            self.log(
                f"  {name}: avg={avg:.2f}, range=[{vmin:.2f}, {vmax:.2f}], "
                f"expected ~[{tared_lo:.2f}, {tared_hi:.2f}]"
            )


class NIReadTemperature(NIAnalogReadTaskCase):
    """Read temperature sensors across E101 chassis — TC, RTD, and resistance.

    Cross-device task spanning 5 modules: 2x NI 9211 (TC), 2x NI 9219 (RTD),
    1x NI 9219 (resistance).

    E101Mod2 (NI 9219) — Resistance:
        Port 0: 4-wire, 0-1 kOhm
        Port 1: 2-wire, 0-10 kOhm
        Port 2: 4-wire, 0-1 kOhm
    E101Mod5 (NI 9211) — Thermocouple:
        Port 0: J-type, DegC
        Port 1: K-type, DegF
        Port 2: T-type, DegC
        Port 3: E-type, Kelvins
    E101Mod6 (NI 9211) — Thermocouple:
        Port 0: R-type, DegC
        Port 1: S-type, DegF
        Port 2: B-type, DegC
        Port 3: N-type, DegR
    E101Mod7 (NI 9219) — RTD:
        Port 0: Pt3750, 4-wire, DegC
        Port 1: Pt3851, 3-wire, DegF
        Port 2: Pt3911, 3-wire, DegC
        Port 3: Pt3916, 4-wire, Kelvins
    E101Mod8 (NI 9219) — RTD:
        Port 0: Pt3920, 3-wire, DegR
        Port 1: Pt3928, 3-wire, DegC
        Port 2: Pt3850, 4-wire, DegF
    """

    task_name = "NI Temperature + Resistance Read"
    device_locations = [
        "E101Mod2",
        "E101Mod5",
        "E101Mod6",
        "E101Mod7",
        "E101Mod8",
    ]

    SAMPLE_RATE = 20 * sy.Rate.HZ
    STREAM_RATE = 5 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIChan]:
        idx = create_index(client, "ni_temp_index")
        mod2 = devices["E101Mod2"]
        mod5 = devices["E101Mod5"]
        mod6 = devices["E101Mod6"]
        mod7 = devices["E101Mod7"]
        mod8 = devices["E101Mod8"]
        return [
            # --- E101Mod2 / NI 9219 (resistance) ---
            sy.ni.AIResistanceChan(
                device=mod2.key,
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
                device=mod2.key,
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
                device=mod2.key,
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
            # --- E101Mod5 / NI 9211 (thermocouple) ---
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
            # --- E101Mod6 / NI 9211 (thermocouple) ---
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
            # --- E101Mod7 / NI 9219 (RTD) ---
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
            # --- E101Mod8 / NI 9219 (RTD) ---
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


class NIReadTempBuiltIn(NIAnalogReadTaskCase):
    """Read built-in temperature sensor on USB-6289.

    USB devices don't support multi-device tasks, so this runs standalone.

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


class NIReadVoltageTerminalConfigs(NIAnalogReadTaskCase):
    """Read voltage with Diff, NRSE, and PseudoDiff terminal configs.

    Cross-device task spanning NI 9206 and NI 9234.

    E102Mod1 (NI 9206): 16 diff / 32 SE channels, 250 kS/s, 16-bit
        Port 0: Differential, +/- 10V
        Port 1: Differential, +/- 5V
        Port 2: NRSE, +/- 10V
    E102Mod2 (NI 9234): 4 BNC channels, 51.2 kS/s/ch, 24-bit
        Port 0: PseudoDiff, +/- 5V
        Port 1: PseudoDiff, +/- 5V
    """

    task_name = "NI Voltage Read (Terminal Configs)"
    device_locations = ["E102Mod1", "E102Mod2"]  # NI 9206 + NI 9234

    SAMPLE_RATE = 10240 * sy.Rate.HZ  # 51200/5, valid for NI 9234 delta-sigma
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, "ni_terminal_cfg_index")
        mod1 = devices["E102Mod1"]
        mod2 = devices["E102Mod2"]
        return [
            # --- E102Mod1 / NI 9206 (Diff + NRSE) ---
            sy.ni.AIVoltageChan(
                device=mod1.key,
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
                device=mod1.key,
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
                device=mod1.key,
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
            # --- E102Mod2 / NI 9234 (PseudoDiff) ---
            sy.ni.AIVoltageChan(
                device=mod2.key,
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
                device=mod2.key,
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


class NIDigitalRead(NIDigitalReadTaskCase):
    """Read digital inputs on NI 9375.

    NI 9375: 16 DI (sinking, 24V) + 16 DO. Port 0 = DI lines 0-7,
    Port 1 = DI lines 0-7.

    Port 0, Line 0: Digital input
    Port 0, Line 1: Digital input
    """

    task_name = "NI Digital Read"
    device_locations = ["E102Mod3"]  # NI 9375

    SAMPLE_RATE = 1000 * sy.Rate.HZ
    STREAM_RATE = 50 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.DIChan]:
        idx = create_index(client, "ni_di_index")
        return [
            sy.ni.DIChan(
                port=0,
                line=i,
                channel=create_channel(
                    client,
                    name=f"ni_di_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
            )
            for i in range(4)
        ]


class NICounterReadEdgeCount(NICounterReadTaskCase):
    """Read edge count on USB-6289 counter 0.

    NI-DAQmx only allows one counter per task on USB devices (-200147),
    so we use a single edge count channel on ctr0.
    """

    task_name = "NI Counter Read Edge Count (USB-6289)"
    device_locations = ["USB-6289"]

    SAMPLE_RATE = 100 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.CIEdgeCountChan]:
        idx = create_index(client, "ni_ci_edge_index")
        return [
            sy.ni.CIEdgeCountChan(
                port=0,
                channel=create_channel(
                    client,
                    name="ni_ci_edge_count_0",
                    data_type=sy.DataType.UINT32,
                    index=idx.key,
                ),
                active_edge="Rising",
                count_direction="CountUp",
                initial_count=0,
            ),
        ]


class NICounterReadFrequency(NICounterReadTaskCase):
    """Read frequency on USB-6289 counter 1.

    NI-DAQmx only allows one counter per task on USB devices (-200147),
    so we use a separate task for the frequency channel on ctr1.
    """

    task_name = "NI Counter Read Frequency (USB-6289)"
    device_locations = ["USB-6289"]

    SAMPLE_RATE = 100 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.CIFrequencyChan]:
        idx = create_index(client, "ni_ci_freq_index")
        return [
            sy.ni.CIFrequencyChan(
                port=1,
                channel=create_channel(
                    client,
                    name="ni_ci_freq_0",
                    data_type=sy.DataType.FLOAT64,
                    index=idx.key,
                ),
                min_val=1,
                max_val=10000,
                units="Hz",
                edge="Rising",
                meas_method="LowFreq1Ctr",
                meas_time=0.001,
                divisor=4,
            ),
        ]

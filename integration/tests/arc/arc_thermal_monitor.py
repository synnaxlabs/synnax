#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Arc Thermal Monitor Integration Test

Tests Arc features not covered by arc_press_sequence:
- Stateful variables ($=) for rising-edge detection and peak tracking
- Custom functions with arithmetic and conditionals
- Looping sequences (cooling => heating cycle)
- Cross-sequence transitions (overheat => abort)
- Multiple entry points (start_monitor_cmd, abort_cmd)
- Calculated channels (temp_error)
"""

import time

import synnax as sy

from console.case import ConsoleCase

ARC_NAME = f"ArcThermalMonitor_{int(time.time())}"

ARC_SOURCE = """
func count_heater_cycles(heater_on u8) i64 {
    prev u8 $= 0
    count i64 $= 0
    if heater_on == 1 and prev == 0 {
        count = count + 1
    }
    prev = heater_on
    return count
}

func track_peak_temp(current f32) f32 {
    peak f32 $= 0.0
    if current > peak {
        peak = current
    }
    return peak
}

func calc_temp_error{setpoint f32}(measured f32) f32 {
    return measured - setpoint
}

start_monitor_cmd => monitor
abort_cmd => abort

heater_cmd -> count_heater_cycles{} -> cycle_count
temp_sensor -> track_peak_temp{} -> peak_temp
temp_sensor -> calc_temp_error{setpoint=50.0} -> temp_error

sequence monitor {
    stage heating {
        1 -> heater_cmd,
        temp_sensor > 80 => abort,
        temp_sensor > 60 => cooling
    }
    stage cooling {
        0 -> heater_cmd,
        temp_sensor > 80 => abort,
        temp_sensor < 40 => heating
    }
}

sequence abort {
    stage safed {
        0 -> heater_cmd,
        1 -> alarm_active
    }
}
"""


class ArcThermalMonitor(ConsoleCase):
    """Test Arc thermal monitor with stateful variables and looping sequence."""

    def setup(self) -> None:
        self.set_manual_timeout(180)
        self._create_channels()
        self.subscribe(
            [
                "temp_sensor",
                "heater_state",
                "cycle_count",
                "peak_temp",
                "temp_error",
                "alarm_active",
                "end_thermal_test_cmd",
            ]
        )
        super().setup()

    def _create_channels(self) -> None:
        self.client.channels.create(
            name="start_monitor_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="abort_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        cycle_count_time = self.client.channels.create(
            name="cycle_count_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="cycle_count",
            data_type=sy.DataType.INT64,
            index=cycle_count_time.key,
            retrieve_if_name_exists=True,
        )

        peak_temp_time = self.client.channels.create(
            name="peak_temp_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="peak_temp",
            data_type=sy.DataType.FLOAT32,
            index=peak_temp_time.key,
            retrieve_if_name_exists=True,
        )

        temp_error_time = self.client.channels.create(
            name="temp_error_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="temp_error",
            data_type=sy.DataType.FLOAT32,
            index=temp_error_time.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        self.log("Creating Arc thermal monitor")
        self.console.arc.create(ARC_NAME, ARC_SOURCE, mode="Text")
        sy.sleep(0.5)

        rack = self.client.racks.retrieve_embedded_rack()

        self.log(f"Selecting rack: {rack.name}")
        self.console.arc.select_rack(rack.name)

        self.log("Configuring Arc task")
        self.console.arc.configure()
        sy.sleep(1.0)

        self.log("Starting Arc task")
        self.console.arc.start()
        sy.sleep(0.5)

        self.log("Triggering monitor sequence")
        with self.client.open_writer(sy.TimeStamp.now(), "start_monitor_cmd") as w:
            w.write("start_monitor_cmd", 1)

        self._verify_thermal_cycling()
        self._verify_stateful_variables()
        self._verify_abort_transition()

        self.log("Stopping Arc task")
        self.console.arc.stop()
        sy.sleep(0.5)

        self.log("Deleting Arc program")
        self.console.arc.delete(ARC_NAME)

        self.log("Signaling thermal sim to stop")
        with self.client.open_writer(sy.TimeStamp.now(), "end_thermal_test_cmd") as w:
            w.write("end_thermal_test_cmd", 1)

        self.log("Arc thermal monitor test completed")

    def _verify_thermal_cycling(self) -> None:
        self.log("Verifying thermal cycling behavior...")

        self.log("Waiting for heater to turn on (heating stage)...")
        while self.should_continue:
            if self.read_tlm("heater_state") == 1:
                self.log("Heater ON - heating stage active")
                break
            if self.should_stop:
                self.fail("Heater should turn on")
                return

        self.log("Waiting for temp to rise and heater to turn off (cooling stage)...")
        while self.should_continue:
            if self.read_tlm("heater_state") == 0:
                temp = self.read_tlm("temp_sensor")
                self.log(f"Heater OFF at temp={temp:.1f} - cooling stage active")
                break
            if self.should_stop:
                self.fail("Heater should turn off when temp > 60")
                return

        self.log("Waiting for heater to turn back on (looping back to heating)...")
        while self.should_continue:
            if self.read_tlm("heater_state") == 1:
                temp = self.read_tlm("temp_sensor")
                self.log(f"Heater ON again at temp={temp:.1f} - loop confirmed")
                break
            if self.should_stop:
                self.fail("Heater should turn back on when temp < 40")
                return

    def _verify_stateful_variables(self) -> None:
        self.log("Verifying stateful variable behavior...")

        cycle_count = self.read_tlm("cycle_count")
        self.log(f"Cycle count: {cycle_count}")
        if cycle_count is None or cycle_count < 2:
            self.fail(f"Expected cycle_count >= 2, got {cycle_count}")
            return

        peak_temp = self.read_tlm("peak_temp")
        self.log(f"Peak temperature tracked: {peak_temp:.1f}" if peak_temp else "None")
        if peak_temp is None or peak_temp < 55:
            self.fail(f"Expected peak_temp > 55, got {peak_temp}")
            return

        temp_error = self.read_tlm("temp_error")
        current_temp = self.read_tlm("temp_sensor")
        if temp_error is None or current_temp is None:
            self.fail(f"temp_error or current_temp is None")
            return
        expected_error = current_temp - 50.0
        self.log(f"Temp error: {temp_error:.1f} (expected ~{expected_error:.1f})")
        if abs(temp_error - expected_error) > 1.0:
            self.fail(
                f"temp_error {temp_error:.1f} doesn't match expected {expected_error:.1f}"
            )
            return

    def _verify_abort_transition(self) -> None:
        self.log("Verifying abort transition (temp > 80)...")

        self.log("Triggering force overheat to simulate thermal runaway")
        with self.client.open_writer(sy.TimeStamp.now(), "force_overheat_cmd") as w:
            w.write("force_overheat_cmd", 1)

        self.log("Waiting for temp to exceed 80...")
        while self.should_continue:
            temp = self.read_tlm("temp_sensor")
            if temp is not None and temp > 80:
                self.log(f"Temperature exceeded 80: {temp:.1f}")
                break
            if self.should_stop:
                self.fail("Temperature should exceed 80 during force overheat")
                return

        self.log("Waiting for abort sequence (heater off, alarm active)...")
        log_counter = 0
        while self.should_continue:
            heater = self.read_tlm("heater_state")
            alarm = self.read_tlm("alarm_active")
            log_counter += 1
            if log_counter % 50 == 0:
                self.log(f"Checking abort: heater={heater}, alarm={alarm}")
            if heater == 0 and alarm == 1:
                self.log("Abort sequence confirmed: heater OFF, alarm ACTIVE")
                break
            if self.should_stop:
                self.fail(
                    f"Abort sequence should activate (heater={heater}, alarm={alarm})"
                )
                return

        with self.client.open_writer(sy.TimeStamp.now(), "force_overheat_cmd") as w:
            w.write("force_overheat_cmd", 0)

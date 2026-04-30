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

from examples.simulators import ThermalSimDAQ

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

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

func calc_temp_error{setpoint f32} (measured f32) f32 {
    return measured - setpoint
}

start_monitor_cmd => monitor
abort_cmd => abort

heater_cmd -> count_heater_cycles{} -> cycle_count
temp_sensor -> track_peak_temp{} -> peak_temp
temp_sensor -> calc_temp_error{50.0} -> temp_error

sequence monitor {
    stage heating {
        1 -> heater_cmd
        temp_sensor > 80 => abort
        temp_sensor > 60 => cooling
    }
    stage cooling {
        0 -> heater_cmd
        temp_sensor > 80 => abort
        temp_sensor < 40 => heating
    }
}

sequence abort {
    stage safed {
        0 -> heater_cmd
        1 -> alarm_active
    }
}
"""


class ThermalMonitor(ArcConsoleCase):
    """Test Arc thermal monitor with stateful variables and looping sequence."""

    arc_source = ARC_SOURCE
    arc_name_prefix = "ArcThermalMonitor"
    start_cmd_channel = "start_monitor_cmd"
    end_cmd_channel = "end_thermal_test_cmd"
    subscribe_channels = [
        "temp_sensor",
        "heater_state",
        "cycle_count",
        "peak_temp",
        "temp_error",
        "alarm_active",
        "end_thermal_test_cmd",
    ]
    sim_daq_class = ThermalSimDAQ

    def setup(self) -> None:
        self._create_additional_channels()
        super().setup()

    def _create_additional_channels(self) -> None:
        create_virtual_channel(self.client, "abort_cmd", sy.DataType.UINT8)
        create_indexed_pair(self.client, "cycle_count", sy.DataType.INT64)
        create_indexed_pair(self.client, "peak_temp", sy.DataType.FLOAT32)
        create_indexed_pair(self.client, "temp_error", sy.DataType.FLOAT32)

    def verify_sequence_execution(self) -> None:
        self._verify_thermal_cycling()
        self._verify_stateful_variables()
        self._verify_abort_transition()

    def _verify_thermal_cycling(self) -> None:
        self.log("Verifying thermal cycling behavior...")

        self.log("Waiting for heater to turn on (heating stage)...")
        self.wait_for_eq("heater_state", 1)
        self.log("Heater ON - heating stage active")

        self.log("Waiting for temp to rise and heater to turn off (cooling stage)...")
        self.wait_for_eq("heater_state", 0)
        temp = self.read_tlm("temp_sensor")
        self.log(f"Heater OFF at temp={temp:.1f} - cooling stage active")

        self.log("Waiting for heater to turn back on (looping back to heating)...")
        self.wait_for_eq("heater_state", 1)
        temp = self.read_tlm("temp_sensor")
        self.log(f"Heater ON again at temp={temp:.1f} - loop confirmed")

    def _verify_stateful_variables(self) -> None:
        self.log("Verifying stateful variable behavior...")

        self.wait_for_ge("cycle_count", 2, timeout=0)
        self.log(f"Cycle count: {self.read_tlm('cycle_count')}")

        self.wait_for_gt("peak_temp", 55, timeout=0)
        self.log(f"Peak temperature tracked: {self.read_tlm('peak_temp'):.1f}")

        temp_error = self.read_tlm("temp_error")
        current_temp = self.read_tlm("temp_sensor")
        if temp_error is None or current_temp is None:
            self.fail("temp_error or current_temp is None")
            return
        expected_error = current_temp - 50.0
        self.log(f"Temp error: {temp_error:.1f} (expected ~{expected_error:.1f})")
        if abs(temp_error - expected_error) > 1.0:
            self.fail(
                f"temp_error {temp_error:.1f} doesn't match expected {expected_error:.1f}"
            )

    def _verify_abort_transition(self) -> None:
        self.log("Verifying abort transition (temp > 80)...")

        self.log("Triggering force overheat to simulate thermal runaway")
        self.writer.write("force_overheat_cmd", 1)

        self.log("Waiting for temp to exceed 80...")
        self.wait_for_gt("temp_sensor", 80)
        temp = self.read_tlm("temp_sensor")
        self.log(f"Temperature exceeded 80: {temp:.1f}")

        self.log("Waiting for abort sequence (heater off, alarm active)...")
        self.wait_for_eq("heater_state", 0)
        self.wait_for_eq("alarm_active", 1)
        self.log("Abort sequence confirmed: heater OFF, alarm ACTIVE")

        self.writer.write("force_overheat_cmd", 0)

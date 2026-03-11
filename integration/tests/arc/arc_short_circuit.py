#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Arc Short Circuit Integration Test

Verifies that when multiple stage transition conditions are true
simultaneously, the first-written transition takes priority and short-circuits
execution on the stage.

Phase 1: ss_temp_a=200, ss_temp_b=400
    Only the second condition is true (ss_temp_b > 300), so we loop on/pause.

Phase 2: set ss_temp_a=400
    Both conditions become true. The first condition (=> off) should win.
"""

import synnax as sy
from examples.simulators import PressSimDAQ

from tests.arc.arc_case import ArcConsoleCase

SHORT_CIRCUIT_SOURCE = """
func count(val u8) f32 {
    n f32 $= 0
    n = n + 1
    return n
}

ss_start_cmd => main

sequence main {
    stage on {
        0 -> ss_sim_stage,
        1 -> ss_heater_cmd,
        "on" -> ss_stage_str,
        interval{period=1s} -> (ss_temp_a > 290 and ss_temp_b > 290) => off,
        interval{period=1s} -> ss_temp_b > 300 => pause,
    }
    stage pause {
        ss_sim_stage -> count{} -> ss_count,
        2 -> ss_sim_stage,
        0 -> ss_heater_cmd,
        "pause" -> ss_stage_str,
        wait{duration=1s} => on,
    }
    stage off {
        3 -> ss_sim_stage,
        0 -> ss_heater_cmd,
        "off" -> ss_stage_str,
        ss_start_cmd => on,
    }
}
"""


class ArcShortCircuit(ArcConsoleCase):
    """Test that first transition short-circuits the stage."""

    arc_source = SHORT_CIRCUIT_SOURCE
    arc_name_prefix = "ArcShortCircuit"
    start_cmd_channel = "ss_start_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "ss_stage_str",
        "ss_sim_stage",
        "ss_heater_cmd",
        "ss_temp_a",
        "ss_temp_b",
        "ss_count",
        "end_test_cmd",
    ]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        client = self.client

        client.channels.create(
            name="ss_stage_str",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="ss_count",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        for name in ["ss_sim_stage", "ss_heater_cmd"]:
            client.channels.create(
                name=name,
                data_type=sy.DataType.UINT8,
                virtual=True,
                retrieve_if_name_exists=True,
            )

        self.ss_sensor_time = client.channels.create(
            name="ss_sensor_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="ss_temp_a",
            index=self.ss_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="ss_temp_b",
            index=self.ss_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        super().setup()

    def verify_sequence_execution(self) -> None:
        with self.client.open_writer(
            start=sy.TimeStamp.now(),
            channels=["ss_sensor_time", "ss_temp_a", "ss_temp_b"],
            name="Short Circuit Sensor Writer",
        ) as self._sensor_writer:
            self._ss_temp_a = 200.0
            self._ss_temp_b = 400.0
            self._write_sensors()

            self._verify_on_pause_loop()
            self._verify_off_transition()

    def _write_sensors(self) -> None:
        now = sy.TimeStamp.now()
        self._sensor_writer.write(
            {
                "ss_sensor_time": now,
                "ss_temp_a": self._ss_temp_a,
                "ss_temp_b": self._ss_temp_b,
            }
        )

    def _verify_on_pause_loop(self) -> None:
        """Phase 1: ss_temp_a=200, ss_temp_b=400 => only pause condition true.

        With 500ms intervals, the 'on' stage may transition to 'pause' before
        we can poll it. We verify the loop by observing 'pause' entries and
        continuously writing sensor values so the runtime receives them.
        """
        self.log("Phase 1: Verifying on/pause loop")
        for i in range(1, 4):
            self.wait_for_near("ss_count", float(i), tolerance=0.01, is_virtual=True)
            self._write_sensors()
        self.log("Phase 1 complete")

    def _verify_off_transition(self) -> None:
        """Phase 2: Set ss_temp_a=400 so both conditions are true => off wins."""
        self.log("Phase 2: Setting ss_temp_a=400")
        self._ss_temp_a = 400.0
        self._write_sensors()

        self.wait_for_eq("ss_stage_str", "off", is_virtual=True, timeout=5.0)
        self.wait_for_eq("ss_sim_stage", 3, is_virtual=True)
        self.wait_for_eq("ss_heater_cmd", 0, is_virtual=True)
        self.log("Phase 2 complete: first transition (=> off) took priority")

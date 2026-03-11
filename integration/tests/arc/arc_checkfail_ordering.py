#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Arc Checkfail Ordering Integration Test

Verifies that when multiple stage transition conditions (checkfails) are true
simultaneously, the first-written condition takes priority.

Phase 1: cf_temp_a=200, cf_temp_b=400
    Only the second condition is true (cf_temp_b > 300), so we loop on/pause.

Phase 2: set cf_temp_a=400
    Both conditions become true. The first condition (=> off) should win.
"""

import synnax as sy
from examples.simulators import PressSimDAQ

from tests.arc.arc_case import ArcConsoleCase

ARC_CHECKFAIL_SOURCE = """
cf_start_cmd => main

sequence main {
    stage on {
        0 -> cf_sim_stage,
        1 -> cf_heater_cmd,
        "on" -> cf_stage_str,
        interval{period=1s} -> (cf_temp_a > 290 and cf_temp_b > 290) => off,
        interval{period=1s} -> cf_temp_b > 300 => pause,
    }
    stage pause {
        2 -> cf_sim_stage,
        0 -> cf_heater_cmd,
        "pause" -> cf_stage_str,
        wait{duration=1s} => on,
    }
    stage off {
        3 -> cf_sim_stage,
        0 -> cf_heater_cmd,
        "off" -> cf_stage_str,
        cf_start_cmd => on,
    }
}
"""


class ArcCheckfailOrdering(ArcConsoleCase):
    """Test that first-written checkfail takes priority over later ones."""

    arc_source = ARC_CHECKFAIL_SOURCE
    arc_name_prefix = "ArcCheckfailOrdering"
    start_cmd_channel = "cf_start_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "cf_stage_str",
        "cf_sim_stage",
        "cf_heater_cmd",
        "cf_temp_a",
        "cf_temp_b",
        "end_test_cmd",
    ]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        client = self.client

        client.channels.create(
            name="cf_stage_str",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        for name in ["cf_sim_stage", "cf_heater_cmd"]:
            client.channels.create(
                name=name,
                data_type=sy.DataType.UINT8,
                virtual=True,
                retrieve_if_name_exists=True,
            )

        self.cf_sensor_time = client.channels.create(
            name="cf_sensor_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="cf_temp_a",
            index=self.cf_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="cf_temp_b",
            index=self.cf_sensor_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        super().setup()

    def verify_sequence_execution(self) -> None:
        with self.client.open_writer(
            start=sy.TimeStamp.now(),
            channels=["cf_sensor_time", "cf_temp_a", "cf_temp_b"],
            name="Checkfail Sensor Writer",
        ) as self._sensor_writer:
            self._cf_temp_a = 200.0
            self._cf_temp_b = 400.0
            self._write_sensors()

            self._verify_on_pause_loop()
            self._verify_off_transition()

    def _write_sensors(self) -> None:
        now = sy.TimeStamp.now()
        self._sensor_writer.write(
            {
                "cf_sensor_time": now,
                "cf_temp_a": self._cf_temp_a,
                "cf_temp_b": self._cf_temp_b,
            }
        )

    def _verify_on_pause_loop(self) -> None:
        """Phase 1: cf_temp_a=200, cf_temp_b=400 => only pause condition true.

        With 500ms intervals, the 'on' stage may transition to 'pause' before
        we can poll it. We verify the loop by observing 'pause' entries and
        continuously writing sensor values so the runtime receives them.
        """
        self.log("Phase 1: Verifying on/pause loop")
        for _ in range(3):
            self.wait_for_eq("cf_stage_str", "pause", is_virtual=True, timeout=10.0)
            self._write_sensors()
        self.log("Phase 1 complete")

    def _verify_off_transition(self) -> None:
        """Phase 2: Set cf_temp_a=400 so both conditions are true => off wins.

        We write sensors on every poll iteration so the runtime picks up the
        new values on the next 'on' entry, even if the on/pause cycle is fast.
        """
        self.log("Phase 2: Setting cf_temp_a=400")
        self._cf_temp_a = 400.0

        timer = sy.Timer()
        timeout = sy.TimeSpan.from_seconds(10)
        while self.should_continue:
            self._write_sensors()
            stage = self.read_tlm("cf_stage_str", "")
            if stage == "off":
                break
            if timer.elapsed() > timeout:
                self.fail(
                    f"Timeout waiting for cf_stage_str == off!\n"
                    f"Actual: {stage}\nTimeout: 10.0s"
                )
                return

        self.wait_for_eq("cf_sim_stage", 3, is_virtual=True)
        self.wait_for_eq("cf_heater_cmd", 0, is_virtual=True)
        self.log("Phase 2 complete: first checkfail (=> off) took priority")

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators import PressSimDAQ

import synnax as sy
from tests.arc.arc_case import ArcConsoleCase

ARC_STAGE_ROUTING = """

func decide_stage{low f64, high f64} (value f64) (vent u8, press u8, abort u8) {
    if (value < low) {
        vent = 1
    } else if (value <= high) {
        press = 1
    } else {
        abort = 1
    }
}

start_stage_routing_cmd => main

sequence main {
    stage select_hold {
        "select_hold" -> routing_stage_log,
        routing_flag -> select{} -> {
            true: select_on,
            false: select_off
        },
    }
    stage select_on {
        "select_on" -> routing_stage_log,
        next_cmd => select_hold
    }
    stage select_off {
        "select_off" -> routing_stage_log,
        next_cmd => decide_stage_hold
    }

    stage decide_stage_hold {
        "decide_stage_hold" -> routing_stage_log,
        routing_sensor -> decide_stage{low=30.0, high=80.0} -> {
            vent: decide_stage_vent,
            press: decide_stage_press,
            abort: decide_stage_abort
        },
    }
    stage decide_stage_vent {
        "decide_stage_vent" -> routing_stage_log,
        next_cmd => decide_stage_hold
    }
    stage decide_stage_press {
        "decide_stage_press" -> routing_stage_log,
        next_cmd => decide_stage_hold
    }
    stage decide_stage_abort {
        "decide_stage_abort" -> routing_stage_log,
        next_cmd => done
    }

    stage done {
        "done" -> routing_stage_log,
    }
}
"""


class StageRouting(ArcConsoleCase):
    """Test routing table with stage targets (SY-4045).

    Exercises both select{} and a custom multi-output function routing tables
    targeting stages within the same sequence.
    Phase 1 uses select to route based on a boolean flag.
    Phase 2 uses decide_stage to route to vent/press/abort based on sensor thresholds."""

    arc_source = ARC_STAGE_ROUTING
    arc_name_prefix = "StageRouting"
    start_cmd_channel = "start_stage_routing_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["routing_stage_log"]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        flag_idx = self.client.channels.create(
            name="routing_flag_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="routing_flag",
            data_type=sy.DataType.UINT8,
            index=flag_idx.key,
            retrieve_if_name_exists=True,
        )
        sensor_idx = self.client.channels.create(
            name="routing_sensor_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="routing_sensor",
            data_type=sy.DataType.FLOAT64,
            index=sensor_idx.key,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="routing_stage_log",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="next_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        super().setup()

    def _write_flag(self, value: int) -> None:
        with self.client.open_writer(
            sy.TimeStamp.now(), ["routing_flag_time", "routing_flag"]
        ) as w:
            w.write(
                {
                    "routing_flag_time": sy.TimeStamp.now(),
                    "routing_flag": value,
                }
            )

    def _write_sensor(self, value: float) -> None:
        with self.client.open_writer(
            sy.TimeStamp.now(), ["routing_sensor_time", "routing_sensor"]
        ) as w:
            w.write(
                {
                    "routing_sensor_time": sy.TimeStamp.now(),
                    "routing_sensor": value,
                }
            )

    def _advance(self) -> None:
        with self.client.open_writer(sy.TimeStamp.now(), "next_cmd") as w:
            w.write("next_cmd", 1)

    def verify_sequence_execution(self) -> None:
        # Phase 1: select routing
        self.log("[select] Sequence activated, expecting select_hold")
        self.wait_for_eq("routing_stage_log", "select_hold", is_virtual=True)

        self.log("[select] flag=1 -> select_on")
        self._write_flag(1)
        self.wait_for_eq("routing_stage_log", "select_on", is_virtual=True)

        self.log("[select] Advancing select_on -> select_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "select_hold", is_virtual=True)

        self.log("[select] flag=0 -> select_off")
        self._write_flag(0)
        self.wait_for_eq("routing_stage_log", "select_off", is_virtual=True)

        # Phase 2: decide_stage routing
        self.log("[decide_stage] Advancing select_off -> decide_stage_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "decide_stage_hold", is_virtual=True)

        self.log("[decide_stage] sensor=50.0 (mid range) -> decide_stage_press")
        self._write_sensor(50.0)
        self.wait_for_eq("routing_stage_log", "decide_stage_press", is_virtual=True)

        self.log("[decide_stage] Advancing decide_stage_press -> decide_stage_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "decide_stage_hold", is_virtual=True)

        self.log("[decide_stage] sensor=10.0 (below low) -> decide_stage_vent")
        self._write_sensor(10.0)
        self.wait_for_eq("routing_stage_log", "decide_stage_vent", is_virtual=True)

        self.log("[decide_stage] Advancing decide_stage_vent -> decide_stage_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "decide_stage_hold", is_virtual=True)

        self.log("[decide_stage] sensor=90.0 (above high) -> decide_stage_abort")
        self._write_sensor(90.0)
        self.wait_for_eq("routing_stage_log", "decide_stage_abort", is_virtual=True)

        self.log("[decide_stage] Advancing decide_stage_abort -> done")
        self._advance()
        self.wait_for_eq("routing_stage_log", "done", is_virtual=True)

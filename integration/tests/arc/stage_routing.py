#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import (
    create_indexed_pair,
    create_virtual_channel,
    create_virtual_channels,
)
from tests.arc.arc import ArcCase

ARC_STAGE_ROUTING = """

func decide_stage(low f64, high f64) {value f64} (vent u8, press u8, abort u8) {
    if (value < low) {
        vent = 1
    } else if (value <= high) {
        press = 1
    } else {
        abort = 1
    }
}

func multi_route() {value f64} (first u8, second u8, third u8, fourth u8) {
    if (value > 50.0) {
        first = 1
        third = 1
    } else {
        second = 1
        fourth = 1
    }
}

func combine() {a f64, b f64} f64 {
    return a + b
}

func scale(k f64) {level f64} f64 {
    return level * k
}

func classify() {a f64, b f64, c f64} (low u8, high u8) {
    if (a + b + c > 50.0) {
        high = 1
    } else {
        low = 1
    }
}

func encode() {lo f64, hi f64} f64 {
    return lo * 10.0 + hi
}

start_stage_routing_cmd => main

sequence main {
    stage select_hold {
        "select_hold" -> routing_stage_log
        routing_flag -> select() -> {
            true: true => select_on,
            false: true => select_off
        }
    }
    stage select_on {
        "select_on" -> routing_stage_log
        next_cmd => select_hold
    }
    stage select_off {
        "select_off" -> routing_stage_log
        next_cmd => decide_stage_hold
    }

    stage decide_stage_hold {
        "decide_stage_hold" -> routing_stage_log
        routing_sensor -> decide_stage(low=30.0, high=80.0) -> {
            vent: true => decide_stage_vent,
            press: true => decide_stage_press,
            abort: true => decide_stage_abort
        }
    }
    stage decide_stage_vent {
        "decide_stage_vent" -> routing_stage_log
        next_cmd => decide_stage_hold
    }
    stage decide_stage_press {
        "decide_stage_press" -> routing_stage_log
        next_cmd => decide_stage_hold
    }
    stage decide_stage_abort {
        "decide_stage_abort" -> routing_stage_log
        next_cmd => chain_body_hold
    }

    stage chain_body_hold {
        "chain_body_hold" -> routing_stage_log
        routing_flag -> select() -> {
            true: "chain_true" -> routing_stage_log,
            false: "chain_false" -> routing_stage_log
        }
        next_cmd => arrow_body_hold
    }

    stage arrow_body_hold {
        "arrow_body_hold" -> routing_stage_log
        routing_flag -> select() -> {
            true: true => arrow_on,
            false: true => arrow_off
        }
    }
    stage arrow_on {
        "arrow_on" -> routing_stage_log
        next_cmd => arrow_body_hold
    }
    stage arrow_off {
        "arrow_off" -> routing_stage_log
        next_cmd => multi_hold
    }

    stage multi_hold {
        "multi_hold" -> routing_stage_log
        routing_sensor -> multi_route() -> {
            first: "multi_1" -> routing_multi_log_1,
            second: "multi_2" -> routing_multi_log_2,
            third: "multi_3" -> routing_multi_log_3,
            fourth: "multi_4" -> routing_multi_log_4
        }
        next_cmd => input_hold
    }

    stage input_hold {
        "input_hold" -> routing_stage_log
        {
            routing_a: a,
            routing_b: b
        } -> combine() -> routing_sum
        next_cmd => mapped_hold
    }

    stage mapped_hold {
        "mapped_hold" -> routing_stage_log
        routing_sensor -> decide_stage(low=30.0, high=80.0) -> {
            vent: 1.0: level,
            press: 2.0 -> routing_level,
            abort: 3.0 -> routing_level
        } -> scale(k=10.0) -> routing_level
        next_cmd => pipeline_hold
    }

    stage pipeline_hold {
        "pipeline_hold" -> routing_stage_log
        {
            routing_a: a,
            routing_b: b,
            routing_c: c
        } -> classify() -> {
            low: 1.0: lo,
            high: 2.0: hi
        } -> encode() -> routing_code
        next_cmd => done
    }

    stage done {
        "done" -> routing_stage_log
    }
}
"""


class StageRouting(ArcCase):
    """Test routing table with stage targets (SY-4045, SY-4460).

    Exercises both select() and a custom multi-output function routing tables
    targeting stages within the same sequence.
    Phase 1 uses select to route based on a boolean flag.
    Phase 2 uses decide_stage to route to vent/press/abort based on sensor thresholds.
    Phase 3 uses select with multi-node case bodies that write string literals
    to a channel, verifying that case-body chains type-check via chain semantics
    rather than against the select's discriminator output.
    Phase 4 (SY-4460) uses select with `=>` transition case bodies, verifying that
    the transition operator drives stage jumps from inside a routing table.
    Phase 5 sets two of four outputs in one invocation, verifying that every set
    output fires its entry in the same tick.
    Phase 6 (SY-4075) feeds two triggers of one function from an input routing
    table, verifying that the function runs once both have a value and reuses the
    latest value of each.
    Phase 7 (SY-4075) maps one entry of an output routing table to a trigger of
    the function after the table with a `: param` suffix, next to entries that
    write a channel.
    Phase 8 (SY-4075) chains both in one flow: three sources into a two-output
    function, whose entries feed two triggers of the function after the table."""

    arc_source = ARC_STAGE_ROUTING
    arc_name_prefix = "StageRouting"
    start_cmd_channel = "start_stage_routing_cmd"
    subscribe_channels = [
        "routing_stage_log",
        "routing_multi_log_1",
        "routing_multi_log_2",
        "routing_multi_log_3",
        "routing_multi_log_4",
        "routing_sum",
        "routing_level",
        "routing_code",
    ]

    def setup(self) -> None:
        create_indexed_pair(self.client, "routing_flag", data_type=sy.DataType.BOOLEAN)
        create_indexed_pair(
            self.client, "routing_sensor", data_type=sy.DataType.FLOAT64
        )
        create_virtual_channel(
            self.client, "routing_stage_log", data_type=sy.DataType.STRING
        )
        create_virtual_channels(
            self.client,
            [(f"routing_multi_log_{i}", sy.DataType.STRING) for i in range(1, 5)],
        )
        create_virtual_channel(self.client, "next_cmd", data_type=sy.DataType.UINT8)
        create_indexed_pair(self.client, "routing_a", data_type=sy.DataType.FLOAT64)
        create_indexed_pair(self.client, "routing_b", data_type=sy.DataType.FLOAT64)
        create_indexed_pair(self.client, "routing_c", data_type=sy.DataType.FLOAT64)
        create_virtual_channels(
            self.client,
            [
                ("routing_sum", sy.DataType.FLOAT64),
                ("routing_level", sy.DataType.FLOAT64),
                ("routing_code", sy.DataType.FLOAT64),
            ],
        )
        super().setup()

    def _write_flag(self, value: int) -> None:
        self.writer.write(
            {"routing_flag_time": sy.TimeStamp.now(), "routing_flag": value}
        )

    def _write_sensor(self, value: float) -> None:
        self.writer.write(
            {"routing_sensor_time": sy.TimeStamp.now(), "routing_sensor": value}
        )

    def _write_input(self, name: str, value: float) -> None:
        self.writer.write({f"{name}_time": sy.TimeStamp.now(), name: value})

    def _advance(self) -> None:
        self.writer.write("next_cmd", 1)

    def verify_sequence_execution(self) -> None:
        # Phase 1: select routing
        self.log("[select] Sequence activated, expecting select_hold")
        self.wait_for_eq("routing_stage_log", "select_hold")

        self.log("[select] flag=1 -> select_on")
        self._write_flag(1)
        self.wait_for_eq("routing_stage_log", "select_on")

        self.log("[select] Advancing select_on -> select_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "select_hold")

        self.log("[select] flag=0 -> select_off")
        self._write_flag(0)
        self.wait_for_eq("routing_stage_log", "select_off")

        # Phase 2: decide_stage routing
        self.log("[decide_stage] Advancing select_off -> decide_stage_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "decide_stage_hold")

        self.log("[decide_stage] sensor=50.0 (mid range) -> decide_stage_press")
        self._write_sensor(50.0)
        self.wait_for_eq("routing_stage_log", "decide_stage_press")

        self.log("[decide_stage] Advancing decide_stage_press -> decide_stage_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "decide_stage_hold")

        self.log("[decide_stage] sensor=10.0 (below low) -> decide_stage_vent")
        self._write_sensor(10.0)
        self.wait_for_eq("routing_stage_log", "decide_stage_vent")

        self.log("[decide_stage] Advancing decide_stage_vent -> decide_stage_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "decide_stage_hold")

        self.log("[decide_stage] sensor=90.0 (above high) -> decide_stage_abort")
        self._write_sensor(90.0)
        self.wait_for_eq("routing_stage_log", "decide_stage_abort")

        self.log("[decide_stage] Advancing decide_stage_abort -> chain_body_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "chain_body_hold")

        # Phase 3: multi-node case body — `true: "literal" -> str_channel`
        self.log("[chain_body] flag=1 -> 'chain_true' written to log")
        self._write_flag(1)
        self.wait_for_eq("routing_stage_log", "chain_true")

        self.log("[chain_body] flag=0 -> 'chain_false' written to log")
        self._write_flag(0)
        self.wait_for_eq("routing_stage_log", "chain_false")

        self.log("[chain_body] Advancing chain_body_hold -> arrow_body_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "arrow_body_hold")

        # Phase 4: `=>` transition case body — `true: 1 => arrow_on`
        self.log("[arrow_body] flag=1 -> arrow_on via => transition")
        self._write_flag(1)
        self.wait_for_eq("routing_stage_log", "arrow_on")

        self.log("[arrow_body] Advancing arrow_on -> arrow_body_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "arrow_body_hold")

        self.log("[arrow_body] flag=0 -> arrow_off via => transition")
        self._write_flag(0)
        self.wait_for_eq("routing_stage_log", "arrow_off")

        self.log("[arrow_body] Advancing arrow_off -> multi_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "multi_hold")

        # Phase 5: one invocation sets two of four outputs; both entries fire.
        # Sentinel writes keep a wait_for_eq from passing on a stale value.
        for i in range(1, 5):
            self.writer.write(f"routing_multi_log_{i}", "zero_value")
            self.wait_for_eq(f"routing_multi_log_{i}", "zero_value")

        self.log("[multi] sensor=75.0 sets first+third")
        self._write_sensor(75.0)
        self.wait_for_eq("routing_multi_log_1", "multi_1")
        self.wait_for_eq("routing_multi_log_3", "multi_3")

        for i in range(1, 5):
            self.writer.write(f"routing_multi_log_{i}", "zero_value")
            self.wait_for_eq(f"routing_multi_log_{i}", "zero_value")

        self.log("[multi] sensor=10.0 sets second+fourth")
        self._write_sensor(10.0)
        self.wait_for_eq("routing_multi_log_2", "multi_2")
        self.wait_for_eq("routing_multi_log_4", "multi_4")

        self.log("[multi] Advancing multi_hold -> input_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "input_hold")

        # Phase 6: `{routing_a: a, routing_b: b} -> combine()` runs once both
        # triggers have a value, then on each new value with the other latched.
        self.log("[input] a=1.0, b=2.0 -> sum 3.0")
        self._write_input("routing_a", 1.0)
        self._write_input("routing_b", 2.0)
        self.wait_for_eq("routing_sum", 3.0)

        self.log("[input] a=5.0 with b latched -> sum 7.0")
        self._write_input("routing_a", 5.0)
        self.wait_for_eq("routing_sum", 7.0)

        self.log("[input] b=-1.0 with a latched -> sum 4.0")
        self._write_input("routing_b", -1.0)
        self.wait_for_eq("routing_sum", 4.0)

        self.log("[input] Advancing input_hold -> mapped_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "mapped_hold")

        # Phase 7: the vent entry's `: level` suffix feeds scale; the other two
        # entries write the channel themselves.
        self.log("[mapped] sensor=20.0 -> vent entry -> scale -> level 10.0")
        self._write_sensor(20.0)
        self.wait_for_eq("routing_level", 10.0)

        self.log("[mapped] sensor=50.0 -> press entry writes 2.0")
        self._write_sensor(50.0)
        self.wait_for_eq("routing_level", 2.0)

        self.log("[mapped] sensor=90.0 -> abort entry writes 3.0")
        self._write_sensor(90.0)
        self.wait_for_eq("routing_level", 3.0)

        self.log("[mapped] sensor=20.0 -> vent entry -> scale -> level 10.0 again")
        self._write_sensor(20.0)
        self.wait_for_eq("routing_level", 10.0)

        self.log("[mapped] Advancing mapped_hold -> pipeline_hold")
        self._advance()
        self.wait_for_eq("routing_stage_log", "pipeline_hold")

        # Phase 8: `{a, b, c} -> classify() -> {low: lo, high: hi} -> encode()`.
        # encode runs once both triggers have fired, then on each entry. A
        # sentinel write separates the runs, since each one writes 12.0.
        self.log("[pipeline] a=1, b=2, c=3 -> low -> lo only, no code yet")
        self._write_input("routing_a", 1.0)
        self._write_input("routing_b", 2.0)
        self._write_input("routing_c", 3.0)
        self.writer.write("routing_code", 0.0)
        self.wait_for_eq("routing_code", 0.0)

        self.log("[pipeline] c=100 -> high -> hi -> code 12.0")
        self._write_input("routing_c", 100.0)
        self.wait_for_eq("routing_code", 12.0)

        self.writer.write("routing_code", 0.0)
        self.wait_for_eq("routing_code", 0.0)
        self.log("[pipeline] a=-60 -> low -> lo again -> code 12.0")
        self._write_input("routing_a", -60.0)
        self.wait_for_eq("routing_code", 12.0)

        self.log("[pipeline] Advancing pipeline_hold -> done")
        self._advance()
        self.wait_for_eq("routing_stage_log", "done")

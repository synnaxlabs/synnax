#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from string import Template

import synnax as sy
from framework.utils import create_virtual_channels
from tests.arc.arc import ArcCase

# The exported sequence calls one that the module keeps private.
LIBRARY_SOURCE = """
offset f32 := 5

sequence send(out chan f32, value f32) {
    stage write {
        value + offset -> out
    }
}

export sequence relay(out chan f32, value f32, ready chan bool) {
    stage wait {
        true -> ready
        pb_mod_go => send(out, value)
    }
}

export stage hold(out chan f32, value f32) {
    value + offset -> out
}
"""

# pb_start starts every instance, and each instance marks its readiness on entry.
AUTOMATION_SOURCE = Template("""
import $library

authority 200

// ──── two instances ────
sequence fill(
    limit f32,
    level chan f32,
    valve chan bool,
    echo chan f32,
    ready chan bool
) {
    stage open {
        level + limit -> echo
        true -> valve
        true -> ready
        level > limit => next
    }
    stage close {
        false -> valve
    }
}
pb_start == 1 => fill(
    limit=10,
    level=pb_fill_level_a,
    valve=pb_fill_valve_a,
    echo=pb_fill_echo_a,
    ready=pb_fill_ready_a
)
pb_start == 1 => fill(
    limit=20,
    level=pb_fill_level_b,
    valve=pb_fill_valve_b,
    echo=pb_fill_echo_b,
    ready=pb_fill_ready_b
)

// ──── defaults and names ────
sequence set(out chan f32, value f32 = 7.5) {
    stage write {
        value -> out
    }
}
pb_start == 1 => set(out=pb_set_default)
pb_start == 1 => set(value=-2.5, out=pb_set_named)

// ──── top-level constant and alias ────
threshold f32 := 5
sink := pb_set_const
pb_start == 1 => set(sink, threshold)

// ──── top-level stage ────
stage mark(out chan bool) {
    true -> out
}
pb_start == 1 => mark(pb_mark_a)
pb_start == 1 => mark(pb_mark_b)

// ──── variables and expressions ────
sequence scale(
    src chan f32,
    gain f32,
    id i64,
    out chan f32,
    text chan str,
    ready chan bool
) {
    doubled := src * 2
    stage run {
        true -> ready
        doubled + gain -> out
        f"id {id}" -> text
    }
}
pb_start == 1 => scale(pb_scale_src, 1, 1, pb_scale_a, pb_text_a, pb_scale_ready_a)
pb_start == 1 => scale(pb_scale_src, 10, 2, pb_scale_b, pb_text_b, pb_scale_ready_b)

// ──── function input ────
func add(offset f32) {value f32} f32 {
    return value + offset
}
sequence shift(offset f32, src chan f32, out chan f32, ready chan bool) {
    stage run {
        true -> ready
        src -> add(offset=offset) -> out
    }
}
pb_start == 1 => shift(1, pb_shift_src, pb_shift_a, pb_shift_ready_a)
pb_start == 1 => shift(100, pb_shift_src, pb_shift_b, pb_shift_ready_b)

// ──── call from an instance ────
sequence pulse(valve chan bool) {
    stage on {
        true -> valve
        pb_stop => safe
    }
}
sequence arm(valve chan bool, ready chan bool) {
    stage wait {
        true -> ready
        pb_go => pulse(valve)
    }
}
sequence safe {
    stage all {
        true -> pb_safe
    }
}
pb_start == 1 => arm(pb_pulse_a, pb_arm_ready_a)
pb_start == 1 => arm(pb_pulse_b, pb_arm_ready_b)

// ──── inline body ────
sequence watch(level chan f32, out chan bool, ready chan bool) {
    stage armed {
        true -> ready
        level > 5 => stage {
            true -> out
        }
    }
}
pb_start == 1 => watch(pb_watch_level_a, pb_watch_a, pb_watch_ready_a)
pb_start == 1 => watch(pb_watch_level_b, pb_watch_b, pb_watch_ready_b)

// ──── module ────
pb_start == 1 => $library.relay(pb_mod_a, 1, pb_mod_ready_a)
pb_start == 1 => $library.relay(pb_mod_b, 2, pb_mod_ready_b)
pb_start == 1 => $library.hold(pb_mod_hold_a, 3)
pb_start == 1 => $library.hold(pb_mod_hold_b, 4)
""")

BOOL = sy.DataType.BOOLEAN
F32 = sy.DataType.FLOAT32
STR = sy.DataType.STRING

# The case writes these channels, and the automation reads them.
INPUTS: dict[str, sy.DataType] = {
    "pb_fill_level_a": F32,
    "pb_fill_level_b": F32,
    "pb_scale_src": F32,
    "pb_shift_src": F32,
    "pb_go": BOOL,
    "pb_stop": BOOL,
    "pb_watch_level_a": F32,
    "pb_watch_level_b": F32,
    "pb_mod_go": BOOL,
}

# The automation writes these channels, and the case reads them.
OUTPUTS: dict[str, sy.DataType] = {
    "pb_fill_ready_a": BOOL,
    "pb_fill_ready_b": BOOL,
    "pb_fill_valve_a": BOOL,
    "pb_fill_valve_b": BOOL,
    "pb_fill_echo_a": F32,
    "pb_fill_echo_b": F32,
    "pb_set_default": F32,
    "pb_set_named": F32,
    "pb_set_const": F32,
    "pb_mark_a": BOOL,
    "pb_mark_b": BOOL,
    "pb_scale_ready_a": BOOL,
    "pb_scale_ready_b": BOOL,
    "pb_scale_a": F32,
    "pb_scale_b": F32,
    "pb_text_a": STR,
    "pb_text_b": STR,
    "pb_shift_ready_a": BOOL,
    "pb_shift_ready_b": BOOL,
    "pb_shift_a": F32,
    "pb_shift_b": F32,
    "pb_arm_ready_a": BOOL,
    "pb_arm_ready_b": BOOL,
    "pb_pulse_a": BOOL,
    "pb_pulse_b": BOOL,
    "pb_safe": BOOL,
    "pb_watch_ready_a": BOOL,
    "pb_watch_ready_b": BOOL,
    "pb_watch_a": BOOL,
    "pb_watch_b": BOOL,
    "pb_mod_ready_a": BOOL,
    "pb_mod_ready_b": BOOL,
    "pb_mod_a": F32,
    "pb_mod_b": F32,
    "pb_mod_hold_a": F32,
    "pb_mod_hold_b": F32,
}


class ParameterizedBodies(ArcCase):
    """Each call of a sequence or stage with parameters runs as its own instance.

    1. Two calls of one sequence run at the same time. Each one has its own value
       argument, its own channels, and its own active stage.
    2. A call leaves out a parameter that has a default, and a call names its
       arguments.
    3. A call takes a top-level constant and a top-level channel alias.
    4. A top-level stage takes a channel parameter.
    5. The body reads its parameters in a variable, an expression, and a format
       string.
    6. The body passes a value parameter to a function input.
    7. An instance calls a second sequence with its own parameters. That instance
       jumps to a sequence with no parameters.
    8. An inline body in the sequence belongs to one instance.
    9. The automation calls a sequence and a stage that a module exports, two times
       each. The sequence calls one that the module does not export.
    """

    arc_source = AUTOMATION_SOURCE.template
    arc_name_prefix = "ArcParameterizedBodies"
    start_cmd_channel = "pb_start"
    subscribe_channels = list(OUTPUTS)

    def setup(self) -> None:
        create_virtual_channels(self.client, list((INPUTS | OUTPUTS).items()))
        super().setup()
        self.set_manual_timeout(60)

    def run(self) -> None:
        self._retrieve_rack()
        library = self.load_module(LIBRARY_SOURCE, name_prefix="pb_library")
        self.arc_name = self.load_arc(
            AUTOMATION_SOURCE.substitute(library=library),
            name_prefix=self.arc_name_prefix,
            trigger=self.start_cmd_channel,
        )
        self.verify_sequence_execution()

    def verify_sequence_execution(self) -> None:
        self._verify_two_instances()
        self._verify_defaults_and_names()
        self._verify_top_level_stage()
        self._verify_variables_and_expressions()
        self._verify_function_input()
        self._verify_call_from_an_instance()
        self._verify_inline_body()
        self._verify_module()

    def _wait_for_ready(self, *channels: str) -> None:
        for channel in channels:
            self.wait_for_eq(channel, 1)

    def _verify_two_instances(self) -> None:
        self.log("Waiting for both fill instances to open their valves")
        self._wait_for_ready("pb_fill_ready_a", "pb_fill_ready_b")
        self.wait_for_eq("pb_fill_valve_a", 1)
        self.wait_for_eq("pb_fill_valve_b", 1)
        self.log("Each instance must add its own limit to its level")
        self.writer.write("pb_fill_level_a", 5.0)
        self.wait_for_eq("pb_fill_echo_a", 15)
        self.writer.write("pb_fill_level_b", 15.0)
        self.wait_for_eq("pb_fill_echo_b", 35)
        self.log("Instance B must stay open above the limit of instance A")
        self.writer.write("pb_fill_level_b", 16.0)
        self.wait_for_eq("pb_fill_echo_b", 36)
        self.log("Driving level A above 10 to close the valve of instance A")
        self.writer.write("pb_fill_level_a", 15.0)
        self.wait_for_eq("pb_fill_valve_a", 0)
        self.log("Instance B must stay open after instance A closes")
        self.writer.write("pb_fill_level_b", 17.0)
        self.wait_for_eq("pb_fill_echo_b", 37)
        self.log("Driving level B above 20 to close the valve of instance B")
        self.writer.write("pb_fill_level_b", 25.0)
        self.wait_for_eq("pb_fill_valve_b", 0)

    def _verify_defaults_and_names(self) -> None:
        self.log("Waiting for the default, the named, and the constant values")
        self.wait_for_eq("pb_set_default", 7.5)
        self.wait_for_eq("pb_set_named", -2.5)
        self.wait_for_eq("pb_set_const", 5)

    def _verify_top_level_stage(self) -> None:
        self.log("Waiting for both instances of the top-level stage")
        self.wait_for_eq("pb_mark_a", 1)
        self.wait_for_eq("pb_mark_b", 1)

    def _verify_variables_and_expressions(self) -> None:
        self._wait_for_ready("pb_scale_ready_a", "pb_scale_ready_b")
        self.log("Each instance must format its own id")
        self.wait_for_eq("pb_text_a", "id 1")
        self.wait_for_eq("pb_text_b", "id 2")
        self.log("Writing pb_scale_src=3 for each instance to double and add its gain")
        self.writer.write("pb_scale_src", 3.0)
        self.wait_for_eq("pb_scale_a", 7)
        self.wait_for_eq("pb_scale_b", 16)

    def _verify_function_input(self) -> None:
        self._wait_for_ready("pb_shift_ready_a", "pb_shift_ready_b")
        self.log("Writing pb_shift_src=5 for each instance to add its own offset")
        self.writer.write("pb_shift_src", 5.0)
        self.wait_for_eq("pb_shift_a", 6)
        self.wait_for_eq("pb_shift_b", 105)

    def _verify_call_from_an_instance(self) -> None:
        self._wait_for_ready("pb_arm_ready_a", "pb_arm_ready_b")
        self.log("Writing pb_go=1 for each arm instance to call pulse with its valve")
        self.writer.write("pb_go", 1)
        self.wait_for_eq("pb_pulse_a", 1)
        self.wait_for_eq("pb_pulse_b", 1)
        self.log("Writing pb_stop=1 for the pulse instances to jump to safe")
        self.writer.write("pb_stop", 1)
        self.wait_for_eq("pb_safe", 1)

    def _verify_inline_body(self) -> None:
        self._wait_for_ready("pb_watch_ready_a", "pb_watch_ready_b")
        self.log("Driving watch level A above 5 to run the inline body of instance A")
        self.writer.write("pb_watch_level_a", 10.0)
        self.wait_for_eq("pb_watch_a", 1)
        if self.read_tlm("pb_watch_b") is not None:
            self.fail("Instance B ran its inline body on the level of instance A")
        self.log("Driving watch level B above 5 to run the inline body of instance B")
        self.writer.write("pb_watch_level_b", 10.0)
        self.wait_for_eq("pb_watch_b", 1)

    def _verify_module(self) -> None:
        self._wait_for_ready("pb_mod_ready_a", "pb_mod_ready_b")
        self.log("Writing pb_mod_go=1 for each relay instance to send its value plus 5")
        self.writer.write("pb_mod_go", 1)
        self.wait_for_eq("pb_mod_a", 6)
        self.wait_for_eq("pb_mod_b", 7)
        self.log("Waiting for each instance of the exported stage")
        self.wait_for_eq("pb_mod_hold_a", 8)
        self.wait_for_eq("pb_mod_hold_b", 9)

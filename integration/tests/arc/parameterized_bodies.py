#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass
from itertools import product
from string import Template

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channels
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

# No flow calls the sequence, and it starts a sequence that has no steps.
UNCALLED_SOURCE = """
sequence empty {}

export sequence drain(valve chan bool) {
    stage open {
        true -> valve
        pb_mod_go => empty
    }
}
"""

# A channel parameter has a default value, in a sequence and in a stage.
CHANNEL_DEFAULT_SOURCES = {
    "sequence": """
export sequence hold(valve chan bool = false) {
    stage open {
        true -> valve
    }
}
""",
    "stage": """
export stage hold(valve chan bool = false) {
    true -> valve
}
""",
}

# The text in front of a use for each way to start it.
STARTS = {"bare": "", "true": "true => ", "trigger": "pb_matrix_go => "}


def _marks(kind: str, out: str) -> str:
    """Returns the braces of a stage or sequence that marks out when it starts."""
    if kind == "stage":
        return f"{{\n    true -> {out}\n}}"
    return f"{{\n    stage run {{\n        true -> {out}\n    }}\n}}"


@dataclass(frozen=True)
class Use:
    """One use of a stage or sequence in a stage or sequence that holds it."""

    kind: str
    form: str
    holder: str
    start: str

    @property
    def channel(self) -> str:
        """The channel that the body marks when it starts."""
        return f"pb_m_{self.kind}_{self.form}_{self.holder}_{self.start}"

    @property
    def after(self) -> str:
        """The channel that the step after the body marks."""
        return f"{self.channel}_after"

    @property
    def body(self) -> str:
        """The name of the body with no parameters that this use has for its own."""
        return self.channel.removeprefix("pb_")

    @property
    def line(self) -> str:
        """The line of the holder that uses the body."""
        text = {
            "named": self.body,
            "call": f"mark_{self.kind}({self.channel})",
            "anonymous": f"{self.kind} {_marks(self.kind, self.channel)}",
        }[self.form]
        return STARTS[self.start] + text


MATRIX = [
    Use(*vector)
    for vector in product(
        ("stage", "sequence"),
        ("named", "call", "anonymous"),
        ("stage", "sequence"),
        STARTS,
    )
]


def _matrix_source() -> str:
    """Returns the Arc source that holds each use of the matrix."""
    parts = [
        "stage mark_stage(out chan bool) " + _marks("stage", "out"),
        "sequence mark_sequence(out chan bool) " + _marks("sequence", "out"),
    ]
    parts += [
        f"{use.kind} {use.body} {_marks(use.kind, use.channel)}"
        for use in MATRIX
        if use.form == "named"
    ]
    held = "\n".join(use.line for use in MATRIX if use.holder == "stage")
    parts.append(
        "stage matrix {\npb_matrix_probe -> pb_matrix_alive\n"
        f"{held}\n}}\npb_start => matrix"
    )
    parts += [
        f"sequence {use.body}_holder {{\n{use.line}\ntrue -> {use.after}\n}}\n"
        f"pb_start => {use.body}_holder"
        for use in MATRIX
        if use.holder == "sequence"
    ]
    return "\n".join(parts)


# pb_start starts every instance, and each instance marks its readiness on entry.
AUTOMATION_SOURCE = Template("""
import $library
import control
import time

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
pb_start => fill(
    limit=10,
    level=pb_fill_level_a,
    valve=pb_fill_valve_a,
    echo=pb_fill_echo_a,
    ready=pb_fill_ready_a
)
pb_start => fill(
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
pb_start => set(out=pb_set_default)
pb_start => set(value=-2.5, out=pb_set_named)

// ──── top-level constant and alias ────
threshold f32 := 5
sink := pb_set_const
pb_start => set(sink, threshold)

// ──── top-level stage ────
stage mark(out chan bool) {
    true -> out
}
pb_start => mark(pb_mark_a)
pb_start => mark(pb_mark_b)

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
pb_start => scale(pb_scale_src, 1, 1, pb_scale_a, pb_text_a, pb_scale_ready_a)
pb_start => scale(pb_scale_src, 10, 2, pb_scale_b, pb_text_b, pb_scale_ready_b)

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
pb_start => shift(1, pb_shift_src, pb_shift_a, pb_shift_ready_a)
pb_start => shift(100, pb_shift_src, pb_shift_b, pb_shift_ready_b)

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
pb_start => arm(pb_pulse_a, pb_arm_ready_a)
pb_start => arm(pb_pulse_b, pb_arm_ready_b)

// ──── anonymous body ────
sequence watch(level chan f32, out chan bool, ready chan bool) {
    stage armed {
        true -> ready
        level > 5 => stage {
            true -> out
        }
    }
}
pb_start => watch(pb_watch_level_a, pb_watch_a, pb_watch_ready_a)
pb_start => watch(pb_watch_level_b, pb_watch_b, pb_watch_ready_b)

// ──── variable and expression arguments ────
sequence lift(base f32, src chan f32, out chan f32, ready chan bool) {
    stage run {
        true -> ready
        src + base -> out
    }
}
sequence lift_variable {
    amount f32 := 1
    stage wait {
        10 -> amount
        true -> pb_live_armed_a
        pb_live_go => lift(amount, pb_live_src, pb_live_a, pb_live_ready_a)
    }
}
sequence lift_expression {
    amount f32 := 1
    stage wait {
        10 -> amount
        true -> pb_live_armed_b
        pb_live_go => lift(amount * 2, pb_live_src, pb_live_b, pb_live_ready_b)
    }
}
computed f32 := 2.5 * 2.0
pb_start => lift_variable
pb_start => lift_expression
pb_start => lift(computed, pb_live_src, pb_live_c, pb_live_ready_c)

// ──── reassigned variable ────
sequence rescale_seq(src chan f32, gain f32, out chan f32, ready chan bool) {
    x := src * gain
    stage first {
        true -> ready
        x -> out
        pb_rescale_go => next
    }
    stage second {
        x = src * gain * 2
        false -> ready
        x -> out
    }
}
stage rescale_stage(src chan f32, gain f32, out chan f32, ready chan bool) {
    x := src * gain
    x = src * gain * 2
    true -> ready
    x -> out
}
pb_start => rescale_seq(
    pb_rescale_src,
    3,
    pb_rescale_seq_a,
    pb_rescale_seq_ready_a
)
pb_start => rescale_seq(
    pb_rescale_src,
    5,
    pb_rescale_seq_b,
    pb_rescale_seq_ready_b
)
pb_start => rescale_stage(
    pb_rescale_src,
    3,
    pb_rescale_stage_a,
    pb_rescale_stage_ready_a
)
pb_start => rescale_stage(
    pb_rescale_src,
    5,
    pb_rescale_stage_b,
    pb_rescale_stage_ready_b
)

// ──── parameter with a unit ────
func long(x i64 ns) {tick bool} bool {
    return x > 1s
}
sequence unit_seq(d i64 ms, out chan bool, ready chan bool) {
    stage run {
        true -> ready
        pb_unit_go -> long(x=d) -> out
    }
}
stage unit_stage(d i64 ms, out chan bool, ready chan bool) {
    true -> ready
    pb_unit_go -> long(x=d) -> out
}
pb_start => unit_seq(2s, pb_unit_seq, pb_unit_seq_ready)
pb_start => unit_stage(2s, pb_unit_stage, pb_unit_stage_ready)

// ──── wait for a parameter ────
sequence wait_seq(d i64 ms, out chan bool, ready chan bool) {
    stage hold {
        true -> ready
        time.wait(duration=d) => next
    }
    stage open {
        true -> out
    }
}
stage wait_open(out chan bool) {
    true -> out
}
stage wait_stage(d i64 ms, out chan bool, ready chan bool) {
    true -> ready
    time.wait(duration=d) => wait_open(out)
}
pb_start => wait_seq(1s, pb_wait_seq_end, pb_wait_seq_begin)
pb_start => wait_stage(1s, pb_wait_stage_end, pb_wait_stage_begin)

// ──── module ────
pb_start => $library.relay(pb_mod_a, 1, pb_mod_ready_a)
pb_start => $library.relay(pb_mod_b, 2, pb_mod_ready_b)
pb_start => $library.hold(pb_mod_hold_a, 3)
pb_start => $library.hold(pb_mod_hold_b, 4)

// ──── authority ────
stage take_stage(level u8, low chan bool, high chan bool, other chan bool) {
    control.set_authority(value=level, channel=low)
    control.set_authority(value=level, channel=high)
    true -> low
    true -> high
    true -> other
}
sequence take_seq(level u8, low chan bool, high chan bool, other chan bool) {
    stage run {
        control.set_authority(value=level, channel=low)
        control.set_authority(value=level, channel=high)
        true -> low
        true -> high
        true -> other
    }
}
pb_auth_go => take_stage(
    230,
    pb_auth_stage_low_a,
    pb_auth_stage_high_a,
    pb_auth_stage_other_a
)
pb_auth_go => take_stage(
    250,
    pb_auth_stage_low_b,
    pb_auth_stage_high_b,
    pb_auth_stage_other_b
)
pb_auth_go => take_seq(
    230,
    pb_auth_seq_low_a,
    pb_auth_seq_high_a,
    pb_auth_seq_other_a
)
pb_auth_go => take_seq(
    250,
    pb_auth_seq_low_b,
    pb_auth_seq_high_b,
    pb_auth_seq_other_b
)

// ──── copies at the top level ────
stage note_stage {
    true -> pb_copy_stage
}
sequence note_sequence {
    stage run {
        true -> pb_copy_sequence
    }
}
pb_copy_a => note_stage
pb_copy_b => note_stage
pb_copy_a => note_sequence
pb_copy_b => note_sequence

// ──── each form, holder, and start ────
$matrix
""")

BOOL = sy.DataType.BOOLEAN
F32 = sy.DataType.FLOAT32
STR = sy.DataType.STRING

# The case writes these channels, and the automation reads them.
INPUTS: dict[str, sy.DataType] = {
    "pb_start": BOOL,
    "pb_fill_level_a": F32,
    "pb_fill_level_b": F32,
    "pb_scale_src": F32,
    "pb_shift_src": F32,
    "pb_go": BOOL,
    "pb_stop": BOOL,
    "pb_watch_level_a": F32,
    "pb_watch_level_b": F32,
    "pb_live_go": BOOL,
    "pb_live_src": F32,
    "pb_mod_go": BOOL,
    "pb_auth_go": BOOL,
    "pb_unit_go": BOOL,
    "pb_rescale_src": F32,
    "pb_rescale_go": BOOL,
    "pb_copy_a": BOOL,
    "pb_copy_b": BOOL,
    "pb_matrix_go": BOOL,
    "pb_matrix_probe": BOOL,
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
    "pb_live_armed_a": BOOL,
    "pb_live_armed_b": BOOL,
    "pb_live_ready_a": BOOL,
    "pb_live_ready_b": BOOL,
    "pb_live_ready_c": BOOL,
    "pb_live_a": F32,
    "pb_live_b": F32,
    "pb_live_c": F32,
    "pb_rescale_seq_ready_a": BOOL,
    "pb_rescale_seq_ready_b": BOOL,
    "pb_rescale_stage_ready_a": BOOL,
    "pb_rescale_stage_ready_b": BOOL,
    "pb_rescale_seq_a": F32,
    "pb_rescale_seq_b": F32,
    "pb_rescale_stage_a": F32,
    "pb_rescale_stage_b": F32,
    "pb_unit_seq_ready": BOOL,
    "pb_unit_stage_ready": BOOL,
    "pb_unit_seq": BOOL,
    "pb_unit_stage": BOOL,
    "pb_mod_ready_a": BOOL,
    "pb_mod_ready_b": BOOL,
    "pb_mod_a": F32,
    "pb_mod_b": F32,
    "pb_mod_hold_a": F32,
    "pb_mod_hold_b": F32,
    "pb_matrix_alive": BOOL,
}
OUTPUTS |= {use.channel: BOOL for use in MATRIX}
OUTPUTS |= {use.after: BOOL for use in MATRIX if use.holder == "sequence"}

# The case holds the low and other channels at 220 and the high channels at 240, above
# the automation at 200. Each name is pb_auth_<stage or seq>_<channel>_<instance>.
HELD: dict[str, int] = {
    f"pb_auth_{body}_{channel}_{instance}": 240 if channel == "high" else 220
    for body in ("stage", "seq")
    for channel in ("low", "high", "other")
    for instance in ("a", "b")
}
OUTPUTS |= dict.fromkeys(HELD, BOOL)

# Each of these has an index, so the Driver stamps the samples that time a wait.
STAMPED: list[str] = [
    f"pb_wait_{body}_{edge}" for body in ("seq", "stage") for edge in ("begin", "end")
]

# Each of these has an index, so the case can count the samples that the copies wrote.
COUNTED: list[str] = ["pb_copy_stage", "pb_copy_sequence"]


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
       starts a sequence with no parameters by its name.
    8. An anonymous body in the sequence belongs to one instance.
    9. The automation calls a sequence and a stage that a module exports, two times
       each. The sequence calls one that the module does not export.
    10. Two calls of a stage and two calls of a sequence set their own authority on
        two channel parameters each. Each call takes only the channels that its
        authority is above.
    11. A module does not compile when a sequence that no flow calls has an error.
    12. A module does not compile when a channel parameter of a sequence or of a
        stage has a default value.
    13. A call takes a variable that its sequence writes, an expression over that
        variable, and a top-level variable that is computed at program start.
    14. A sequence and a stage reassign a variable that reads a channel parameter
        and a value parameter. Each instance keeps its own arguments.
    15. A sequence and a stage pass a parameter in milliseconds to a function input
        in nanoseconds. The function gets the argument of 2 s as more than 1 s.
    16. A sequence and a stage wait for a parameter in milliseconds that gets an
        argument of 1 s. The Driver stamps a sample before and after the wait, and
        the case fails if the two are less than 500 ms apart.
    17. Two top-level flows start one stage, and two start one sequence. Each flow
        starts its own copy.
    18. A stage and a sequence hold each kind of body: a stage or a sequence, with
        a name, with parameters, or anonymous. Each one starts with no trigger, on
        true, and on a trigger. The holder keeps running.
    """

    arc_source = AUTOMATION_SOURCE.template
    arc_name_prefix = "ArcParameterizedBodies"
    start_cmd_channel = "pb_start"
    subscribe_channels = [*OUTPUTS, *STAMPED, *COUNTED]
    start: sy.TimeStamp

    def setup(self) -> None:
        create_virtual_channels(self.client, list((INPUTS | OUTPUTS).items()))
        for name in [*STAMPED, *COUNTED]:
            create_indexed_pair(self.client, name, data_type=BOOL)
        self.start = sy.TimeStamp.now()
        super().setup()
        self.set_manual_timeout(60)

    def run(self) -> None:
        self._retrieve_rack()
        library = self.load_module(LIBRARY_SOURCE, name_prefix="pb_library")
        self.arc_name = self.load_arc(
            AUTOMATION_SOURCE.substitute(library=library, matrix=_matrix_source()),
            name_prefix=self.arc_name_prefix,
            trigger=self.start_cmd_channel,
        )
        self.verify_sequence_execution()

    def verify_sequence_execution(self) -> None:
        # The modules compile, and the instances wait, while the other checks run.
        self.log("Creating the modules that must not compile")
        uncalled = self.load_module(UNCALLED_SOURCE, name_prefix="pb_uncalled")
        defaults = {
            body: self.load_module(source, name_prefix=f"pb_default_{body}")
            for body, source in CHANNEL_DEFAULT_SOURCES.items()
        }
        self._verify_two_instances()
        self._verify_defaults_and_names()
        self._verify_top_level_stage()
        self._verify_variables_and_expressions()
        self._verify_function_input()
        self._verify_call_from_an_instance()
        self._verify_anonymous_body()
        self._verify_variable_arguments()
        self._verify_reassigned_variable()
        self._verify_parameter_unit()
        self._verify_module()
        self._verify_authority()
        self._verify_uncalled_declaration(uncalled)
        self._verify_channel_default(defaults)
        self._verify_parameter_wait()
        self._verify_copies_at_the_top_level()
        self._verify_matrix()

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
        self.log("Writing pb_stop=1 for the pulse instances to start safe")
        self.writer.write("pb_stop", 1)
        self.wait_for_eq("pb_safe", 1)

    def _verify_anonymous_body(self) -> None:
        self._wait_for_ready("pb_watch_ready_a", "pb_watch_ready_b")
        self.log(
            "Driving watch level A above 5 to run the anonymous body of instance A"
        )
        self.writer.write("pb_watch_level_a", 10.0)
        self.wait_for_eq("pb_watch_a", 1)
        if self.read_tlm("pb_watch_b") is not None:
            self.fail("Instance B ran its anonymous body on the level of instance A")
        self.log(
            "Driving watch level B above 5 to run the anonymous body of instance B"
        )
        self.writer.write("pb_watch_level_b", 10.0)
        self.wait_for_eq("pb_watch_b", 1)

    def _verify_variable_arguments(self) -> None:
        self._wait_for_ready("pb_live_armed_a", "pb_live_armed_b")
        self.log("Writing pb_live_go=1 for each sequence to call lift")
        self.writer.write("pb_live_go", 1)
        self._wait_for_ready("pb_live_ready_a", "pb_live_ready_b", "pb_live_ready_c")
        self.log("Writing pb_live_src=1 for each instance to add its argument")
        self.writer.write("pb_live_src", 1.0)
        self.wait_for_eq("pb_live_a", 11)
        self.wait_for_eq("pb_live_b", 21)
        self.wait_for_eq("pb_live_c", 6)

    def _verify_reassigned_variable(self) -> None:
        self._wait_for_ready(
            "pb_rescale_seq_ready_a",
            "pb_rescale_seq_ready_b",
            "pb_rescale_stage_ready_a",
            "pb_rescale_stage_ready_b",
        )
        self.log("Writing pb_rescale_src=10 for each instance to scale by its gain")
        self.writer.write("pb_rescale_src", 10.0)
        self.wait_for_eq("pb_rescale_seq_a", 30)
        self.wait_for_eq("pb_rescale_seq_b", 50)
        self.log("Each stage instance must use the value its body assigns last")
        self.wait_for_eq("pb_rescale_stage_a", 60)
        self.wait_for_eq("pb_rescale_stage_b", 100)
        self.log("Writing pb_rescale_go=1 for each sequence to reassign its variable")
        self.writer.write("pb_rescale_go", 1)
        self.wait_for_eq("pb_rescale_seq_ready_a", 0)
        self.wait_for_eq("pb_rescale_seq_ready_b", 0)
        self.writer.write("pb_rescale_src", 20.0)
        self.wait_for_eq("pb_rescale_seq_a", 120)
        self.wait_for_eq("pb_rescale_seq_b", 200)

    def _verify_parameter_unit(self) -> None:
        self._wait_for_ready("pb_unit_seq_ready", "pb_unit_stage_ready")
        self.log("Writing pb_unit_go=1 for each instance to compare its 2 s to 1 s")
        self.writer.write("pb_unit_go", 1)
        self.wait_for_eq("pb_unit_seq", 1)
        self.wait_for_eq("pb_unit_stage", 1)

    def _verify_parameter_wait(self) -> None:
        self.log("Each instance must write 1 s after it starts its wait")
        span = sy.TimeRange(self.start, sy.TimeStamp.MAX)
        for body in ("seq", "stage"):
            begin, end = f"pb_wait_{body}_begin_time", f"pb_wait_{body}_end_time"
            self.wait_for_eq(f"pb_wait_{body}_end", 1)
            frame = self.client.read(span, [begin, end])
            held = sy.TimeSpan(
                int(frame[end].to_numpy()[0]) - int(frame[begin].to_numpy()[0])
            )
            if held < 500 * sy.TimeSpan.MILLISECOND:
                self.fail(f"pb_wait_{body} held its wait of 1 s for only {held}")

    def _verify_module(self) -> None:
        self._wait_for_ready("pb_mod_ready_a", "pb_mod_ready_b")
        self.log("Writing pb_mod_go=1 for each relay instance to send its value plus 5")
        self.writer.write("pb_mod_go", 1)
        self.wait_for_eq("pb_mod_a", 6)
        self.wait_for_eq("pb_mod_b", 7)
        self.log("Waiting for each instance of the exported stage")
        self.wait_for_eq("pb_mod_hold_a", 8)
        self.wait_for_eq("pb_mod_hold_b", 9)

    def _verify_authority(self) -> None:
        self.log("Holding the authority channels above the automation")
        # A retrieve by name does not keep the order of the names.
        by_name = {ch.name: ch.key for ch in self.client.channels.retrieve(list(HELD))}
        keys = [by_name[name] for name in HELD]
        with self.client.open_writer(
            sy.TimeStamp.now(), keys, authorities=list(HELD.values())
        ) as override:
            override.write(dict.fromkeys(keys, 0))
            for channel in HELD:
                self.wait_for_eq(channel, 0)
            self.log("Writing pb_auth_go=1 for each instance to set its authority")
            self.writer.write("pb_auth_go", 1)
            for body in ("stage", "seq"):
                self.log(f"The {body} instance at 250 must take both of its channels")
                self.wait_for_eq(f"pb_auth_{body}_low_b", 1)
                self.wait_for_eq(f"pb_auth_{body}_high_b", 1)
                self.log(f"The {body} instance at 230 must take only its low channel")
                self.wait_for_eq(f"pb_auth_{body}_low_a", 1)
                for channel in ("high_a", "other_a", "other_b"):
                    if self.read_tlm(f"pb_auth_{body}_{channel}") != 0:
                        self.fail(f"The {body} wrote pb_auth_{body}_{channel}")

    def _verify_uncalled_declaration(self, path: str) -> None:
        self.log("The module with an error in a sequence that no flow calls")
        status = self.wait_for_module_status(path, "error", 10 * sy.TimeSpan.SECOND)
        assert "sequence 'empty' has no steps" in status.description, status

    def _verify_channel_default(self, paths: dict[str, str]) -> None:
        for body, path in paths.items():
            self.log(f"The module with a channel default in a {body}")
            status = self.wait_for_module_status(path, "error", 10 * sy.TimeSpan.SECOND)
            assert (
                "channel parameter 'valve' cannot have a default" in status.description
            ), status

    def _verify_copies_at_the_top_level(self) -> None:
        span = sy.TimeRange(self.start, sy.TimeStamp.MAX)
        for trigger, count in (("pb_copy_a", 1), ("pb_copy_b", 2)):
            self.log(f"Writing {trigger}=1 for copy {count} of each body to start")
            self.writer.write(trigger, 1)
            for name in COUNTED:

                def written(name: str = name, count: int = count) -> bool | None:
                    return len(self.client.read(span, name)) == count or None

                if (
                    sy.poll(written, timeout=5 * sy.TimeSpan.SECOND, interval=0.05)
                    is None
                ):
                    self.fail(f"{name} does not hold {count} samples after {trigger}")

    def _verify_matrix(self) -> None:
        self.log("Each body with no trigger and each body on true must run")
        for use in MATRIX:
            if use.start != "trigger":
                self.wait_for_eq(use.channel, 1)
        self.log("No body on a trigger can run before the trigger")
        for use in MATRIX:
            if use.start == "trigger" and self.read_tlm(use.channel) is not None:
                self.fail(f"{use.channel} ran before pb_matrix_go")
        self.log("Writing pb_matrix_go=1 for each body on a trigger to start")
        self.writer.write("pb_matrix_go", 1)
        for use in MATRIX:
            self.wait_for_eq(use.channel, 1)
        self.log("Writing pb_matrix_probe=1 for the stage that holds the bodies")
        self.writer.write("pb_matrix_probe", 1)
        self.wait_for_eq("pb_matrix_alive", 1)
        self.log("No sequence that holds a body can move to its next step")
        for use in MATRIX:
            if use.holder == "sequence" and self.read_tlm(use.after) is not None:
                self.fail(f"{use.after} ran, so the sequence left {use.channel}")

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

# trigger_1 (fired by the harness) drives the module-scope bodies and the named
# sibling sequence. trigger_2 activates the select sibling chain. trigger_3 and
# trigger_4 activate the gated stage and sequence selects respectively, kept
# separate so no single source fans out to two transitions. trigger_5 and
# trigger_6 activate the lifecycle flows (blocking/wait race and
# re-entry/deactivation) one at a time, separate from the trigger_1 bodies.
ARC_INLINE_BODY_SOURCE = """
// Flat stage: a multi-write body; all writes fire in parallel.
trigger_1 -> stage {
    1 -> stage_1
    2 -> stage_2
    3 -> stage_3
}

// Flat sequence: a multi-write body run as ordered steps.
trigger_1 -> sequence {
    1 -> seq_1
    2 -> seq_2
    3 -> seq_3
}

// Nested stage: an inner stage body fires in the same cycle.
trigger_1 -> stage {
    1 -> stage {
        1 -> nested_stage_out_1
        2 -> nested_stage_out_2
        3 -> nested_stage_out_3
    }
}

// Nested sequence: an inner sequence body fires in the same cycle.
trigger_1 -> sequence {
    1 -> sequence {
        1 -> nested_seq_out_1
        2 -> nested_seq_out_2
        3 -> nested_seq_out_3
    }
}

// Deep stage: four nested levels; the innermost fires in-cycle.
trigger_1 -> stage {
    1 -> stage {
        1 -> stage {
            1 -> stage { 1 -> deep_stage_out }
        }
    }
}

// Deep sequence: four nested levels; the innermost fires in-cycle.
trigger_1 -> sequence {
    1 -> sequence {
        1 -> sequence {
            1 -> sequence { 1 -> deep_seq_out }
        }
    }
}

// Mixed kind: a stage body routes into an inline sequence.
trigger_1 -> stage {
    1 -> sequence { 1 -> mixed_stage_seq_out }
}

// Mixed kind: a sequence body routes into an inline stage.
trigger_1 -> sequence {
    1 -> stage { 1 -> mixed_seq_stage_out }
}

// Alternating-kind nesting (stage outermost): the kind switches at each level.
trigger_1 -> stage {
    1 -> sequence {
        1 -> stage {
            1 -> sequence { 1 -> alt_nested_stage_out }
        }
    }
}

// Alternating-kind nesting (sequence outermost): the kind switches at each level.
trigger_1 -> sequence {
    1 -> stage {
        1 -> sequence {
            1 -> stage { 1 -> alt_nested_seq_out }
        }
    }
}

// Select into a multi-write stage branch; writes fire in parallel.
trigger_1 -> select{} -> {
    true: stage {
        1 -> select_multi_stage_1
        2 -> select_multi_stage_2
        3 -> select_multi_stage_3
    }
}

// Select into a multi-write sequence branch; writes run as steps.
trigger_1 -> select{} -> {
    true: sequence {
        1 -> select_multi_seq_1
        2 -> select_multi_seq_2
        3 -> select_multi_seq_3
    }
}

// Single-branch select: a true-only stage branch.
trigger_1 -> select{} -> {
    true: stage { 1 -> select_single_stage_out }
}

// Single-branch select: a true-only sequence branch.
trigger_1 -> select{} -> {
    true: sequence { 1 -> select_single_seq_out }
}

// Triple-nested select (stage): the innermost branch fires in-cycle.
trigger_1 -> select{} -> {
    true: stage {
        1 -> select{} -> {
            true: stage {
                1 -> select{} -> {
                    true: stage { 1 -> nested_select_stage_out }
                }
            }
        }
    }
}

// Triple-nested select (sequence): the innermost branch fires in-cycle.
trigger_1 -> select{} -> {
    true: sequence {
        1 -> select{} -> {
            true: sequence {
                1 -> select{} -> {
                    true: sequence { 1 -> nested_select_seq_out }
                }
            }
        }
    }
}

// Cross-nested: a flow stage body wrapping a select.
trigger_1 -> stage {
    1 -> select{} -> {
        true: stage { 1 -> cross_flow_select_out }
    }
}

// Cross-nested: a select wrapping a flow stage body.
trigger_1 -> select{} -> {
    true: stage {
        1 -> stage { 1 -> cross_select_flow_out }
    }
}

// Cross-nested: a flow sequence body wrapping a select.
trigger_1 -> sequence {
    1 -> select{} -> {
        true: stage { 1 -> cross_seq_select_out }
    }
}

// Cross-nested: a select wrapping a flow sequence body.
trigger_1 -> select{} -> {
    true: sequence {
        1 -> sequence { 1 -> cross_select_seq_out }
    }
}

// Sibling transition: a multi-write inline stage then a multi-step inline
// sequence body each write their outputs and advance main to the next sibling
// stage, walking through to a third stage.
trigger_1 => main

sequence main {
    stage first_stage {
        1 -> stage {
            1 -> sibling_first_a
            1 -> sibling_first_b
            1 -> sibling_first_c
            1 => second_stage
        }
    }
    stage second_stage {
        1 -> sequence {
            1 -> sibling_second_a
            1 -> sibling_second_b
            1 -> sibling_second_c
            1 => third_stage
        }
    }
    stage third_stage {
        1 -> sibling_third_out
    }
}

// Select branch selection (stage): truthy fires true, falsy fires false.
select_stage_flag -> select{} -> {
    true: stage { 1 -> select_stage_true_out },
    false: stage { 1 -> select_stage_false_out }
}

// Select branch selection (sequence): same true/false routing.
select_seq_flag -> select{} -> {
    true: sequence { 1 -> select_seq_true_out },
    false: sequence { 1 -> select_seq_false_out }
}

// Single-branch select (false-only): the stage branch fires on a falsy input.
select_stage_flag -> select{} -> {
    false: stage { 1 -> select_false_stage_out }
}

// Single-branch select (false-only): the sequence-branch variant.
select_seq_flag -> select{} -> {
    false: sequence { 1 -> select_false_seq_out }
}

// Select sibling transition: a select branch writes an output and advances the
// enclosing sequence to the next sibling stage.
trigger_2 => select_main

sequence select_main {
    stage sel_first_stage {
        1 -> select{} -> {
            true: stage {
                1 -> select_sibling_first_a
                1 -> select_sibling_first_b
                1 -> select_sibling_first_c
                1 => sel_second_stage
            }
        }
    }
    stage sel_second_stage {
        1 -> select{} -> {
            true: sequence {
                1 -> select_sibling_second_a
                1 -> select_sibling_second_b
                1 -> select_sibling_second_c
                1 => sel_third_stage
            }
        }
    }
    stage sel_third_stage {
        1 -> select_sibling_third_out
    }
}

// Gated select (stage): fires only once trigger_3 activates the named stage.
trigger_3 => select_gated_stage_main

stage select_gated_stage_main {
    1 -> select{} -> {
        true: stage { 1 -> select_gated_stage_out }
    }
}

// Gated select (sequence): fires only once trigger_4 activates the named
// sequence.
trigger_4 => select_gated_seq_main

sequence select_gated_seq_main {
    1 -> select{} -> {
        true: sequence { 1 -> select_gated_seq_out }
    }
}

// ============================================================================
// LIFECYCLE FLOWS
//
// Temporal/lifecycle semantics the bodies above do not cover: stage blocking,
// wait{} backstop exits, multi-exit races, sub-sequence reset on re-entry, and
// stage deactivation. Each is gated behind its own trigger and verified one at
// a time, separate from the trigger_1-driven bodies.
// ============================================================================

// Blocking inline stages with a multi-exit race. The sequence pauses at each
// inline stage until an exit fires. The first stage's condition exit beats its
// wait backstop; the second stage's condition is unreachable, so the wait
// backstop fires instead.
trigger_5 => race_main

sequence race_main {
    1 -> race_first_out
    stage {
        race_cond < 15 => next
        wait{10s} => next
    }
    0 -> race_first_out
    1 -> race_second_out
    stage {
        race_cond > 9000 => next
        wait{2s} => next
    }
    0 -> race_second_out
}

// ============================================================================

// Inline sub-sequence running alongside a stage's reactive exit. Re-entering
// the fire stage resets the sub-sequence (reentry_fire_out returns to 1); the
// exit stage stops writing once left (reentry_exit_out must not re-apply).
trigger_6 => reentry_main

sequence reentry_main {
    stage fire {
        sequence {
            1 -> reentry_fire_out
        }
        reentry_cond < 15 => exit
    }
    stage exit {
        0 -> reentry_fire_out
        1 -> reentry_exit_out
        reentry_cond > 100 => fire
    }
}
"""

# Flat bodies: each write lands on the output whose suffix matches the value.
FLAT_STAGE_OUTS = ["stage_1", "stage_2", "stage_3"]
FLAT_SEQUENCE_OUTS = ["seq_1", "seq_2", "seq_3"]
NESTED_STAGE_OUTS = ["nested_stage_out_1", "nested_stage_out_2", "nested_stage_out_3"]
NESTED_SEQUENCE_OUTS = ["nested_seq_out_1", "nested_seq_out_2", "nested_seq_out_3"]
DEEP_NESTED_OUTS = ["deep_stage_out", "deep_seq_out"]
MIXED_NESTED_OUTS = ["mixed_stage_seq_out", "mixed_seq_stage_out"]
ALT_NESTED_OUTS = ["alt_nested_stage_out", "alt_nested_seq_out"]
SELECT_MULTI_STAGE_OUTS = [
    "select_multi_stage_1",
    "select_multi_stage_2",
    "select_multi_stage_3",
]
SELECT_MULTI_SEQUENCE_OUTS = [
    "select_multi_seq_1",
    "select_multi_seq_2",
    "select_multi_seq_3",
]
SELECT_SINGLE_OUTS = ["select_single_stage_out", "select_single_seq_out"]
NESTED_SELECT_OUTS = ["nested_select_stage_out", "nested_select_seq_out"]
CROSS_NESTED_OUTS = [
    "cross_flow_select_out",
    "cross_select_flow_out",
    "cross_seq_select_out",
    "cross_select_seq_out",
]
SIBLING_OUTS = [
    "sibling_first_a",
    "sibling_first_b",
    "sibling_first_c",
    "sibling_second_a",
    "sibling_second_b",
    "sibling_second_c",
    "sibling_third_out",
]
SELECT_BRANCH_OUTS = [
    "select_stage_true_out",
    "select_stage_false_out",
    "select_seq_true_out",
    "select_seq_false_out",
]
SELECT_FALSE_OUTS = ["select_false_stage_out", "select_false_seq_out"]
SELECT_SIBLING_OUTS = [
    "select_sibling_first_a",
    "select_sibling_first_b",
    "select_sibling_first_c",
    "select_sibling_second_a",
    "select_sibling_second_b",
    "select_sibling_second_c",
    "select_sibling_third_out",
]
GATED_OUTS = ["select_gated_stage_out", "select_gated_seq_out"]
RACE_OUTS = ["race_first_out", "race_second_out"]
REENTRY_OUTS = ["reentry_fire_out", "reentry_exit_out"]

OUTPUTS = (
    FLAT_STAGE_OUTS
    + FLAT_SEQUENCE_OUTS
    + NESTED_STAGE_OUTS
    + NESTED_SEQUENCE_OUTS
    + DEEP_NESTED_OUTS
    + MIXED_NESTED_OUTS
    + ALT_NESTED_OUTS
    + SELECT_MULTI_STAGE_OUTS
    + SELECT_MULTI_SEQUENCE_OUTS
    + SELECT_SINGLE_OUTS
    + NESTED_SELECT_OUTS
    + CROSS_NESTED_OUTS
    + SIBLING_OUTS
    + SELECT_BRANCH_OUTS
    + SELECT_FALSE_OUTS
    + SELECT_SIBLING_OUTS
    + GATED_OUTS
    + RACE_OUTS
    + REENTRY_OUTS
)

# Inputs the test drives. trigger_1 is created by the base class as the start cmd.
INPUTS = [
    "trigger_2",
    "trigger_3",
    "trigger_4",
    "trigger_5",
    "trigger_6",
    "select_stage_flag",
    "select_seq_flag",
]

# Float inputs driving the ported lifecycle flows' reactive conditions.
FLOAT_INPUTS = ["race_cond", "reentry_cond"]

CREATE_CHANNELS = OUTPUTS + INPUTS


class InlineBodies(ArcConsoleCase):
    """Anonymous inline ``stage``/``sequence`` bodies used as routing targets,
    exercised across module scope, nested bodies, sequence-scope sibling
    transitions, and ``select`` routing-table branches.

    Shapes exercised:

    - Flat: a multi-write body per kind. A ``stage`` fires all writes in
      parallel; a ``sequence`` runs them as ordered steps.
    - Nested: each body routes into an inner inline body of the same kind, which
      fires in the same cycle.
    - Deep nesting: same-kind inline bodies nested several levels deep; the
      innermost body fires in the same cycle.
    - Mixed-kind nesting: a ``stage`` body routes into an inline ``sequence`` and
      a ``sequence`` body into an inline ``stage``; the inner body fires in the
      same cycle regardless of kind.
    - Alternating-kind nesting: inline bodies nested several levels deep with the
      kind switching at each level; the innermost body fires in the same cycle.
    - Select multi-write branch: a truthy value routed into a ``select`` fires a
      multi-write branch body — a ``stage`` writes all outputs in parallel, a
      ``sequence`` as ordered steps.
    - Single-branch select: a ``select`` with a single branch fires it when its
      routed value matches — a ``true``-only branch on a truthy value, a
      ``false``-only branch on a falsy value — for both ``stage`` and
      ``sequence`` bodies.
    - Nested select: a ``select`` branch body routes into a further ``select``,
      three deep; the innermost branch fires in the same cycle.
    - Cross-nested flow/routing: an inline flow body wrapping a ``select`` and a
      ``select`` wrapping an inline flow body — for both ``stage`` and
      ``sequence`` wrappers — each fire their innermost body in the same cycle.
    - Sibling transition: inside a named ``sequence``, a multi-write inline
      ``stage`` body (first stage) and a multi-step inline ``sequence`` body
      (second stage) each write their outputs and transition to the next sibling
      stage, walking the sequence through to a third stage.
    - Select branch selection: a value routed into a ``select`` fires the
      matching branch body — truthy routes the ``true`` branch, falsy the
      ``false`` branch — for both ``stage`` and ``sequence`` bodies.
    - Select sibling transition: a constant routed through a ``select`` inside a
      sequence stage fires the matching branch body, which writes an output and
      transitions to the next sibling stage, cascading the sequence off a single
      activation edge.
    - Gated select: a ``select`` inside a named scope fires only once that scope
      is activated, routing its constant into the matching branch body.
    - Blocking / wait race: inside a named ``sequence``, an inline ``stage``
      blocks progression until an exit fires — a condition exit beats its
      ``wait{}`` backstop, while an unreachable condition lets the backstop win.
    - Re-entry / deactivation: an inline ``sequence`` inside a stage re-runs from
      its first step when the parent stage is re-entered, and a stage stops
      applying its writes once it is left.

    Every output holds its written value.
    """

    arc_source = ARC_INLINE_BODY_SOURCE
    arc_name_prefix = "ArcInlineBodies"
    start_cmd_channel = "trigger_1"
    subscribe_channels = OUTPUTS

    def setup(self) -> None:
        for channel in CREATE_CHANNELS:
            create_virtual_channel(self.client, channel, sy.DataType.UINT8)
        for channel in FLOAT_INPUTS:
            create_virtual_channel(self.client, channel, sy.DataType.FLOAT32)
        super().setup()

    def verify_sequence_execution(self) -> None:
        self._verify_flat_bodies()
        self._verify_nested_bodies()
        self._verify_deep_nesting()
        self._verify_mixed_nesting()
        self._verify_alt_nesting()
        self._verify_select_multiwrite()
        self._verify_select_single_branch()
        self._verify_nested_selects()
        self._verify_cross_nested()
        self._verify_sibling_sequence()
        self._verify_select_branches()
        self._verify_false_only_selects()
        self._verify_select_sibling_chain()
        self._verify_gated_selects()
        self._verify_blocking_wait_race()
        self._verify_reentry_deactivation()

    def _verify_flat_bodies(self) -> None:
        self.log("Verifying module-scope flat bodies")
        for value, out in enumerate(FLAT_STAGE_OUTS, start=1):
            self.wait_for_eq(out, value, is_virtual=True)
        for value, out in enumerate(FLAT_SEQUENCE_OUTS, start=1):
            self.wait_for_eq(out, value, is_virtual=True)

    def _verify_nested_bodies(self) -> None:
        self.log("Verifying module-scope nested bodies")
        for value, out in enumerate(NESTED_STAGE_OUTS, start=1):
            self.wait_for_eq(out, value, is_virtual=True)
        for value, out in enumerate(NESTED_SEQUENCE_OUTS, start=1):
            self.wait_for_eq(out, value, is_virtual=True)

    def _verify_deep_nesting(self) -> None:
        self.log("Verifying deep same-kind nested bodies")
        for out in DEEP_NESTED_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_mixed_nesting(self) -> None:
        self.log("Verifying mixed-kind nested bodies")
        for out in MIXED_NESTED_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_alt_nesting(self) -> None:
        self.log("Verifying alternating-kind nested bodies")
        for out in ALT_NESTED_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_select_multiwrite(self) -> None:
        self.log("Verifying multi-write select branch bodies")
        for value, out in enumerate(SELECT_MULTI_STAGE_OUTS, start=1):
            self.wait_for_eq(out, value, is_virtual=True)
        for value, out in enumerate(SELECT_MULTI_SEQUENCE_OUTS, start=1):
            self.wait_for_eq(out, value, is_virtual=True)

    def _verify_select_single_branch(self) -> None:
        self.log("Verifying single-branch (true-only) selects")
        for out in SELECT_SINGLE_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_nested_selects(self) -> None:
        self.log("Verifying triple-nested selects")
        for out in NESTED_SELECT_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_cross_nested(self) -> None:
        self.log("Verifying cross-nested flow/routing bodies")
        for out in CROSS_NESTED_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_sibling_sequence(self) -> None:
        self.log("Verifying sibling-transition sequence")
        for out in SIBLING_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_select_branches(self) -> None:
        self.log("Verifying module-scope select branch selection")
        self.writer.write("select_stage_flag", 1)
        self.writer.write("select_seq_flag", 1)
        self.wait_for_eq("select_stage_true_out", 1, is_virtual=True)
        self.wait_for_eq("select_seq_true_out", 1, is_virtual=True)
        self.writer.write("select_stage_flag", 0)
        self.writer.write("select_seq_flag", 0)
        self.wait_for_eq("select_stage_false_out", 1, is_virtual=True)
        self.wait_for_eq("select_seq_false_out", 1, is_virtual=True)

    def _verify_false_only_selects(self) -> None:
        self.log("Verifying false-only select branches")
        for out in SELECT_FALSE_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_select_sibling_chain(self) -> None:
        self.log("Activating select sibling chain via trigger_2")
        self.writer.write("trigger_2", 1)
        for out in SELECT_SIBLING_OUTS:
            self.wait_for_eq(out, 1, is_virtual=True)

    def _verify_gated_selects(self) -> None:
        self.log("Activating gated stage select via trigger_3")
        self.writer.write("trigger_3", 1)
        self.wait_for_eq("select_gated_stage_out", 1, is_virtual=True)
        self.log("Activating gated sequence select via trigger_4")
        self.writer.write("trigger_4", 1)
        self.wait_for_eq("select_gated_seq_out", 1, is_virtual=True)

    def _verify_blocking_wait_race(self) -> None:
        self.log("Activating blocking/wait race via trigger_5")
        self.writer.write("trigger_5", 1)
        self.wait_for_eq("race_first_out", 1, is_virtual=True)

        self.log("Holding race_cond high; first stage must keep blocking")
        self.writer.write("race_cond", 100.0)
        sy.sleep(0.5)
        blocked = self.read_tlm("race_first_out")
        if blocked != 1:
            self.fail(
                f"race_first_out={blocked} while race_cond=100; the inline stage "
                "should still be blocking the sequence"
            )
            return

        self.log("Dropping race_cond low; condition exit should win the race")
        self.writer.write("race_cond", 10.0)
        self.wait_for_eq(
            "race_first_out", 0, timeout=2 * sy.TimeSpan.SECOND, is_virtual=True
        )

        self.log("Second stage entered; only the wait{2s} backstop can exit it")
        self.wait_for_eq("race_second_out", 1, is_virtual=True)
        t_entry = sy.TimeStamp.now()
        self.wait_for_eq(
            "race_second_out", 0, timeout=4 * sy.TimeSpan.SECOND, is_virtual=True
        )
        elapsed = t_entry.span(sy.TimeStamp.now()).seconds
        self.log(f"Wait backstop fired after {elapsed:.2f}s")
        if elapsed < 1.5:
            self.fail(
                f"second stage exited in {elapsed:.2f}s but wait{{2s}} should not "
                "fire before ~2s; the condition exit may have fired spuriously"
            )
            return

    def _verify_reentry_deactivation(self) -> None:
        self.log("Activating re-entry/deactivation cycle via trigger_6")
        self.writer.write("trigger_6", 1)
        self.wait_for_eq("reentry_fire_out", 1, is_virtual=True)

        self.log("Dropping reentry_cond low; fire -> exit")
        self.writer.write("reentry_cond", 10.0)
        self.wait_for_eq("reentry_fire_out", 0, is_virtual=True)
        self.wait_for_eq("reentry_exit_out", 1, is_virtual=True)

        self.log("Driving reentry_cond high; exit -> fire re-runs the sub-sequence")
        self.writer.write("reentry_cond", 150.0)
        self.wait_for_eq("reentry_fire_out", 1, is_virtual=True)

        self.log("Exit stage must be deactivated; its write must not re-apply")
        self.writer.write("reentry_exit_out", 0)
        sy.sleep(0.5)
        exit_out = self.read_tlm("reentry_exit_out")
        if exit_out != 0:
            self.fail(
                f"reentry_exit_out={exit_out} after manual reset while in the fire "
                "stage; the exit stage's write is still applying, so the stage did "
                "not fully deactivate on the backward transition"
            )
            return

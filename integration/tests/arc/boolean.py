#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass

import synnax as sy
from framework.utils import create_virtual_channels
from tests.arc.arc import ArcCase

BOOL = sy.DataType.BOOL
INT32 = sy.DataType.INT32
INT64 = sy.DataType.INT64
UINT8 = sy.DataType.UINT8

ARC_BOOLEAN_SOURCE = """
// ─────────────────────── literals and constant folds ────────────────

func do_literals(a bool) {
    bool_lit_true_out = true
    bool_lit_false_out = false
    bool_not_true_out = not true
    bool_not_false_out = not false
    bool_and_tt_out = true and true
    bool_and_tf_out = true and false
    bool_or_tf_out = true or false
    bool_or_ff_out = false or false
}
bool_a1_trigger -> do_literals{}

bool_flow_trigger -> sequence {
    true -> bool_lit_true_out
    false -> bool_lit_false_out
    not true -> bool_not_true_out
    not false -> bool_not_false_out
    true and true -> bool_and_tt_out
    true and false -> bool_and_tf_out
    true or false -> bool_or_tf_out
    false or false -> bool_or_ff_out
}

// ─────────────────────── logical operators (a = 0) ──────────────────

func do_logical_a0(a bool) {
    bool_not_0_out = not a
    bool_and_00_out = a and bool_and_00_b
    bool_and_01_out = a and bool_and_01_b
    bool_or_00_out = a or bool_or_00_b
    bool_or_01_out = a or bool_or_01_b
}
bool_a0_trigger -> do_logical_a0{}

bool_flow_trigger -> stage {
    not bool_flow_trigger -> bool_not_0_out
    bool_flow_trigger and bool_and_00_b -> bool_and_00_out
    bool_flow_trigger and bool_and_01_b -> bool_and_01_out
    bool_flow_trigger or bool_or_00_b -> bool_or_00_out
    bool_flow_trigger or bool_or_01_b -> bool_or_01_out
}

// ─────────────────────── logical operators (a = 1) ──────────────────

func do_logical_a1(a bool) {
    bool_not_1_out = not a
    bool_and_10_out = a and bool_and_10_b
    bool_and_11_out = a and bool_and_11_b
    bool_or_10_out = a or bool_or_10_b
    bool_or_11_out = a or bool_or_11_b
}
bool_a1_trigger -> do_logical_a1{}

bool_flow_trigger -> stage {
    not bool_flow_trigger -> bool_not_1_out
    bool_flow_trigger and bool_and_10_b -> bool_and_10_out
    bool_flow_trigger and bool_and_11_b -> bool_and_11_out
    bool_flow_trigger or bool_or_10_b -> bool_or_10_out
    bool_flow_trigger or bool_or_11_b -> bool_or_11_out
}

// ─────────────────────── symbol aliases (!, &&, ||) ─────────────────

func do_symbols(a bool) {
    bool_bang_out = !a
    bool_andand_out = a && bool_andand_b
    bool_oror_out = a || bool_oror_b
}
bool_a1_trigger -> do_symbols{}

bool_flow_trigger -> stage {
    !bool_flow_trigger -> bool_bang_out
    bool_flow_trigger && bool_andand_b -> bool_andand_out
    bool_flow_trigger || bool_oror_b -> bool_oror_out
}

// ─────────────────────── bitwise operators (i64) ────────────────────

func do_bitwise_i64(a i64) {
    bool_xor_kw_out = a xor bool_xor_kw_b
    bool_xor_kw_zero_out = a xor bool_xor_kw_zero_b
    bool_xor_kw_self_out = a xor bool_xor_kw_self_b
    bool_xor_sym_out = a ^ bool_xor_sym_b
    bool_xor_sym_zero_out = a ^ bool_xor_sym_zero_b
    bool_xor_sym_self_out = a ^ bool_xor_sym_self_b
    bool_xor_neg_out = a ^ bool_xor_neg_b
    bool_tilde_out = ~a
    bool_tilde_lit_out = ~12
    bool_tilde_eq_out = ~12 == ~i64(12)
}
bool_i64_trigger -> do_bitwise_i64{}

bool_i64_flow_trigger -> stage {
    bool_i64_flow_trigger xor bool_xor_kw_b -> bool_xor_kw_out
    bool_i64_flow_trigger xor bool_xor_kw_zero_b -> bool_xor_kw_zero_out
    bool_i64_flow_trigger xor bool_xor_kw_self_b -> bool_xor_kw_self_out
    bool_i64_flow_trigger ^ bool_xor_sym_b -> bool_xor_sym_out
    bool_i64_flow_trigger ^ bool_xor_sym_zero_b -> bool_xor_sym_zero_out
    bool_i64_flow_trigger ^ bool_xor_sym_self_b -> bool_xor_sym_self_out
    bool_i64_flow_trigger ^ bool_xor_neg_b -> bool_xor_neg_out
    ~bool_i64_flow_trigger -> bool_tilde_out
    ~12 -> bool_tilde_lit_out
    ~12 == ~i64(12) -> bool_tilde_eq_out
}

// ─────────────────────── bitwise operators (i32) ────────────────────

func do_bitwise_i32(a i32) {
    bool_xor_kw_i32_out = a xor bool_xor_kw_i32_b
    bool_xor_sym_i32_out = a ^ bool_xor_sym_i32_b
    bool_amp_out = a & bool_amp_b
    bool_pipe_out = a | bool_pipe_b
}
bool_i32_trigger -> do_bitwise_i32{}

bool_i32_flow_trigger -> stage {
    bool_i32_flow_trigger xor bool_xor_kw_i32_b -> bool_xor_kw_i32_out
    bool_i32_flow_trigger ^ bool_xor_sym_i32_b -> bool_xor_sym_i32_out
    bool_i32_flow_trigger & bool_amp_b -> bool_amp_out
    bool_i32_flow_trigger | bool_pipe_b -> bool_pipe_out
}

// ─────────────────────── comparisons produce bool ───────────────────

func do_compare(a i64) {
    bool_cmp_gt_out = a > bool_cmp_gt_b
    bool_cmp_lt_out = a < bool_cmp_lt_b
    bool_cmp_eq_out = a == bool_cmp_eq_b
    bool_cmp_neq_out = a != bool_cmp_neq_b
}
bool_i64_trigger -> do_compare{}

bool_i64_flow_trigger -> stage {
    bool_i64_flow_trigger > bool_cmp_gt_b -> bool_cmp_gt_out
    bool_i64_flow_trigger < bool_cmp_lt_b -> bool_cmp_lt_out
    bool_i64_flow_trigger == bool_cmp_eq_b -> bool_cmp_eq_out
    bool_i64_flow_trigger != bool_cmp_neq_b -> bool_cmp_neq_out
}

// ─────────────────────── if on bool (a = 1) ─────────────────────────

func do_if_a1(a bool) {
    if a {
        bool_if1_out = true
    } else {
        bool_if1_out = false
    }
    if not a {
        bool_if1_not_out = true
    } else {
        bool_if1_not_out = false
    }
}
bool_a1_trigger -> do_if_a1{}

// ─────────────────────── if on bool (a = 0) ─────────────────────────

func do_if_a0(a bool) {
    if a {
        bool_if0_out = true
    } else {
        bool_if0_out = false
    }
    if not a {
        bool_if0_not_out = true
    } else {
        bool_if0_not_out = false
    }
}
bool_a0_trigger -> do_if_a0{}

// ─────────────────────── select on bool keys ────────────────────────

bool_sel1_in -> select{} -> {
    true: 1 -> bool_sel1_out,
    false: 2 -> bool_sel1_out
}
bool_sel0_in -> select{} -> {
    true: 1 -> bool_sel0_out,
    false: 2 -> bool_sel0_out
}

// ─────────────────────── bool transition guard ──────────────────────

start_boolean_cmd => bool_seq

sequence bool_seq {
    stage first {
        1 -> bool_seq_ready
        bool_gate => second
    }
    stage second {
        1 -> bool_seq_done
    }
}
"""


@dataclass
class Case:
    """One assertion: ``bool_{label}_out`` == ``expected`` after the section fires."""

    label: str
    expected: int
    b_val: int | None = None
    out_type: sy.DataType | None = None


@dataclass
class Section:
    """Cases fired by one shared-trigger write, which carries the ``a`` operand.

    The trigger channel derives from ``a_type`` and ``a_val``; sections with the
    same pair share one trigger.
    """

    a_type: sy.DataType
    a_val: int
    cases: list[Case]


LITERALS = Section(
    BOOL,
    a_val=1,
    cases=[
        Case("lit_true", expected=1),
        Case("lit_false", expected=0),
        Case("not_true", expected=0),
        Case("not_false", expected=1),
        Case("and_tt", expected=1),
        Case("and_tf", expected=0),
        Case("or_tf", expected=1),
        Case("or_ff", expected=0),
    ],
)

LOGICAL_A0 = Section(
    BOOL,
    a_val=0,
    cases=[
        Case("not_0", expected=1),
        Case("and_00", b_val=0, expected=0),
        Case("and_01", b_val=1, expected=0),
        Case("or_00", b_val=0, expected=0),
        Case("or_01", b_val=1, expected=1),
    ],
)

LOGICAL_A1 = Section(
    BOOL,
    a_val=1,
    cases=[
        Case("not_1", expected=0),
        Case("and_10", b_val=0, expected=0),
        Case("and_11", b_val=1, expected=1),
        Case("or_10", b_val=0, expected=1),
        Case("or_11", b_val=1, expected=1),
    ],
)

SYMBOLS = Section(
    BOOL,
    a_val=1,
    cases=[
        Case("bang", expected=0),
        Case("andand", b_val=1, expected=1),
        Case("oror", b_val=0, expected=1),
    ],
)

BITWISE_I64 = Section(
    INT64,
    a_val=12,
    cases=[
        Case("xor_kw", b_val=10, expected=6),
        Case("xor_kw_zero", b_val=0, expected=12),
        Case("xor_kw_self", b_val=12, expected=0),
        Case("xor_sym", b_val=10, expected=6),
        Case("xor_sym_zero", b_val=0, expected=12),
        Case("xor_sym_self", b_val=12, expected=0),
        Case("xor_neg", b_val=-1, expected=-13),
        Case("tilde", expected=-13),
        Case("tilde_lit", expected=-13),
        Case("tilde_eq", expected=1, out_type=BOOL),
    ],
)

BITWISE_I32 = Section(
    INT32,
    a_val=12,
    cases=[
        Case("xor_kw_i32", b_val=10, expected=6),
        Case("xor_sym_i32", b_val=10, expected=6),
        Case("amp", b_val=10, expected=8),
        Case("pipe", b_val=10, expected=14),
    ],
)

COMPARE = Section(
    INT64,
    a_val=12,
    cases=[
        Case("cmp_gt", b_val=10, expected=1, out_type=BOOL),
        Case("cmp_lt", b_val=10, expected=0, out_type=BOOL),
        Case("cmp_eq", b_val=12, expected=1, out_type=BOOL),
        Case("cmp_neq", b_val=12, expected=0, out_type=BOOL),
    ],
)

IF_A1 = Section(
    BOOL,
    a_val=1,
    cases=[
        Case("if1", expected=1),
        Case("if1_not", expected=0),
    ],
)

IF_A0 = Section(
    BOOL,
    a_val=0,
    cases=[
        Case("if0", expected=0),
        Case("if0_not", expected=1),
    ],
)

SECTIONS = [
    LITERALS,
    LOGICAL_A0,
    LOGICAL_A1,
    SYMBOLS,
    BITWISE_I64,
    BITWISE_I32,
    COMPARE,
    IF_A1,
    IF_A0,
]

SELECT_CHANNELS: list[tuple[str, sy.DataType]] = [
    ("bool_sel1_in", BOOL),
    ("bool_sel0_in", BOOL),
    ("bool_sel1_out", UINT8),
    ("bool_sel0_out", UINT8),
]

FLOW_TRIGGERS: list[tuple[str, sy.DataType]] = [
    ("bool_flow_trigger", BOOL),
    ("bool_i64_flow_trigger", INT64),
    ("bool_i32_flow_trigger", INT32),
]

TRANSITION_CHANNELS: list[tuple[str, sy.DataType]] = [
    ("bool_gate", BOOL),
    ("bool_seq_ready", UINT8),
    ("bool_seq_done", UINT8),
]


def _ch(c: Case, part: str) -> str:
    return f"bool_{c.label}_{part}"


def _trigger(s: Section) -> str:
    if s.a_type == BOOL:
        return f"bool_a{s.a_val}_trigger"
    return "bool_i64_trigger" if s.a_type == INT64 else "bool_i32_trigger"


def _flow_trigger(s: Section) -> str:
    if s.a_type == BOOL:
        return "bool_flow_trigger"
    return "bool_i64_flow_trigger" if s.a_type == INT64 else "bool_i32_flow_trigger"


class Boolean(ArcCase):
    """Boolean literals, operators, control flow, and routing through channels.

    Expression sections fire off one shared trigger each; every case derives its
    channels and one assertion from its row. The transition section runs a small
    sequence gated on a bool channel.
    """

    arc_source = ARC_BOOLEAN_SOURCE
    arc_name_prefix = "ArcBoolean"
    start_cmd_channel = "start_boolean_cmd"
    subscribe_channels = [_ch(c, "out") for s in SECTIONS for c in s.cases] + [
        "bool_sel1_out",
        "bool_sel0_out",
        "bool_seq_ready",
        "bool_seq_done",
    ]

    def setup(self) -> None:
        specs: list[tuple[str, sy.DataType]] = list(
            FLOW_TRIGGERS + SELECT_CHANNELS + TRANSITION_CHANNELS
        )
        for s in SECTIONS:
            specs.append((_trigger(s), s.a_type))
            for c in s.cases:
                if c.b_val is not None:
                    specs.append((_ch(c, "b"), s.a_type))
                specs.append((_ch(c, "out"), c.out_type or s.a_type))
        create_virtual_channels(self.client, list(dict.fromkeys(specs)))
        super().setup()

    def _drive_func(self, s: Section) -> None:
        for c in s.cases:
            if c.b_val is not None:
                self.log(f"[{c.label}] Writing b={c.b_val} to {_ch(c, 'b')}")
                self.writer.write(_ch(c, "b"), c.b_val)
        self.log(f"Writing a={s.a_val} to {_trigger(s)}")
        self.writer.write(_trigger(s), s.a_val)
        for c in s.cases:
            self.log(f"[{c.label}] Expecting {_ch(c, 'out')} == {c.expected}")
            self.wait_for_eq(_ch(c, "out"), c.expected)

    def _drive_flow(self, s: Section) -> None:
        trigger = _flow_trigger(s)
        # The func pass left every out at its expected value; a stale sample must
        # not satisfy the flow asserts. Knock each out off its expected value so
        # only a fresh flow write can pass.
        for c in s.cases:
            out_type = c.out_type or s.a_type
            sentinel = 1 - c.expected if out_type == BOOL else c.expected + 1
            self.writer.write(_ch(c, "out"), sentinel)
        self.log(f"Writing a={s.a_val} to {trigger}")
        self.writer.write(trigger, s.a_val)
        for c in s.cases:
            if c.b_val is not None:
                self.log(f"[{c.label}] Writing b={c.b_val} to {_ch(c, 'b')}")
                self.writer.write(_ch(c, "b"), c.b_val)
        self.writer.write(trigger, s.a_val)
        for c in s.cases:
            self.log(f"[{c.label}] Expecting {_ch(c, 'out')} == {c.expected}")
            self.wait_for_eq(_ch(c, "out"), c.expected)

    def verify_sequence_execution(self) -> None:
        self.log("=== literals and constant folds ===")
        self._drive_func(LITERALS)
        self._drive_flow(LITERALS)

        self.log("=== logical operators (a = 0) ===")
        self._drive_func(LOGICAL_A0)
        self._drive_flow(LOGICAL_A0)

        self.log("=== logical operators (a = 1) ===")
        self._drive_func(LOGICAL_A1)
        self._drive_flow(LOGICAL_A1)

        self.log("=== symbol aliases (!, &&, ||) ===")
        self._drive_func(SYMBOLS)
        self._drive_flow(SYMBOLS)

        self.log("=== bitwise operators (i64) ===")
        self._drive_func(BITWISE_I64)
        self._drive_flow(BITWISE_I64)

        self.log("=== bitwise operators (i32) ===")
        self._drive_func(BITWISE_I32)
        self._drive_flow(BITWISE_I32)

        self.log("=== comparisons produce bool ===")
        self._drive_func(COMPARE)
        self._drive_flow(COMPARE)

        self.log("=== if on bool (a = 1) ===")
        self._drive_func(IF_A1)

        self.log("=== if on bool (a = 0) ===")
        self._drive_func(IF_A0)

        self.log("=== select on bool keys ===")
        self.writer.write("bool_sel1_in", 1)
        self.wait_for_eq("bool_sel1_out", 1)
        self.writer.write("bool_sel0_in", 0)
        self.wait_for_eq("bool_sel0_out", 2)

        self.log("=== bool transition guard ===")
        self.wait_for_eq("bool_seq_ready", 1)
        self.writer.write("bool_gate", 1)
        self.wait_for_eq("bool_seq_done", 1)

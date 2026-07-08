#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_virtual_channels
from tests.arc.arc import ArcCase

ARC_VARIABLES_SOURCE = """
import time

// ──────────────────────────────── Const ────────────────────────────────
vars_start => const_main_sequence

sequence const_main_sequence {
    c_f64 f64 := 10.0
    c_u32 u32 := 7
    c_i64 i64 := -5
    c_str str := "hi"
    c_f64 -> f64_initial
    c_u32 -> u32_initial
    c_i64 -> i64_initial
    c_str -> str_initial
    c_f64 = 42.0
    c_u32 = 99
    c_i64 = -100
    c_str = "bye"
    c_f64 -> f64_final
    c_u32 -> u32_final
    c_i64 -> i64_final
    c_str -> str_final
}

// ──────────────────────────── ChannelAlias ─────────────────────────────
vars_start => alias_main_sequence

sequence alias_main_sequence {
    al_f64 := alias_f64_a
    al_u32 := alias_u32_a
    al_i64 := alias_i64_a
    al_str := alias_str_a
    al_f64 -> f64_initial
    al_u32 -> u32_initial
    al_i64 -> i64_initial
    al_str -> str_initial
    al_f64 = alias_f64_b
    al_u32 = alias_u32_b
    al_i64 = alias_i64_b
    al_str = alias_str_b
    al_f64 -> f64_final
    al_u32 -> u32_final
    al_i64 -> i64_final
    al_str -> str_final
}

// ─────────────────────────────── Reactive ──────────────────────────────
vars_start => reactive_main_sequence

sequence reactive_main_sequence {
    r_f64 := reactive_f64 + 1.0
    r_u32 := reactive_u32 + 1
    r_i64 := reactive_i64 + 1
    r_str := reactive_str + "!"
    r_f64 -> f64_initial
    r_u32 -> u32_initial
    r_i64 -> i64_initial
    r_str -> str_initial
    r_f64 = reactive_f64 + 100.0
    r_u32 = reactive_u32 + 10
    r_i64 = reactive_i64 + 10
    r_str = reactive_str + "?"
    r_f64 -> f64_final
    r_u32 -> u32_final
    r_i64 -> i64_final
    r_str -> str_final
}

// ─────────── variable inherited into an inline stage/sequence ────────────
vars_start => inherit_main

sequence inherit_main {
    vc f64 := 0.0

    stage inherit_load {
        inherit_in -> vc
        inherit_to_run_cmd => inherit_run
    }

    stage inherit_run {
        1 -> stage {
            vc -> inherit_out_inline_stage
            f"vc={vc}" -> inherit_out_inline_fmt
        }
        1 -> sequence {
            vc -> inherit_out_inline_seq
        }
    }
}

// ─────────────────────────── top-level scope ───────────────────────────
top_const str := "top"

vars_start => toplevel_seq_reader
vars_start => toplevel_stage_reader

sequence toplevel_seq_reader {
    top_const -> toplevel_from_seq
}

stage toplevel_stage_reader {
    top_const -> toplevel_from_stage
}

// ────────────────────────── stage-scoped var ───────────────────────────
vars_start => stage_scope_main

stage stage_scope_main {
    ss_var str := "stage"
    ss_var -> stage_scoped_out
}

// ───────── alias/reactive inherited into a nested inline body ───────────
vars_start => inherit_kind_main

sequence inherit_kind_main {
    ik_alias := inherit_alias_src
    ik_react := inherit_react_src + "!"

    stage {
        ik_alias -> inherit_alias_direct
        f"a={ik_alias}" -> inherit_alias_fmt
        ik_react -> inherit_react_direct
    }
}

// ────────────── scope-entry reset across nested re-entry ────────────────
counter_1 i64 := 0
counter_2 i64 $= 0

vars_start => reset_matrix_main

sequence reset_matrix_main {
    counter_3 i64 := 0
    counter_4 i64 $= 0

    stage s1 {
        counter_5 i64 := 0
        counter_6 i64 $= 5
        counter_7 i64 $= 0

        1 => counter_1 + 1 => counter_1
        1 => counter_2 + 1 => counter_2
        1 => counter_3 + 1 => counter_3
        1 => counter_4 + 1 => counter_4
        1 => counter_5 + 1 => counter_5
        1 => counter_6 + 1 => counter_6
        1 => counter_7 + 1 => counter_7

        counter_1 -> counter_out_1
        counter_2 -> counter_out_2
        counter_3 -> counter_out_3
        counter_4 -> counter_out_4
        counter_5 -> counter_out_5
        counter_6 -> counter_out_6
        counter_7 -> counter_out_7

        counter_7 >= 3 => reset_done_stage
        time.wait{100ms} => s2
    }
    stage s2 {
        1 => s1
    }
    stage reset_done_stage {
        1 -> reset_done
    }
}

// ---------- reactive re-expr across jumps (skip + reorder) ----------
vars_start => rx_main

sequence rx_main {
    rx str := "init: " + rx_src

    stage rx_entry {
        rx -> rx_out
        e_to_b >= 1 => rx_b
        e_to_c >= 1 => rx_c
    }
    stage rx_a {
        rx = "a: " + rx_src
        rx -> rx_out
        a_to_d >= 1 => rx_d
    }
    stage rx_b {
        rx = "b: " + rx_src
        rx -> rx_out
    }
    stage rx_c {
        rx = "c: " + rx_src
        rx -> rx_out
        c_to_a >= 1 => rx_a
    }
    stage rx_d {
        rx = "d: " + rx_src
        rx -> rx_out
    }
}
"""

VAR_OUTPUTS = [
    "f64_initial",
    "u32_initial",
    "i64_initial",
    "str_initial",
    "f64_final",
    "u32_final",
    "i64_final",
    "str_final",
]
INHERIT_OUTPUTS = [
    "inherit_out_inline_stage",
    "inherit_out_inline_fmt",
    "inherit_out_inline_seq",
]
SCOPE_OUTPUTS = [
    "toplevel_from_seq",
    "toplevel_from_stage",
    "stage_scoped_out",
    "inherit_alias_direct",
    "inherit_alias_fmt",
    "inherit_react_direct",
]
RESET_OUTPUTS = [
    "counter_out_1",
    "counter_out_2",
    "counter_out_3",
    "counter_out_4",
    "counter_out_5",
    "counter_out_6",
    "counter_out_7",
    "reset_done",
]
REEXPR_OUTPUTS = [
    "rx_out",
]
OUTPUTS = VAR_OUTPUTS + INHERIT_OUTPUTS + SCOPE_OUTPUTS + RESET_OUTPUTS + REEXPR_OUTPUTS

F64_CHANNELS = [
    "alias_f64_a",
    "alias_f64_b",
    "reactive_f64",
    "inherit_in",
    "f64_initial",
    "f64_final",
    "inherit_out_inline_stage",
    "inherit_out_inline_seq",
]
U32_CHANNELS = [
    "alias_u32_a",
    "alias_u32_b",
    "reactive_u32",
    "u32_initial",
    "u32_final",
]
I64_CHANNELS = [
    "alias_i64_a",
    "alias_i64_b",
    "reactive_i64",
    "i64_initial",
    "i64_final",
    "counter_out_1",
    "counter_out_2",
    "counter_out_3",
    "counter_out_4",
    "counter_out_5",
    "counter_out_6",
    "counter_out_7",
]
STR_CHANNELS = [
    "alias_str_a",
    "alias_str_b",
    "reactive_str",
    "str_initial",
    "str_final",
    "inherit_out_inline_fmt",
    "toplevel_from_seq",
    "toplevel_from_stage",
    "stage_scoped_out",
    "inherit_alias_src",
    "inherit_react_src",
    "inherit_alias_direct",
    "inherit_alias_fmt",
    "inherit_react_direct",
    "rx_src",
    "rx_out",
]
U8_CHANNELS = [
    "inherit_to_run_cmd",
    "reset_done",
    "e_to_c",
    "e_to_b",
    "c_to_a",
    "a_to_d",
]

CHANNELS: list[tuple[str, sy.DataType]] = (
    [(name, sy.DataType.FLOAT64) for name in F64_CHANNELS]
    + [(name, sy.DataType.UINT8) for name in U8_CHANNELS]
    + [(name, sy.DataType.UINT32) for name in U32_CHANNELS]
    + [(name, sy.DataType.INT64) for name in I64_CHANNELS]
    + [(name, sy.DataType.STRING) for name in STR_CHANNELS]
)


class Variables(ArcCase):
    """Runtime coverage for scoped Arc variables and channel aliases.

    Const, ChannelAlias, and Reactive variables across the f64/u32/i64/str data
    types, each read to its own channel, plus reassignment (const overwrite,
    alias rebind, reactive re-express), and inheritance into inline bodies.
    """

    arc_source = ARC_VARIABLES_SOURCE
    arc_name_prefix = "ArcVariables"
    start_cmd_channel = "vars_start"
    subscribe_channels = OUTPUTS

    def setup(self) -> None:
        create_virtual_channels(self.client, CHANNELS)
        super().setup()

    def verify_sequence_execution(self) -> None:
        self._verify_const()
        self._verify_channel_alias()
        self._verify_reactive()
        self._verify_inline_inheritance()
        self._verify_top_level_scope()
        self._verify_stage_scope()
        self._verify_kind_inheritance()
        self._verify_scope_reset_matrix()
        self._verify_reexpr()

    def _verify_const(self) -> None:
        self.log("=== Const ===")
        self.wait_for_eq("f64_initial", 10.0)
        self.wait_for_eq("u32_initial", 7)
        self.wait_for_eq("i64_initial", -5)
        self.wait_for_eq("str_initial", "hi")
        self.wait_for_eq("f64_final", 42.0)
        self.wait_for_eq("u32_final", 99)
        self.wait_for_eq("i64_final", -100)
        self.wait_for_eq("str_final", "bye")

    def _verify_channel_alias(self) -> None:
        self.log("=== ChannelAlias ===")
        self.writer.write("alias_f64_a", 1.5)
        self.writer.write("alias_u32_a", 2)
        self.writer.write("alias_i64_a", -3)
        self.writer.write("alias_str_a", "a")
        self.wait_for_eq("f64_initial", 1.5)
        self.wait_for_eq("u32_initial", 2)
        self.wait_for_eq("i64_initial", -3)
        self.wait_for_eq("str_initial", "a")

        self.writer.write("alias_f64_b", 9.0)
        self.writer.write("alias_u32_b", 8)
        self.writer.write("alias_i64_b", -7)
        self.writer.write("alias_str_b", "b")
        self.wait_for_eq("f64_final", 9.0)
        self.wait_for_eq("u32_final", 8)
        self.wait_for_eq("i64_final", -7)
        self.wait_for_eq("str_final", "b")

    def _verify_reactive(self) -> None:
        self.log("=== Reactive ===")
        self.writer.write("reactive_f64", 2.0)
        self.writer.write("reactive_u32", 5)
        self.writer.write("reactive_i64", -4)
        self.writer.write("reactive_str", "x")
        self.wait_for_eq("f64_initial", 3.0)
        self.wait_for_eq("u32_initial", 6)
        self.wait_for_eq("i64_initial", -3)
        self.wait_for_eq("str_initial", "x!")

        self.writer.write("reactive_f64", 2.0)
        self.writer.write("reactive_u32", 5)
        self.writer.write("reactive_i64", -4)
        self.writer.write("reactive_str", "x")
        self.wait_for_eq("f64_final", 102.0)
        self.wait_for_eq("u32_final", 15)
        self.wait_for_eq("i64_final", 6)
        self.wait_for_eq("str_final", "x?")

    def _verify_inline_inheritance(self) -> None:
        self.log("=== inherited into inline stage/sequence ===")
        self.writer.write("inherit_in", 12.0)
        self.writer.write("inherit_to_run_cmd", 1)
        self.wait_for_eq("inherit_out_inline_seq", 12.0)
        self.wait_for_eq("inherit_out_inline_stage", 12.0)
        self.wait_for_eq("inherit_out_inline_fmt", "vc=12")

    def _verify_top_level_scope(self) -> None:
        self.log("=== top-level scope ===")
        self.wait_for_eq("toplevel_from_seq", "top")
        self.wait_for_eq("toplevel_from_stage", "top")

    def _verify_stage_scope(self) -> None:
        self.log("=== stage-scoped var ===")
        self.wait_for_eq("stage_scoped_out", "stage")

    def _verify_kind_inheritance(self) -> None:
        self.log("=== alias/reactive inherited into a nested stage ===")
        self.writer.write("inherit_alias_src", "A")
        self.writer.write("inherit_react_src", "R")
        self.wait_for_eq("inherit_alias_direct", "A")
        self.wait_for_eq("inherit_alias_fmt", "a=A")
        self.wait_for_eq("inherit_react_direct", "R!")

    def _verify_scope_reset_matrix(self) -> None:
        self.log("=== scope-entry reset across nested re-entry ===")
        self.wait_for_eq("reset_done", 1)
        self.wait_for_eq("counter_out_5", 1)
        self.wait_for_eq("counter_out_1", 3)
        self.wait_for_eq("counter_out_2", 3)
        self.wait_for_eq("counter_out_3", 3)
        self.wait_for_eq("counter_out_4", 3)
        self.wait_for_eq("counter_out_6", 8)
        self.wait_for_eq("counter_out_7", 3)

    def _verify_reexpr(self) -> None:
        self.log("=== reactive re-expr across jumps (skip + reorder) ===")
        # rx is re-expressed by jumping stages out of source order (entry->c->a->d,
        # skipping b). A reassignment only swaps rx's derivation and fires on the
        # next rx_src write, so each hop writes a fresh rx_src and asserts on it.
        self.writer.write("rx_src", "1")
        self.wait_for_eq("rx_out", "init: 1")
        self.writer.write("e_to_c", 1)
        self.writer.write("rx_src", "2")
        self.wait_for_eq("rx_out", "c: 2")
        self.writer.write("c_to_a", 1)
        self.writer.write("rx_src", "3")
        self.wait_for_eq("rx_out", "a: 3")
        self.writer.write("a_to_d", 1)
        self.writer.write("rx_src", "4")
        self.wait_for_eq("rx_out", "d: 4")

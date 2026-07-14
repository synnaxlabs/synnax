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

// ──────────────────────────────── Literal ────────────────────────────────
vars_start => literal_main_sequence

sequence literal_main_sequence {
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

// ────────────────────────── ChannelReadWrite ───────────────────────────
vars_start => channel_read_write_main_sequence

sequence channel_read_write_main_sequence {
    al_f64 := channel_read_write_f64_a
    al_u32 := channel_read_write_u32_a
    al_i64 := channel_read_write_i64_a
    al_str := channel_read_write_str_a
    al_f64 -> f64_initial
    al_u32 -> u32_initial
    al_i64 -> i64_initial
    al_str -> str_initial
    al_f64 = channel_read_write_f64_b
    al_u32 = channel_read_write_u32_b
    al_i64 = channel_read_write_i64_b
    al_str = channel_read_write_str_b
    al_f64 -> f64_final
    al_u32 -> u32_final
    al_i64 -> i64_final
    al_str -> str_final
}

// ────────────────────────────── ChannelRead ────────────────────────────
vars_start => channel_read_main_sequence

sequence channel_read_main_sequence {
    r_f64 := channel_read_f64 + 1.0
    r_u32 := channel_read_u32 + 1
    r_i64 := channel_read_i64 + 1
    r_str := channel_read_str + "!"
    r_f64 -> f64_initial
    r_u32 -> u32_initial
    r_i64 -> i64_initial
    r_str -> str_initial
    r_f64 = channel_read_f64 + 100.0
    r_u32 = channel_read_u32 + 10
    r_i64 = channel_read_i64 + 10
    r_str = channel_read_str + "?"
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
top_literal str := "top"

vars_start => toplevel_seq_reader
vars_start => toplevel_stage_reader

sequence toplevel_seq_reader {
    top_literal -> toplevel_from_seq
}

stage toplevel_stage_reader {
    top_literal -> toplevel_from_stage
}

// ────────────────────────── stage-scoped var ───────────────────────────
vars_start => stage_scope_main

stage stage_scope_main {
    ss_var str := "stage"
    ss_var -> stage_scoped_out
}

// ── channel-read/write & channel-read inherited into a nested inline body ──
vars_start => inherit_kind_main

sequence inherit_kind_main {
    ik_channel_read_write := inherit_channel_read_write_src
    ik_channel_read := inherit_channel_read_src + "!"

    stage {
        ik_channel_read_write -> inherit_channel_read_write_direct
        f"a={ik_channel_read_write}" -> inherit_channel_read_write_fmt
        ik_channel_read -> inherit_channel_read_direct
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

// ─────────────── reassignment across jumps (skip + reorder) ───────────────
vars_start => reassign_main

sequence reassign_main {
    // Five reassignables on one jump path: rx channel-read, ra channel-read/write
    // read, wa channel-read/write write, rc literal, rs stateful. Each stage mutates
    // its vars (first block) then emits the readable ones (second block); wa is
    // write-only.
    rx str := "init: " + rx_src
    ra := ch_init
    wa := wa_init
    rc str := "init"
    rs str $= "init"

    stage r_entry {
        // initial bindings; "entry" -> wa_init
        "entry" -> wa

        rx -> rx_out
        ra -> ra_out
        rc -> rc_out
        rs -> rs_out

        e_to_b >= 1 => r_b
        e_to_c >= 1 => r_c
    }
    stage r_a {
        // jumped back from r_c; out-of-order "a" -> live wa_c
        rx = "a: " + rx_src
        "a" -> wa

        rx -> rx_out
        ra -> ra_out
        rc -> rc_out
        rs -> rs_out

        a_to_d >= 1 => r_d
    }
    stage r_b {
        // skipped on this path (the e_to_b branch)
        rx = "b: " + rx_src
        ra = chb
        wa = wa_b
        rc = "b"
        rs = "b"

        rx -> rx_out
        ra -> ra_out
        rc -> rc_out
        rs -> rs_out
    }
    stage r_c {
        // rebind ra -> chc, wa -> wa_c
        rx = "c: " + rx_src
        ra = chc
        wa = wa_c
        rc = "c"
        rs = "c"

        rx -> rx_out
        ra -> ra_out
        rc -> rc_out
        rs -> rs_out

        c_to_a >= 1 => r_a
    }
    stage r_d {
        // rebind ra -> chd, wa -> wa_d
        rx = "d: " + rx_src
        ra = chd
        wa = wa_d
        rc = "d"
        rs = "d"

        rx -> rx_out
        ra -> ra_out
        rc -> rc_out
        rs -> rs_out

        d_to_e >= 1 => r_e
    }
    stage r_e {
            // forward: "e" -> wa_d (rebound in r_d)
        rx = "e: " + rx_src
        "e" -> wa

        rx -> rx_out
        ra -> ra_out
        rc -> rc_out
        rs -> rs_out
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
    "inherit_channel_read_write_direct",
    "inherit_channel_read_write_fmt",
    "inherit_channel_read_direct",
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
    "ra_out",
    "rc_out",
    "rs_out",
    "wa_init",
    "wa_c",
    "wa_d",
]
OUTPUTS = VAR_OUTPUTS + INHERIT_OUTPUTS + SCOPE_OUTPUTS + RESET_OUTPUTS + REEXPR_OUTPUTS

F64_CHANNELS = [
    "channel_read_write_f64_a",
    "channel_read_write_f64_b",
    "channel_read_f64",
    "inherit_in",
    "f64_initial",
    "f64_final",
    "inherit_out_inline_stage",
    "inherit_out_inline_seq",
]
U32_CHANNELS = [
    "channel_read_write_u32_a",
    "channel_read_write_u32_b",
    "channel_read_u32",
    "u32_initial",
    "u32_final",
]
I64_CHANNELS = [
    "channel_read_write_i64_a",
    "channel_read_write_i64_b",
    "channel_read_i64",
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
    "channel_read_write_str_a",
    "channel_read_write_str_b",
    "channel_read_str",
    "str_initial",
    "str_final",
    "inherit_out_inline_fmt",
    "toplevel_from_seq",
    "toplevel_from_stage",
    "stage_scoped_out",
    "inherit_channel_read_write_src",
    "inherit_channel_read_src",
    "inherit_channel_read_write_direct",
    "inherit_channel_read_write_fmt",
    "inherit_channel_read_direct",
    "rx_src",
    "rx_out",
    "ch_init",
    "chb",
    "chc",
    "chd",
    "wa_init",
    "wa_b",
    "wa_c",
    "wa_d",
    "ra_out",
    "rc_out",
    "rs_out",
]
U8_CHANNELS = [
    "inherit_to_run_cmd",
    "reset_done",
    "e_to_c",
    "e_to_b",
    "c_to_a",
    "a_to_d",
    "d_to_e",
]

CHANNELS: list[tuple[str, sy.DataType]] = (
    [(name, sy.DataType.FLOAT64) for name in F64_CHANNELS]
    + [(name, sy.DataType.UINT8) for name in U8_CHANNELS]
    + [(name, sy.DataType.UINT32) for name in U32_CHANNELS]
    + [(name, sy.DataType.INT64) for name in I64_CHANNELS]
    + [(name, sy.DataType.STRING) for name in STR_CHANNELS]
)


class Variables(ArcCase):
    """Runtime coverage for scoped Arc variables and channel read/write variables.

    Literal, ChannelReadWrite, and ChannelRead variables across the f64/u32/i64/str data
    types, each read to its own channel, plus reassignment (literal overwrite,
    channel read/write rebind, channel-read re-express), and inheritance into inline bodies.
    """

    arc_source = ARC_VARIABLES_SOURCE
    arc_name_prefix = "ArcVariables"
    start_cmd_channel = "vars_start"
    subscribe_channels = OUTPUTS

    def setup(self) -> None:
        create_virtual_channels(self.client, CHANNELS)
        super().setup()

    def verify_sequence_execution(self) -> None:
        self._verify_literal()
        self._verify_channel_read_write()
        self._verify_channel_read()
        self._verify_inline_inheritance()
        self._verify_top_level_scope()
        self._verify_stage_scope()
        self._verify_kind_inheritance()
        self._verify_scope_reset_matrix()
        self._verify_reexpr()

    def _verify_literal(self) -> None:
        self.log("=== Literal ===")
        self.wait_for_eq("f64_initial", 10.0)
        self.wait_for_eq("u32_initial", 7)
        self.wait_for_eq("i64_initial", -5)
        self.wait_for_eq("str_initial", "hi")
        self.wait_for_eq("f64_final", 42.0)
        self.wait_for_eq("u32_final", 99)
        self.wait_for_eq("i64_final", -100)
        self.wait_for_eq("str_final", "bye")

    def _verify_channel_read_write(self) -> None:
        self.log("=== ChannelAlias ===")
        self.writer.write("channel_read_write_f64_a", 1.5)
        self.writer.write("channel_read_write_u32_a", 2)
        self.writer.write("channel_read_write_i64_a", -3)
        self.writer.write("channel_read_write_str_a", "a")
        self.wait_for_eq("f64_initial", 1.5)
        self.wait_for_eq("u32_initial", 2)
        self.wait_for_eq("i64_initial", -3)
        self.wait_for_eq("str_initial", "a")

        self.writer.write("channel_read_write_f64_b", 9.0)
        self.writer.write("channel_read_write_u32_b", 8)
        self.writer.write("channel_read_write_i64_b", -7)
        self.writer.write("channel_read_write_str_b", "b")
        self.wait_for_eq("f64_final", 9.0)
        self.wait_for_eq("u32_final", 8)
        self.wait_for_eq("i64_final", -7)
        self.wait_for_eq("str_final", "b")

    def _verify_channel_read(self) -> None:
        self.log("=== Reactive ===")
        self.writer.write("channel_read_f64", 2.0)
        self.writer.write("channel_read_u32", 5)
        self.writer.write("channel_read_i64", -4)
        self.writer.write("channel_read_str", "x")
        self.wait_for_eq("f64_initial", 3.0)
        self.wait_for_eq("u32_initial", 6)
        self.wait_for_eq("i64_initial", -3)
        self.wait_for_eq("str_initial", "x!")

        self.writer.write("channel_read_f64", 2.0)
        self.writer.write("channel_read_u32", 5)
        self.writer.write("channel_read_i64", -4)
        self.writer.write("channel_read_str", "x")
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
        self.log(
            "=== channel-read/write & channel-read inherited into a nested stage ==="
        )
        self.writer.write("inherit_channel_read_write_src", "A")
        self.writer.write("inherit_channel_read_src", "R")
        self.wait_for_eq("inherit_channel_read_write_direct", "A")
        self.wait_for_eq("inherit_channel_read_write_fmt", "a=A")
        self.wait_for_eq("inherit_channel_read_direct", "R!")

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
        self.log("=== reassignment across jumps (skip + reorder) ===")
        # reassign_main drives five reassignable variables through one jump path,
        # entry -> c -> a -> d -> e (skipping b), so each is exercised across the
        # out-of-order jump back into r_a (earlier in source than r_c / r_d):
        #   rx: channel-read re-expr    ra: channel-read/write read
        #   rc: literal re-expr    wa: channel-read/write write
        #   rs: stateful re-expr
        # Reads (rx / ra / rc / rs) converge on the live binding every stage. wa
        # writes are one-shot, so each lands a stage after its rebind (a same-stage
        # rebind+write races - all nodes in a stage fire at once): "entry" -> wa_init,
        # out-of-order "a" -> wa_c, forward "e" -> wa_d, then a sweep proves no leak.

        # entry: seed initial values; "entry" lands in the initial binding wa_init.
        self.writer.write("rx_src", "1")
        self.writer.write("ch_init", "init")
        self.wait_for_eq("rx_out", "init: 1")
        self.wait_for_eq("rc_out", "init")
        self.wait_for_eq("rs_out", "init")
        self.wait_for_eq("ra_out", "init")
        self.wait_for_eq("wa_init", "entry")

        # c: rebind ra -> chc and wa -> wa_c; reads reflect the new bindings.
        self.writer.write("e_to_c", 1)
        self.writer.write("rx_src", "2")
        self.writer.write("chc", "c1")
        self.wait_for_eq("rx_out", "c: 2")
        self.wait_for_eq("rc_out", "c")
        self.wait_for_eq("rs_out", "c")
        self.wait_for_eq("ra_out", "c1")

        # a: jump back to the earlier-in-source r_a. Reads still track chc, and the
        # out-of-order "a" write lands in the live wa_c, not the baked wa_init.
        self.writer.write("c_to_a", 1)
        self.writer.write("rx_src", "3")
        self.writer.write("chc", "c2")
        self.wait_for_eq("rx_out", "a: 3")
        self.wait_for_eq("rc_out", "c")
        self.wait_for_eq("rs_out", "c")
        self.wait_for_eq("ra_out", "c2")
        self.wait_for_eq("wa_c", "a")

        # d: rebind ra -> chd and wa -> wa_d.
        self.writer.write("a_to_d", 1)
        self.writer.write("rx_src", "4")
        self.writer.write("chd", "d")
        self.wait_for_eq("rx_out", "d: 4")
        self.wait_for_eq("rc_out", "d")
        self.wait_for_eq("rs_out", "d")
        self.wait_for_eq("ra_out", "d")

        # e: forward "e" write lands in wa_d; sweep confirms wa_c / wa_init intact.
        self.writer.write("d_to_e", 1)
        self.writer.write("rx_src", "5")
        self.wait_for_eq("rx_out", "e: 5")
        self.wait_for_eq("wa_d", "e")
        self.wait_for_eq("wa_c", "a")
        self.wait_for_eq("wa_init", "entry")

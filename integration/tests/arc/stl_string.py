#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_STL_STRING_SOURCE = """
// ──────────────────────────── string.len ─────────────────────────────
// len(const)
func len_cc(s str) i64 {
    return len("hello")
}
str_trigger -> len_cc{} -> len_cc_out
// len(chan)
func len_ch(s str) i64 {
    return len(s)
}
str_trigger -> len_ch{} -> len_ch_out
// ───────────────────────────── concat (+) ────────────────────────────
// concat(const, const)
func concat_cc(s str) i64 {
    return len("ab" + "cd")
}
str_trigger -> concat_cc{} -> concat_cc_out
// concat(chan, const)
func concat_xc(s str) i64 {
    return len(s + " world")
}
str_trigger -> concat_xc{} -> concat_xc_out
// concat(const, chan)
func concat_cx(s str) i64 {
    return len("prefix:" + s)
}
str_trigger -> concat_cx{} -> concat_cx_out
// concat(chan, chan)
func concat_xx(s str) {
    concat_xx_out = len(s + str_second)
}
str_trigger -> concat_xx{}
// ───────────────────────────── equal (==) ────────────────────────────
// equal(const, const)
func equal_cc(s str) u8 {
    return "abc" == "abc"
}
str_trigger -> equal_cc{} -> equal_cc_out
// equal(chan, const) — match
func equal_xc_match(s str) u8 {
    return s == "hello"
}
str_trigger -> equal_xc_match{} -> equal_xc_match_out
// equal(chan, const) — mismatch
func equal_xc_mismatch(s str) u8 {
    return s == "world"
}
str_trigger -> equal_xc_mismatch{} -> equal_xc_mismatch_out
// equal(const, chan)
func equal_cx(s str) u8 {
    return "hello" == s
}
str_trigger -> equal_cx{} -> equal_cx_out
// equal(chan, chan) — same
func equal_xx_same(s str) u8 {
    return s == s
}
str_trigger -> equal_xx_same{} -> equal_xx_same_out
// equal(chan, chan) — different
func equal_xx_diff(s str) {
    equal_xx_diff_out = s == str_second
}
str_trigger -> equal_xx_diff{}
// ──────────────────────────────── misc ───────────────────────────────
// nested concat
func concat_nested(s str) {
    concat_nested_out = len(s + "-" + str_second)
}
str_trigger -> concat_nested{}
// multi-string addition
func multi_add(s str) {
    multi_add_out = len(s + str_second + "_suffix" + str_third)
}
str_trigger -> multi_add{}
// ──────────────────────── format strings (function) ───────────────────
// One function exercises constants, local variables, and channel refs.
func fmt_fn() {
    a := 99
    b := 1.5
    fmt_const_int_fn_out = `int: {42}`
    fmt_const_hex_fn_out = `hex: {u8(255):x}`
    fmt_const_float_fn_out = `pi: {3.14159:.2f}`
    fmt_var_int_fn_out = `var int: {a}`
    fmt_var_float_fn_out = `var float: {b:.1f}`
    fmt_var_expr_fn_out = `expr: {a + 1}`
    fmt_chan_int_fn_out = `chan: {fmt_int_in}`
    fmt_chan_float_fn_out = `chan: {fmt_float_in:.2f}`
    fmt_chan_str_fn_out = `chan: {fmt_str_in:q}`
}
fmt_trigger -> fmt_fn{}
// ──────────────────────── format strings (flow) ───────────────────────
// Constants in flow position.
fmt_trigger -> `int: {42}` -> fmt_const_int_flow_out
fmt_trigger -> `hex: {u8(255):x}` -> fmt_const_hex_flow_out
fmt_trigger -> `pi: {3.14159:.2f}` -> fmt_const_float_flow_out
// Channel references in flow position.
fmt_trigger -> `chan: {fmt_int_in}` -> fmt_chan_int_flow_out
fmt_trigger -> `chan: {fmt_float_in:.2f}` -> fmt_chan_float_flow_out
fmt_trigger -> `chan: {fmt_str_in:q}` -> fmt_chan_str_flow_out
// Multiple placeholders in one flow expression.
fmt_trigger -> `i={fmt_int_in}, f={fmt_float_in:.1f}` -> fmt_multi_flow_out
"""

VIRTUAL_CHANNELS: list[tuple[str, sy.DataType]] = [
    ("str_trigger", sy.DataType.STRING),
    ("str_second", sy.DataType.STRING),
    ("str_third", sy.DataType.STRING),
    ("concat_xx_out", sy.DataType.INT64),
    ("equal_xx_diff_out", sy.DataType.UINT8),
    ("concat_nested_out", sy.DataType.INT64),
    ("multi_add_out", sy.DataType.INT64),
    # Format string inputs and outputs (all virtual to keep setup uniform).
    ("fmt_trigger", sy.DataType.UINT8),
    ("fmt_int_in", sy.DataType.INT64),
    ("fmt_float_in", sy.DataType.FLOAT64),
    ("fmt_str_in", sy.DataType.STRING),
    ("fmt_const_int_fn_out", sy.DataType.STRING),
    ("fmt_const_hex_fn_out", sy.DataType.STRING),
    ("fmt_const_float_fn_out", sy.DataType.STRING),
    ("fmt_var_int_fn_out", sy.DataType.STRING),
    ("fmt_var_float_fn_out", sy.DataType.STRING),
    ("fmt_var_expr_fn_out", sy.DataType.STRING),
    ("fmt_chan_int_fn_out", sy.DataType.STRING),
    ("fmt_chan_float_fn_out", sy.DataType.STRING),
    ("fmt_chan_str_fn_out", sy.DataType.STRING),
    ("fmt_const_int_flow_out", sy.DataType.STRING),
    ("fmt_const_hex_flow_out", sy.DataType.STRING),
    ("fmt_const_float_flow_out", sy.DataType.STRING),
    ("fmt_chan_int_flow_out", sy.DataType.STRING),
    ("fmt_chan_float_flow_out", sy.DataType.STRING),
    ("fmt_chan_str_flow_out", sy.DataType.STRING),
    ("fmt_multi_flow_out", sy.DataType.STRING),
]

INDEXED_CHANNELS: list[tuple[str, sy.DataType]] = [
    ("len_cc_out", sy.DataType.INT64),
    ("len_ch_out", sy.DataType.INT64),
    ("concat_cc_out", sy.DataType.INT64),
    ("concat_xc_out", sy.DataType.INT64),
    ("concat_cx_out", sy.DataType.INT64),
    ("equal_cc_out", sy.DataType.UINT8),
    ("equal_xc_match_out", sy.DataType.UINT8),
    ("equal_xc_mismatch_out", sy.DataType.UINT8),
    ("equal_cx_out", sy.DataType.UINT8),
    ("equal_xx_same_out", sy.DataType.UINT8),
]

ALL_CHANNELS = [name for name, _ in VIRTUAL_CHANNELS] + [
    name for name, _ in INDEXED_CHANNELS
]


class StlString(ArcConsoleCase):
    """Test string operations: len(), + concat, == equal.

    Primary axis: function (len, concat, equal).
    Secondary axis: input type (const/chan combinations).
    All triggered by writing to str_trigger string channel.
    """

    arc_source = ARC_STL_STRING_SOURCE
    arc_name_prefix = "ArcStlString"
    start_cmd_channel = "start_stl_string_cmd"
    subscribe_channels = ALL_CHANNELS

    def setup(self) -> None:
        for name, dtype in VIRTUAL_CHANNELS:
            create_virtual_channel(self.client, name, dtype)
        for name, dtype in INDEXED_CHANNELS:
            create_indexed_pair(self.client, name, dtype)
        super().setup()

    def _trigger(self) -> None:
        """Pre-write secondary channels, then trigger all functions."""
        self.writer.write("str_second", "other")
        self.writer.write("str_third", "!")
        self.writer.write("str_trigger", "hello")

    def _test_len(self) -> None:
        self.log("=== string.len ===")
        self._trigger()

        self.log("[len_cc] Expecting 5 (len('hello'))")
        self.wait_for_eq("len_cc_out", 5)

        self.log("[len_ch] Expecting 5 (len('hello') from channel)")
        self.wait_for_eq("len_ch_out", 5)

    def _test_concat(self) -> None:
        self.log("=== concat (+) ===")
        self._trigger()

        self.log("[concat_cc] Expecting 4 (len('abcd'))")
        self.wait_for_eq("concat_cc_out", 4)

        self.log("[concat_xc] Expecting 11 (len('hello world'))")
        self.wait_for_eq("concat_xc_out", 11)

        self.log("[concat_cx] Expecting 12 (len('prefix:hello'))")
        self.wait_for_eq("concat_cx_out", 12)

        self.log("[concat_xx] Expecting 10 (len('helloother'))")
        self.wait_for_eq("concat_xx_out", 10, is_virtual=True)

    def _test_equal(self) -> None:
        self.log("=== equal (==) ===")
        self._trigger()

        self.log("[equal_cc] Expecting 1 (equal('abc', 'abc'))")
        self.wait_for_eq("equal_cc_out", 1)

        self.log("[equal_xc_match] Expecting 1 (equal('hello', 'hello'))")
        self.wait_for_eq("equal_xc_match_out", 1)

        self.log("[equal_xc_mismatch] Expecting 0 (equal('hello', 'world'))")
        self.wait_for_eq("equal_xc_mismatch_out", 0)

        self.log("[equal_cx] Expecting 1 (equal('hello', 'hello'))")
        self.wait_for_eq("equal_cx_out", 1)

        self.log("[equal_xx_same] Expecting 1 (equal(s, s))")
        self.wait_for_eq("equal_xx_same_out", 1)

        self.log("[equal_xx_diff] Expecting 0 (equal('hello', 'other'))")
        self.wait_for_eq("equal_xx_diff_out", 0, is_virtual=True)

    def _test_misc(self) -> None:
        self.log("=== misc ===")
        self._trigger()

        self.log("[concat_nested] Expecting 11 (len('hello-other'))")
        self.wait_for_eq("concat_nested_out", 11, is_virtual=True)

        # "hello" + "other" + "_suffix" + "!" = "helloother_suffix!" = 18
        self.log("[multi_add] Expecting 18 (len('helloother_suffix!'))")
        self.wait_for_eq("multi_add_out", 18, is_virtual=True)

    def _test_format(self) -> None:
        """Format strings in function and flow contexts.

        Format strings are a language feature (parser + compiler + runtime),
        not part of the stl/strings module. They live here because they
        produce string values and are closely related to the other string
        operations exercised by this case; folding them in avoids the
        overhead of a separate test case with its own arc, channels, and
        teardown.

        Pre-writes input channels, then writes fmt_trigger to fire every flow
        and the fmt_fn function in a single pass. Each output asserts the
        end-to-end pipeline: parser, analyzer, compiler, runtime formatting.
        """
        self.log("=== format strings ===")
        self.writer.write("fmt_int_in", 42)
        self.writer.write("fmt_float_in", 2.71828)
        self.writer.write("fmt_str_in", "hello")
        self.writer.write("fmt_trigger", 1)

        # Function context: constants
        self.log("[fmt_const_int_fn] Expecting 'int: 42'")
        self.wait_for_eq("fmt_const_int_fn_out", "int: 42", is_virtual=True)
        self.log("[fmt_const_hex_fn] Expecting 'hex: ff'")
        self.wait_for_eq("fmt_const_hex_fn_out", "hex: ff", is_virtual=True)
        self.log("[fmt_const_float_fn] Expecting 'pi: 3.14'")
        self.wait_for_eq("fmt_const_float_fn_out", "pi: 3.14", is_virtual=True)

        # Function context: local variables
        self.log("[fmt_var_int_fn] Expecting 'var int: 99'")
        self.wait_for_eq("fmt_var_int_fn_out", "var int: 99", is_virtual=True)
        self.log("[fmt_var_float_fn] Expecting 'var float: 1.5'")
        self.wait_for_eq("fmt_var_float_fn_out", "var float: 1.5", is_virtual=True)
        self.log("[fmt_var_expr_fn] Expecting 'expr: 100'")
        self.wait_for_eq("fmt_var_expr_fn_out", "expr: 100", is_virtual=True)

        # Function context: channel references
        self.log("[fmt_chan_int_fn] Expecting 'chan: 42'")
        self.wait_for_eq("fmt_chan_int_fn_out", "chan: 42", is_virtual=True)
        self.log("[fmt_chan_float_fn] Expecting 'chan: 2.72'")
        self.wait_for_eq("fmt_chan_float_fn_out", "chan: 2.72", is_virtual=True)
        self.log("[fmt_chan_str_fn] Expecting 'chan: \"hello\"'")
        self.wait_for_eq("fmt_chan_str_fn_out", 'chan: "hello"', is_virtual=True)

        # Flow context: constants
        self.log("[fmt_const_int_flow] Expecting 'int: 42'")
        self.wait_for_eq("fmt_const_int_flow_out", "int: 42", is_virtual=True)
        self.log("[fmt_const_hex_flow] Expecting 'hex: ff'")
        self.wait_for_eq("fmt_const_hex_flow_out", "hex: ff", is_virtual=True)
        self.log("[fmt_const_float_flow] Expecting 'pi: 3.14'")
        self.wait_for_eq("fmt_const_float_flow_out", "pi: 3.14", is_virtual=True)

        # Flow context: channel references
        self.log("[fmt_chan_int_flow] Expecting 'chan: 42'")
        self.wait_for_eq("fmt_chan_int_flow_out", "chan: 42", is_virtual=True)
        self.log("[fmt_chan_float_flow] Expecting 'chan: 2.72'")
        self.wait_for_eq("fmt_chan_float_flow_out", "chan: 2.72", is_virtual=True)
        self.log("[fmt_chan_str_flow] Expecting 'chan: \"hello\"'")
        self.wait_for_eq("fmt_chan_str_flow_out", 'chan: "hello"', is_virtual=True)

        # Flow context: multiple placeholders
        self.log("[fmt_multi_flow] Expecting 'i=42, f=2.7'")
        self.wait_for_eq("fmt_multi_flow_out", "i=42, f=2.7", is_virtual=True)

    def verify_sequence_execution(self) -> None:
        self._test_len()
        self._test_concat()
        self._test_equal()
        self._test_misc()
        self._test_format()

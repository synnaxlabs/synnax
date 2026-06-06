#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from tests.arc.arc import ArcCase

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
    fmt_const_int_fn_out = f"int: {42}"
    fmt_const_hex_fn_out = f"hex: {u8(255):x}"
    fmt_const_float_fn_out = f"pi: {3.14159:.2f}"
    fmt_var_int_fn_out = f"var int: {a}"
    fmt_var_float_fn_out = f"var float: {b:.1f}"
    fmt_var_expr_fn_out = f"expr: {a + 1}"
    fmt_chan_int_fn_out = f"chan: {fmt_int_in}"
    fmt_chan_float_fn_out = f"chan: {fmt_float_in:.2f}"
    fmt_chan_str_fn_out = f"chan: {fmt_str_in:q}"
    // Verb coverage: every numeric verb supported by the analyzer.
    fmt_bin_fn_out = f"{5:b}"
    fmt_oct_fn_out = f"{8:o}"
    fmt_goct_fn_out = f"{8:O}"
    fmt_hex_upper_fn_out = f"{u8(255):X}"
    fmt_rune_ascii_fn_out = f"{u32(65):c}"
    fmt_rune_utf8_fn_out = f"{u32(9731):c}"
    fmt_sci_lower_fn_out = f"{1000000.0:e}"
    fmt_sci_upper_fn_out = f"{1000000.0:E}"
    fmt_short_fn_out = f"{3.14:g}"
    // Alt flag (#): parity with Go's prefix-on-zero rule.
    // %#x on 0 emits "0x0"; %#o on 0 emits "0" (no prefix).
    fmt_alt_hex_zero_fn_out = f"{u8(0):#x}"
    fmt_alt_oct_zero_fn_out = f"{u8(0):#o}"
    fmt_alt_bin_fn_out = f"{5:#b}"
    // Width, precision, sign flags.
    fmt_width_fn_out = f"{42:5d}"
    fmt_left_fn_out = f"{42:-5d}"
    fmt_zero_pad_fn_out = f"{42:05d}"
    fmt_plus_fn_out = f"{42:+d}"
    fmt_prec_int_fn_out = f"{42:.4d}"
    // Negative ints with non-decimal verbs: Go preserves the sign on the
    // magnitude, unlike C printf which treats as unsigned.
    fmt_neg_hex_fn_out = f"{-255:x}"
    fmt_neg_alt_hex_fn_out = f"{-255:#x}"
    fmt_neg_bin_fn_out = f"{-5:b}"
    // String width and precision count UTF-8 runes, not bytes.
    fmt_utf8_width_fn_out = f"{fmt_utf8_in:6s}"
    fmt_utf8_prec_fn_out = f"{fmt_utf8_in:.3s}"
    // Doubled-brace literal-brace escapes: {{ -> { and }} -> }.
    fmt_brace_pair_fn_out = f"{{a}}"
    fmt_brace_around_fn_out = f"{{{42}}}"
    fmt_brace_path_fn_out = rf"C:\logs\{{abc}}.txt"
    fmt_brace_path_int_fn_out = rf"C:\logs\{{{42}}}.txt"
}
fmt_trigger -> fmt_fn{}
// ──────────────────────── format strings (flow) ───────────────────────
// Constants in flow position.
fmt_trigger -> f"int: {42}" -> fmt_const_int_flow_out
fmt_trigger -> f"hex: {u8(255):x}" -> fmt_const_hex_flow_out
fmt_trigger -> f"pi: {3.14159:.2f}" -> fmt_const_float_flow_out
// Channel references in flow position.
fmt_trigger -> f"chan: {fmt_int_in}" -> fmt_chan_int_flow_out
fmt_trigger -> f"chan: {fmt_float_in:.2f}" -> fmt_chan_float_flow_out
fmt_trigger -> f"chan: {fmt_str_in:q}" -> fmt_chan_str_flow_out
// Multiple placeholders in one flow expression.
fmt_trigger -> f"i={fmt_int_in}, f={fmt_float_in:.1f}" -> fmt_multi_flow_out
// Doubled-brace literal-brace escapes in flow position. This path is the
// one that historically dropped the {{/}} collapse and emitted the raw
// body verbatim; the assertions below pin the fixed behavior.
fmt_trigger -> rf"C:\logs\{{abc}}.txt" -> fmt_brace_path_flow_out
fmt_trigger -> rf"C:\logs\{42}.txt" -> fmt_brace_backslash_flow_out
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
    ("fmt_utf8_in", sy.DataType.STRING),
    ("fmt_const_int_fn_out", sy.DataType.STRING),
    ("fmt_const_hex_fn_out", sy.DataType.STRING),
    ("fmt_const_float_fn_out", sy.DataType.STRING),
    ("fmt_var_int_fn_out", sy.DataType.STRING),
    ("fmt_var_float_fn_out", sy.DataType.STRING),
    ("fmt_var_expr_fn_out", sy.DataType.STRING),
    ("fmt_chan_int_fn_out", sy.DataType.STRING),
    ("fmt_chan_float_fn_out", sy.DataType.STRING),
    ("fmt_chan_str_fn_out", sy.DataType.STRING),
    # Verb coverage outputs.
    ("fmt_bin_fn_out", sy.DataType.STRING),
    ("fmt_oct_fn_out", sy.DataType.STRING),
    ("fmt_goct_fn_out", sy.DataType.STRING),
    ("fmt_hex_upper_fn_out", sy.DataType.STRING),
    ("fmt_rune_ascii_fn_out", sy.DataType.STRING),
    ("fmt_rune_utf8_fn_out", sy.DataType.STRING),
    ("fmt_sci_lower_fn_out", sy.DataType.STRING),
    ("fmt_sci_upper_fn_out", sy.DataType.STRING),
    ("fmt_short_fn_out", sy.DataType.STRING),
    # Alt-flag (#) parity outputs.
    ("fmt_alt_hex_zero_fn_out", sy.DataType.STRING),
    ("fmt_alt_oct_zero_fn_out", sy.DataType.STRING),
    ("fmt_alt_bin_fn_out", sy.DataType.STRING),
    # Width, precision, sign flags.
    ("fmt_width_fn_out", sy.DataType.STRING),
    ("fmt_left_fn_out", sy.DataType.STRING),
    ("fmt_zero_pad_fn_out", sy.DataType.STRING),
    ("fmt_plus_fn_out", sy.DataType.STRING),
    ("fmt_prec_int_fn_out", sy.DataType.STRING),
    # Negative non-decimal parity outputs.
    ("fmt_neg_hex_fn_out", sy.DataType.STRING),
    ("fmt_neg_alt_hex_fn_out", sy.DataType.STRING),
    ("fmt_neg_bin_fn_out", sy.DataType.STRING),
    # UTF-8 rune-count parity outputs.
    ("fmt_utf8_width_fn_out", sy.DataType.STRING),
    ("fmt_utf8_prec_fn_out", sy.DataType.STRING),
    # Literal-brace escape ({{ / }}) outputs.
    ("fmt_brace_pair_fn_out", sy.DataType.STRING),
    ("fmt_brace_around_fn_out", sy.DataType.STRING),
    ("fmt_brace_path_fn_out", sy.DataType.STRING),
    ("fmt_brace_path_int_fn_out", sy.DataType.STRING),
    ("fmt_brace_path_flow_out", sy.DataType.STRING),
    ("fmt_brace_backslash_flow_out", sy.DataType.STRING),
    ("fmt_const_int_flow_out", sy.DataType.STRING),
    ("fmt_const_hex_flow_out", sy.DataType.STRING),
    ("fmt_const_float_flow_out", sy.DataType.STRING),
    ("fmt_chan_int_flow_out", sy.DataType.STRING),
    ("fmt_chan_float_flow_out", sy.DataType.STRING),
    ("fmt_chan_str_flow_out", sy.DataType.STRING),
    ("fmt_multi_flow_out", sy.DataType.STRING),
    # len / concat / equal outputs. Virtual (stream-read) rather than indexed
    # (DB-read): these assert pure STL-function results that only need to flow
    # through, so reading them off the subscription avoids the database
    # round-trip and its not-ready retry penalty.
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

ALL_CHANNELS = [name for name, _ in VIRTUAL_CHANNELS]


class StlString(ArcCase):
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
        self.client.channels.create(
            [
                sy.Channel(name=name, data_type=dtype, virtual=True)
                for name, dtype in VIRTUAL_CHANNELS
            ],
            retrieve_if_name_exists=True,
        )
        super().setup()

    def _trigger(self) -> None:
        """Pre-write secondary channels, then trigger all functions."""
        self.writer.write("str_second", "other")
        self.writer.write("str_third", "!")
        self.writer.write("str_trigger", "hello")

    def _test_len(self) -> None:
        self.log("=== string.len ===")
        self.log("[len_cc] Expecting 5 (len('hello'))")
        self.wait_for_eq("len_cc_out", 5)

        self.log("[len_ch] Expecting 5 (len('hello') from channel)")
        self.wait_for_eq("len_ch_out", 5)

    def _test_concat(self) -> None:
        self.log("=== concat (+) ===")
        self.log("[concat_cc] Expecting 4 (len('abcd'))")
        self.wait_for_eq("concat_cc_out", 4)

        self.log("[concat_xc] Expecting 11 (len('hello world'))")
        self.wait_for_eq("concat_xc_out", 11)

        self.log("[concat_cx] Expecting 12 (len('prefix:hello'))")
        self.wait_for_eq("concat_cx_out", 12)

        self.log("[concat_xx] Expecting 10 (len('helloother'))")
        self.wait_for_eq("concat_xx_out", 10)

    def _test_equal(self) -> None:
        self.log("=== equal (==) ===")
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
        self.wait_for_eq("equal_xx_diff_out", 0)

    def _test_misc(self) -> None:
        self.log("=== misc ===")
        self.log("[concat_nested] Expecting 11 (len('hello-other'))")
        self.wait_for_eq("concat_nested_out", 11)

        # "hello" + "other" + "_suffix" + "!" = "helloother_suffix!" = 18
        self.log("[multi_add] Expecting 18 (len('helloother_suffix!'))")
        self.wait_for_eq("multi_add_out", 18)

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
        # 5 runes / 6 bytes — exercises Go's rune-count width and precision.
        self.writer.write("fmt_utf8_in", "héllo")
        self.writer.write("fmt_trigger", 1)

        # Function context: constants
        self.log("[fmt_const_int_fn] Expecting 'int: 42'")
        self.wait_for_eq("fmt_const_int_fn_out", "int: 42")
        self.log("[fmt_const_hex_fn] Expecting 'hex: ff'")
        self.wait_for_eq("fmt_const_hex_fn_out", "hex: ff")
        self.log("[fmt_const_float_fn] Expecting 'pi: 3.14'")
        self.wait_for_eq("fmt_const_float_fn_out", "pi: 3.14")

        # Function context: local variables
        self.log("[fmt_var_int_fn] Expecting 'var int: 99'")
        self.wait_for_eq("fmt_var_int_fn_out", "var int: 99")
        self.log("[fmt_var_float_fn] Expecting 'var float: 1.5'")
        self.wait_for_eq("fmt_var_float_fn_out", "var float: 1.5")
        self.log("[fmt_var_expr_fn] Expecting 'expr: 100'")
        self.wait_for_eq("fmt_var_expr_fn_out", "expr: 100")

        # Function context: channel references
        self.log("[fmt_chan_int_fn] Expecting 'chan: 42'")
        self.wait_for_eq("fmt_chan_int_fn_out", "chan: 42")
        self.log("[fmt_chan_float_fn] Expecting 'chan: 2.72'")
        self.wait_for_eq("fmt_chan_float_fn_out", "chan: 2.72")
        self.log("[fmt_chan_str_fn] Expecting 'chan: \"hello\"'")
        self.wait_for_eq("fmt_chan_str_fn_out", 'chan: "hello"')

        # Flow context: constants
        self.log("[fmt_const_int_flow] Expecting 'int: 42'")
        self.wait_for_eq("fmt_const_int_flow_out", "int: 42")
        self.log("[fmt_const_hex_flow] Expecting 'hex: ff'")
        self.wait_for_eq("fmt_const_hex_flow_out", "hex: ff")
        self.log("[fmt_const_float_flow] Expecting 'pi: 3.14'")
        self.wait_for_eq("fmt_const_float_flow_out", "pi: 3.14")

        # Flow context: channel references
        self.log("[fmt_chan_int_flow] Expecting 'chan: 42'")
        self.wait_for_eq("fmt_chan_int_flow_out", "chan: 42")
        self.log("[fmt_chan_float_flow] Expecting 'chan: 2.72'")
        self.wait_for_eq("fmt_chan_float_flow_out", "chan: 2.72")
        self.log("[fmt_chan_str_flow] Expecting 'chan: \"hello\"'")
        self.wait_for_eq("fmt_chan_str_flow_out", 'chan: "hello"')

        # Flow context: multiple placeholders
        self.log("[fmt_multi_flow] Expecting 'i=42, f=2.7'")
        self.wait_for_eq("fmt_multi_flow_out", "i=42, f=2.7")

        # Verb coverage: one case per verb supported by the analyzer.
        self.log("[fmt_bin] Expecting '101'")
        self.wait_for_eq("fmt_bin_fn_out", "101")
        self.log("[fmt_oct] Expecting '10'")
        self.wait_for_eq("fmt_oct_fn_out", "10")
        self.log("[fmt_goct] Expecting '0o10'")
        self.wait_for_eq("fmt_goct_fn_out", "0o10")
        self.log("[fmt_hex_upper] Expecting 'FF'")
        self.wait_for_eq("fmt_hex_upper_fn_out", "FF")
        self.log("[fmt_rune_ascii] Expecting 'A'")
        self.wait_for_eq("fmt_rune_ascii_fn_out", "A")
        self.log("[fmt_rune_utf8] Expecting '☃'")
        self.wait_for_eq("fmt_rune_utf8_fn_out", "☃")
        self.log("[fmt_sci_lower] Expecting '1.000000e+06'")
        self.wait_for_eq("fmt_sci_lower_fn_out", "1.000000e+06")
        self.log("[fmt_sci_upper] Expecting '1.000000E+06'")
        self.wait_for_eq("fmt_sci_upper_fn_out", "1.000000E+06")
        self.log("[fmt_short] Expecting '3.14'")
        self.wait_for_eq("fmt_short_fn_out", "3.14")

        # Alt flag (#) on zero: Go emits "0x0"/"0b0" but suppresses for octal.
        self.log("[fmt_alt_hex_zero] Expecting '0x0'")
        self.wait_for_eq("fmt_alt_hex_zero_fn_out", "0x0")
        self.log("[fmt_alt_oct_zero] Expecting '0'")
        self.wait_for_eq("fmt_alt_oct_zero_fn_out", "0")
        self.log("[fmt_alt_bin] Expecting '0b101'")
        self.wait_for_eq("fmt_alt_bin_fn_out", "0b101")

        # Width, precision, sign flags.
        self.log("[fmt_width] Expecting '   42'")
        self.wait_for_eq("fmt_width_fn_out", "   42")
        self.log("[fmt_left] Expecting '42   '")
        self.wait_for_eq("fmt_left_fn_out", "42   ")
        self.log("[fmt_zero_pad] Expecting '00042'")
        self.wait_for_eq("fmt_zero_pad_fn_out", "00042")
        self.log("[fmt_plus] Expecting '+42'")
        self.wait_for_eq("fmt_plus_fn_out", "+42")
        self.log("[fmt_prec_int] Expecting '0042'")
        self.wait_for_eq("fmt_prec_int_fn_out", "0042")

        # Negative ints with non-decimal verbs: sign preserved per Go.
        self.log("[fmt_neg_hex] Expecting '-ff'")
        self.wait_for_eq("fmt_neg_hex_fn_out", "-ff")
        self.log("[fmt_neg_alt_hex] Expecting '-0xff'")
        self.wait_for_eq("fmt_neg_alt_hex_fn_out", "-0xff")
        self.log("[fmt_neg_bin] Expecting '-101'")
        self.wait_for_eq("fmt_neg_bin_fn_out", "-101")

        # UTF-8 width and precision: Go counts code points, not bytes.
        # "héllo" is 5 runes / 6 bytes; %6s pads to 6 runes, %.3s keeps 3.
        self.log("[fmt_utf8_width] Expecting ' héllo'")
        self.wait_for_eq("fmt_utf8_width_fn_out", " héllo")
        self.log("[fmt_utf8_prec] Expecting 'hél'")
        self.wait_for_eq("fmt_utf8_prec_fn_out", "hél")

        # Literal-brace escapes: {{ -> { and }} -> }.
        self.log("[fmt_brace_pair] Expecting '{a}'")
        self.wait_for_eq("fmt_brace_pair_fn_out", "{a}")
        self.log("[fmt_brace_around] Expecting '{42}'")
        self.wait_for_eq("fmt_brace_around_fn_out", "{42}")
        self.log("[fmt_brace_path] Expecting 'C:\\logs\\{abc}.txt'")
        self.wait_for_eq("fmt_brace_path_fn_out", r"C:\logs\{abc}.txt")
        self.log("[fmt_brace_path_int] Expecting 'C:\\logs\\{42}.txt'")
        self.wait_for_eq("fmt_brace_path_int_fn_out", r"C:\logs\{42}.txt")
        self.log("[fmt_brace_path_flow] Expecting 'C:\\logs\\{abc}.txt'")
        self.wait_for_eq("fmt_brace_path_flow_out", r"C:\logs\{abc}.txt")
        self.log("[fmt_brace_backslash_flow] Expecting 'C:\\logs\\42.txt'")
        self.wait_for_eq("fmt_brace_backslash_flow_out", r"C:\logs\42.txt")

    def verify_sequence_execution(self) -> None:
        # len, concat, equal, and misc are all gated on str_trigger, so a single
        # write computes every output in one pass; the groups below just assert
        # the results.
        self._trigger()
        self._test_len()
        self._test_concat()
        self._test_equal()
        self._test_misc()
        self._test_format()

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

ARC_STL_STRING_SOURCE = """
// ──────────────────────────── string.len ─────────────────────────────

// len(const)
func len_cc(s str) i64 { return string.len("hello") }
str_trigger -> len_cc{} -> len_cc_out

// len(chan)
func len_ch(s str) i64 { return string.len(s) }
str_trigger -> len_ch{} -> len_ch_out

// ─────────────────────────── string.concat ───────────────────────────

// concat(const, const)
func concat_cc(s str) i64 { return string.len(string.concat("ab", "cd")) }
str_trigger -> concat_cc{} -> concat_cc_out

// concat(chan, const)
func concat_xc(s str) i64 { return string.len(string.concat(s, " world")) }
str_trigger -> concat_xc{} -> concat_xc_out

// concat(const, chan)
func concat_cx(s str) i64 { return string.len(string.concat("prefix:", s)) }
str_trigger -> concat_cx{} -> concat_cx_out

// concat(chan, chan)
func concat_xx(s str) { concat_xx_out = string.len(string.concat(s, str_second)) }
str_trigger -> concat_xx{}

// ─────────────────────────── string.equal ────────────────────────────

// equal(const, const)
func equal_cc(s str) i32 { return string.equal("abc", "abc") }
str_trigger -> equal_cc{} -> equal_cc_out

// equal(chan, const) — match
func equal_xc_match(s str) i32 { return string.equal(s, "hello") }
str_trigger -> equal_xc_match{} -> equal_xc_match_out

// equal(chan, const) — mismatch
func equal_xc_mismatch(s str) i32 { return string.equal(s, "world") }
str_trigger -> equal_xc_mismatch{} -> equal_xc_mismatch_out

// equal(const, chan)
func equal_cx(s str) i32 { return string.equal("hello", s) }
str_trigger -> equal_cx{} -> equal_cx_out

// equal(chan, chan) — same
func equal_xx_same(s str) i32 { return string.equal(s, s) }
str_trigger -> equal_xx_same{} -> equal_xx_same_out

// equal(chan, chan) — different
func equal_xx_diff(s str) { equal_xx_diff_out = string.equal(s, str_second) }
str_trigger -> equal_xx_diff{}

// ──────────────────────────────── misc ───────────────────────────────

// nested concat
func concat_nested(s str) {
    concat_nested_out = string.len(string.concat(string.concat(s, "-"), str_second))
}
str_trigger -> concat_nested{}

// multi-string addition
func multi_add(s str) {
    multi_add_out = string.len(s + str_second + "_suffix" + str_third)
}
str_trigger -> multi_add{}
"""

VIRTUAL_CHANNELS: list[tuple[str, sy.DataType]] = [
    ("str_trigger", sy.DataType.STRING),
    ("str_second", sy.DataType.STRING),
    ("str_third", sy.DataType.STRING),
    ("concat_xx_out", sy.DataType.INT64),
    ("equal_xx_diff_out", sy.DataType.INT32),
    ("concat_nested_out", sy.DataType.INT64),
    ("multi_add_out", sy.DataType.INT64),
]

INDEXED_CHANNELS: list[tuple[str, sy.DataType]] = [
    ("len_cc_out", sy.DataType.INT64),
    ("len_ch_out", sy.DataType.INT64),
    ("concat_cc_out", sy.DataType.INT64),
    ("concat_xc_out", sy.DataType.INT64),
    ("concat_cx_out", sy.DataType.INT64),
    ("equal_cc_out", sy.DataType.INT32),
    ("equal_xc_match_out", sy.DataType.INT32),
    ("equal_xc_mismatch_out", sy.DataType.INT32),
    ("equal_cx_out", sy.DataType.INT32),
    ("equal_xx_same_out", sy.DataType.INT32),
]

ALL_CHANNELS = [name for name, _ in VIRTUAL_CHANNELS] + [
    name for name, _ in INDEXED_CHANNELS
]


class StlString(ArcConsoleCase):
    """Test string module with qualified syntax: string.len(),
    string.concat(), string.equal().

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
            idx = self.client.channels.create(
                name=f"{name}_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            self.client.channels.create(
                name=name,
                data_type=dtype,
                index=idx.key,
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
        self._trigger()

        self.log("[len_cc] Expecting 5 (len('hello'))")
        self.wait_for_eq("len_cc_out", 5)

        self.log("[len_ch] Expecting 5 (len('hello') from channel)")
        self.wait_for_eq("len_ch_out", 5)

    def _test_concat(self) -> None:
        self.log("=== string.concat ===")
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
        self.log("=== string.equal ===")
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

    def verify_sequence_execution(self) -> None:
        self._test_len()
        self._test_concat()
        self._test_equal()
        self._test_misc()

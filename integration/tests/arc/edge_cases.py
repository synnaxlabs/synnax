#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Sequence
from dataclasses import dataclass, field

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc import ArcCase

# ── Main arc source: channel propagation edge cases (valid, runs at runtime) ──

ARC_CHANNEL_EDGE_CASES_SOURCE = """
// Edge case 1: Multiple callees write the same channel (last write wins).
func set_to_one() {
    edge_same_ch = 1.0
}
func set_to_two() {
    edge_same_ch = 2.0
}
func test_same_channel_write() {
    set_to_one()
    set_to_two()
}
// Edge case 2: Diamond dependency (value through two paths).
func diamond_leaf() {
    edge_diamond = 42.0
}
func diamond_left() {
    diamond_leaf()
}
func diamond_right() {
    diamond_leaf()
}
func test_diamond() {
    diamond_left()
    diamond_right()
}
// Edge case 3: Multiple callees write different channels.
func write_a() {
    edge_multi_a = 10.0
}
func write_b() {
    edge_multi_b = 20.0
}
func test_multi_callee() {
    write_a()
    write_b()
}
// Edge case 4: Multi-level chain (top -> mid -> leaf writes).
func chain_leaf() {
    edge_chain = 99.0
}
func chain_mid() {
    chain_leaf()
}
func test_chain() {
    chain_mid()
}
// Edge case 5: Forward reference (caller declared before callee).
func test_fwd_ref() {
    fwd_callee()
}
func fwd_callee() {
    edge_fwd = 55.0
}
// Edge case 6: Basic write through chan input param.
func chan_write() {ch chan f32} {
    ch = 77.0
}
func test_chan_param_write() {
    chan_write(edge_chan_basic)
}
// Edge case 7: Multi-level chain through chan params.
func chan_chain_leaf() {ch chan f32} {
    ch = 88.0
}
func chan_chain_mid() {ch chan f32} {
    chan_chain_leaf(ch)
}
func test_chan_chain() {
    chan_chain_mid(edge_chan_chain)
}
// Edge case 8: Same function called with different chan args.
func chan_set() {ch chan f32} {
    ch = 33.0
}
func test_chan_diff_args() {
    chan_set(edge_chan_arg_a)
    chan_set(edge_chan_arg_b)
}
// Edge case 9: Multiple chan params written in one function.
func chan_multi_write() {a chan f32, b chan f32} {
    a = 11.0
    b = 22.0
}
func test_chan_multi_param() {
    chan_multi_write(edge_chan_mp_a, edge_chan_mp_b)
}
// Edge case 10: Forward reference with chan param.
func test_chan_fwd_ref() {
    chan_fwd_callee(edge_chan_fwd)
}
func chan_fwd_callee() {ch chan f32} {
    ch = 66.0
}

start_edge_case_cmd -> test_same_channel_write()
start_edge_case_cmd -> test_diamond()
start_edge_case_cmd -> test_multi_callee()
start_edge_case_cmd -> test_chain()
start_edge_case_cmd -> test_fwd_ref()
start_edge_case_cmd -> test_chan_param_write()
start_edge_case_cmd -> test_chan_chain()
start_edge_case_cmd -> test_chan_diff_args()
start_edge_case_cmd -> test_chan_multi_param()
start_edge_case_cmd -> test_chan_fwd_ref()
"""

# ── Read-only monitor (no write channels, only status.set) ──
# Regression test: the C++ driver unconditionally created writer/streamer
# pipelines even when write_channels was empty, causing the server to reject
# the empty keys list.

ARC_READ_ONLY_MONITOR = """
import status
start_read_only_monitor_cmd => main

sequence main {
    stage on {
        edge_monitor_input > 30 => status.set(
            key_or_name = "press_monitor_status",
            variant = "warning",
            message = "Pressure high"
        )
        edge_monitor_input < 30 => status.set(
            key_or_name = "press_monitor_status",
            variant = "success",
            message = "Pressure nominal"
        )
    }
}
"""

# ── Circular dependency sources (invalid, caught at configure time) ──
# Comprehensive topology coverage is in the Go unit tests
# (arc/go/analyzer/analyzer_test.go). These integration tests verify that
# circular detection works end-to-end through the Python client.

# a -> a
ARC_SELF_REC = """
func self_rec() {
    ch1 = 1.0
    self_rec()
}
start_edge_case_cmd -> self_rec()
"""

# Callee called in ALL branches of if-else (no exit path)
ARC_ALL_BRANCHES = """
func ping() {
    ch1 = 1.0
    if ch1 > 0 {
        pong()
    } else {
        pong()
    }
}
func pong() {
    ping()
}
start_edge_case_cmd -> ping()
"""

# Tangled web of 5 functions forming one big cycle. Branches decide arithmetic,
# calls are unconditional after the branching. No exit path anywhere.
ARC_TANGLED_WEB = """
func init_seq() {
    ch1 = 1.0
    if ch1 > 50 {
        ch1 = ch1 + 20.0
    } else if ch1 > 20 {
        ch1 = ch1 + 5.0
    }
    proc_alpha()
}
func proc_alpha() {
    if ch1 > 80 {
        ch1 = ch1 - 30.0
    } else {
        ch1 = ch1 + 10.0
    }
    xform()
}
func xform() {
    ch1 = ch1 + 15.0
    if ch1 > 120 {
        ch1 = 100.0
    } else {
        ch1 = ch1 + 20.0
    }
    route_beta()
}
func route_beta() {
    if ch1 > 90 {
        ch1 = ch1 - 40.0
    } else if ch1 > 60 {
        ch1 = ch1 + 15.0
    }
    commit()
}
func commit() {
    ch1 = 50.0
    init_seq()
}
start_edge_case_cmd -> init_seq()
"""

# ── ExecContext violations (invalid, caught at configure time) ──

# Flow-only function called in a func body
ARC_FLOW_IN_FUNC = """
func bad() {
    ch1 = 1.0
    interval(100)
}
start_edge_case_cmd -> bad()
"""


# ── time.format with invalid inputs (runs at runtime) ──
# Each context gets its own Arc, so a warning traces to one of them.

ARC_FORMAT_FN = """
import time
func format_fn() {
    edge_fmt_fn_out = time.format(edge_fmt_ts_in, edge_fmt_in, edge_fmt_tz_in)
}
edge_fmt_tz_in -> format_fn()
"""

ARC_FORMAT_FLOW = """
import time
time.format(edge_fmt_ts_in, edge_fmt_in, edge_fmt_tz_in) -> edge_fmt_flow_out
"""

ARC_FORMAT_LITERAL_ZONE_FN = """
import time
func format_lit_fn() {
    edge_fmt_lit_fn_out = time.format(edge_fmt_ts_in, "%H:%M", "Mars/Olympus")
}
edge_fmt_ts_in -> format_lit_fn()
"""

ARC_FORMAT_LITERAL_ZONE_FLOW = """
import time
time.format(edge_fmt_ts_in, "%H:%M", "Mars/Olympus") -> edge_fmt_lit_flow_out
"""

# A Saturday in March before daylight time starts.
FORMAT_STAMP = 1_772_895_845_000_000_000


@dataclass
class FormatInvalidCase:
    label: str
    fmt: str
    zone: str


FORMAT_INVALID_CASES = [
    FormatInvalidCase("UnknownZone", "%H:%M %Z", "Mars/Olympus"),
    FormatInvalidCase("UnknownZoneISO", "", "Mars/Phobos"),
    FormatInvalidCase("UnknownZoneUnknownCode", "%Q", "Mars/Deimos"),
    FormatInvalidCase("AbbreviationNotAZone", "%H:%M", "PST"),
    FormatInvalidCase("PaddedZone", "%H:%M", " America/Denver"),
    FormatInvalidCase("PartialZone", "%H:%M", "America/"),
]


@dataclass
class FormatOddCase:
    label: str
    fmt: str
    zone: str
    expected: str


# Inputs that look wrong but have a defined output. Consecutive expectations differ,
# so each wait proves a new write landed.
FORMAT_ODD_CASES = [
    FormatOddCase("UnknownCodeInAZone", "%Q %Z", "America/Denver", "%Q MST"),
    FormatOddCase("TrailingPercentInAZone", "%H %", "Europe/Berlin", "16 %"),
    FormatOddCase("UpperCaseZone", "%Z", "AMERICA/DENVER", "MST"),
    FormatOddCase("EmptyZone", "%Z %z", "", "UTC +0000"),
    FormatOddCase("FixedZone", "%Z %z", "EST", "EST -0500"),
]


@dataclass
class Case:
    label: str
    source: str
    wait_substr: str = ""
    expect: list[str] = field(default_factory=list)


EXEC_CONTEXT_CASES = [
    Case(
        "FlowInFunc",
        ARC_FLOW_IN_FUNC,
        "cannot be called inside a func block",
    ),
]

# ── WASM-only callee as a flow node (valid, runs at runtime) ──

ARC_WASM_IN_FLOW = """
len(edge_len_input) -> edge_len_out
"""

# ── Import diagnostic sources (invalid, caught at configure time) ──
# Verify the four import diagnostics surface end-to-end through the Python client.
# Comprehensive coverage is in arc/go/import_test.go.

# Qualified module member used without an `import (…)` block.
ARC_IMPORT_MISSING = """
func get_now() i64 ns {
    return time.now()
}
start_edge_case_cmd -> get_now()
"""

# Module imported but never referenced.
ARC_IMPORT_UNUSED = """
import time
func passthrough() {
    ch1 = ch1 + 1.0
}
start_edge_case_cmd -> passthrough()
"""

# Import names a module the resolver does not expose.
ARC_IMPORT_UNKNOWN = """
import banana
func passthrough() {
    ch1 = ch1 + 1.0
}
start_edge_case_cmd -> passthrough()
"""

# Same module listed twice in one block.
ARC_IMPORT_DUPLICATE = """
import ( time time )
func get_now() i64 ns {
    return time.now()
}
start_edge_case_cmd -> get_now()
"""


IMPORT_CASES = [
    Case(
        "Missing",
        ARC_IMPORT_MISSING,
        'module "time" is not imported',
    ),
    Case(
        "Unused",
        ARC_IMPORT_UNUSED,
        'imported module "time" is unused',
    ),
    Case(
        "UnknownModule",
        ARC_IMPORT_UNKNOWN,
        'unknown module "banana"',
    ),
    Case(
        "Duplicate",
        ARC_IMPORT_DUPLICATE,
        "duplicate import",
    ),
]

# ── Type mismatch sources (invalid, caught at configure time) ──

# f32 psi piped into f32 bar via flow connection between functions.
# Both units share the pressure dimension but differ by name; the analyzer
# must reject the assignment so that diagnostics surface unit identity, not
# just dimensional compatibility.
ARC_UNIT_MISMATCH_PSI_BAR = """
func producer() f32 psi {
    return 5psi
}
func consumer() {v f32 bar} {}
producer() -> consumer()
"""

# A float series index is rejected at configure time.
ARC_FLOAT_SERIES_INDEX = """
func pick() f64 {
    data series f64 := [1.0, 2.0, 3.0]
    i f32 := 1.0
    return data[i]
}
func consumer() {v f64} {}
pick() -> consumer()
"""

# A slice is a series, so a function that returns f64 cannot return it.
ARC_SLICE_RETURNED_AS_SCALAR = """
func slice_as_scalar() f64 {
    data := [1.0, 2.0, 3.0]
    sub := data[1:3]
    return sub
}
func sink() {v f64} {}
slice_as_scalar() -> sink()
"""

# A slice is a series, so a variable declared as f64 cannot hold it.
ARC_SLICE_DECLARED_AS_SCALAR = """
func slice_into_scalar() {
    data := [1.0, 2.0, 3.0]
    x f64 := data[1:3]
}
start_edge_case_cmd -> slice_into_scalar()
"""

# A slice is a series, so an f32 channel cannot take it.
ARC_SLICE_WRITTEN_TO_SCALAR_CHANNEL = """
func slice_into_channel() {
    data series f32 := [1.0, 2.0, 3.0]
    edge_diamond = data[1:3]
}
start_edge_case_cmd -> slice_into_channel()
"""


TYPE_MISMATCH_CASES = [
    Case(
        "UnitMismatchPsiBar",
        ARC_UNIT_MISMATCH_PSI_BAR,
        "does not match",
    ),
    Case(
        "SliceReturnedAsScalar",
        ARC_SLICE_RETURNED_AS_SCALAR,
        "cannot return series f64 from 'slice_as_scalar': expected f64",
    ),
    Case(
        "SliceDeclaredAsScalar",
        ARC_SLICE_DECLARED_AS_SCALAR,
        "cannot assign series f64 to 'x' (type f64)",
    ),
    Case(
        "SliceWrittenToScalarChannel",
        ARC_SLICE_WRITTEN_TO_SCALAR_CHANNEL,
        "cannot write series f32 to channel 'edge_diamond' (type f32)",
    ),
    Case(
        "FloatSeriesIndex",
        ARC_FLOAT_SERIES_INDEX,
        "series index must be an integer type, got f32",
    ),
]

# ── Transition into a routing table or select (invalid, caught at configure time) ──

# A routing key selects its entry, so `=>` before the table has nothing to gate.
ARC_TRANSITION_INTO_ROUTING_TABLE = """
ch1 > 0.0 -> select() => {
    true: 1.0 -> edge_diamond,
    false: 0.0 -> edge_diamond,
}
"""

# `=>` only passes true, so select's false entry could never run.
ARC_TRANSITION_INTO_SELECT = """
ch1 > 0.0 => select() -> {
    true: 1.0 -> edge_diamond,
    false: 0.0 -> edge_diamond,
}
"""


ROUTING_TABLE_CASES = [
    Case(
        "TransitionIntoTable",
        ARC_TRANSITION_INTO_ROUTING_TABLE,
        "cannot feed a routing table",
    ),
    Case(
        "TransitionIntoSelect",
        ARC_TRANSITION_INTO_SELECT,
        "cannot feed select",
    ),
]

# ── Guarded circular calls (valid, should configure successfully) ──
# Comprehensive guarded topology coverage is in the Go unit tests.

# Self-recursion guarded by if
ARC_GUARDED_SELF_REC = """
func self_rec() {
    if ch1 > 0 {
        ch1 = ch1 - 1.0
        self_rec()
    }
}
start_edge_case_cmd -> self_rec()
"""

# Same tangled web but route_beta wraps its call in if ch1 > 0, providing one exit path.
ARC_GUARDED_TANGLED_WEB = """
func init_seq() {
    ch1 = 1.0
    if ch1 > 50 {
        ch1 = ch1 + 20.0
    } else if ch1 > 20 {
        ch1 = ch1 + 5.0
    }
    proc_alpha()
}
func proc_alpha() {
    if ch1 > 80 {
        ch1 = ch1 - 30.0
    } else {
        ch1 = ch1 + 10.0
    }
    xform()
}
func xform() {
    ch1 = ch1 + 15.0
    if ch1 > 120 {
        ch1 = 100.0
    } else {
        ch1 = ch1 + 20.0
    }
    route_beta()
}
func route_beta() {
    if ch1 > 0 {
        if ch1 > 90 {
            ch1 = ch1 - 40.0
        } else if ch1 > 60 {
            ch1 = ch1 + 15.0
        }
        commit()
    }
}
func commit() {
    ch1 = 50.0
    init_seq()
}
start_edge_case_cmd -> init_seq()
"""

CHANNEL_VIRTUAL = [
    "edge_same_ch",
    "edge_diamond",
    "edge_multi_a",
    "edge_multi_b",
    "edge_chain",
    "edge_fwd",
    "edge_chan_basic",
    "edge_chan_chain",
    "edge_chan_arg_a",
    "edge_chan_arg_b",
    "edge_chan_mp_a",
    "edge_chan_mp_b",
    "edge_chan_fwd",
    "ch1",
]


CIRCULAR_CASES = [
    Case("SelfRec", ARC_SELF_REC, "self_rec -> self_rec"),
    Case(
        "AllBranches",
        ARC_ALL_BRANCHES,
        "circular function call",
        expect=["ping", "pong"],
    ),
    Case(
        "TangledWeb",
        ARC_TANGLED_WEB,
        "circular function call",
        expect=["init_seq"],
    ),
]


GUARDED_CASES = [
    Case("SelfRec", ARC_GUARDED_SELF_REC),
    Case("TangledWeb", ARC_GUARDED_TANGLED_WEB),
]


class EdgeCases(ArcCase):
    """Test channel propagation edge cases at runtime and circular dependency
    detection at configure time. Comprehensive circular/guarded topology
    coverage (mutual, chain, diamond, buried, overlap) is in the Go unit tests
    at arc/go/analyzer/analyzer_test.go. This file focuses on end-to-end
    verification through the Python client."""

    arc_source = ARC_CHANNEL_EDGE_CASES_SOURCE
    arc_name_prefix = "ArcEdgeCases"
    start_cmd_channel = "start_edge_case_cmd"
    subscribe_channels = [
        *CHANNEL_VIRTUAL,
        "edge_len_out",
        "edge_fmt_fn_out",
        "edge_fmt_flow_out",
    ]
    collect_notifications = True
    collect_task_status = True

    def setup(self) -> None:
        for ch in CHANNEL_VIRTUAL:
            create_virtual_channel(self.client, ch)
        create_virtual_channel(
            self.client, "start_read_only_monitor_cmd", data_type=sy.DataType.UINT8
        )
        create_virtual_channel(self.client, "edge_monitor_input")
        create_virtual_channel(
            self.client, "edge_len_input", data_type=sy.DataType.STRING
        )
        create_virtual_channel(self.client, "edge_len_out", data_type=sy.DataType.INT64)
        create_virtual_channel(
            self.client, "edge_fmt_ts_in", data_type=sy.DataType.TIMESTAMP
        )
        for ch in (
            "edge_fmt_in",
            "edge_fmt_tz_in",
            "edge_fmt_fn_out",
            "edge_fmt_flow_out",
            "edge_fmt_lit_fn_out",
            "edge_fmt_lit_flow_out",
        ):
            create_virtual_channel(self.client, ch, data_type=sy.DataType.STRING)
        self.client.statuses.set(
            sy.Status(
                key="press_monitor_status",
                name="Press Monitor",
                variant="disabled",
                message="Initialized",
            )
        )
        super().setup()

    def _verify_channel_edge_cases(self) -> None:
        self.log("=== Channel propagation edge cases ===")

        self.log("[SameChannel] last write wins -> 2.0")
        self.wait_for_near("edge_same_ch", 2.0, tolerance=0.01)

        self.log("[Diamond] diamond_leaf -> 42.0")
        self.wait_for_near("edge_diamond", 42.0, tolerance=0.01)

        self.log("[MultiCallee] write_a -> 10.0, write_b -> 20.0")
        self.wait_for_near("edge_multi_a", 10.0, tolerance=0.01)
        self.wait_for_near("edge_multi_b", 20.0, tolerance=0.01)

        self.log("[Chain] chain_leaf -> 99.0")
        self.wait_for_near("edge_chain", 99.0, tolerance=0.01)

        self.log("[FwdRef] fwd_callee -> 55.0")
        self.wait_for_near("edge_fwd", 55.0, tolerance=0.01)

        self.log("[ChanParam] chan_write -> 77.0")
        self.wait_for_near("edge_chan_basic", 77.0, tolerance=0.01)

        self.log("[ChanChain] chan_chain_leaf -> 88.0")
        self.wait_for_near("edge_chan_chain", 88.0, tolerance=0.01)

        self.log("[ChanDiffArgs] chan_set -> 33.0 for both")
        self.wait_for_near("edge_chan_arg_a", 33.0, tolerance=0.01)
        self.wait_for_near("edge_chan_arg_b", 33.0, tolerance=0.01)

        self.log("[ChanMultiParam] chan_multi_write -> 11.0, 22.0")
        self.wait_for_near("edge_chan_mp_a", 11.0, tolerance=0.01)
        self.wait_for_near("edge_chan_mp_b", 22.0, tolerance=0.01)

        self.log("[ChanFwdRef] chan_fwd_callee -> 66.0")
        self.wait_for_near("edge_chan_fwd", 66.0, tolerance=0.01)

    def _verify_circular_cases(self) -> None:
        self.log("=== Circular dependency detection ===")
        for case in CIRCULAR_CASES:
            self.log(f"[{case.label}] Testing circular dependency")
            self._assert_configure_error(
                case.source,
                name_prefix=f"Circ{case.label}",
                substr=case.wait_substr,
                expect=case.expect,
            )

    def _assert_guarded_configures(self, case: Case) -> None:
        self.log(f"[Guarded {case.label}] Testing guarded recursion")
        self.load_arc(case.source, name_prefix=f"Guard{case.label}")

    def _verify_guarded_cases(self) -> None:
        self.log("=== Guarded recursion (should deploy successfully) ===")
        for case in GUARDED_CASES:
            self._assert_guarded_configures(case)

    def _verify_exec_context_cases(self) -> None:
        self.log("=== ExecContext violation detection ===")
        for case in EXEC_CONTEXT_CASES:
            self.log(f"[{case.label}] Testing exec context violation")
            self._assert_configure_error(
                case.source,
                name_prefix=f"ExecCtx{case.label}",
                substr=case.wait_substr,
            )

    def _verify_wasm_in_flow(self) -> None:
        self.log("=== WASM-only callee as a flow node ===")
        self.load_arc(ARC_WASM_IN_FLOW, name_prefix="WasmInFlow")
        self.writer.write("edge_len_input", "hello")
        self.wait_for_eq("edge_len_out", 5)

    def _expect_format_warning(self, names: Sequence[str], zone: str) -> None:
        """Fail unless every named Arc warned about zone and kept running."""
        description = f'The time zone "{zone}" is invalid.'
        for name in names:
            key = self.task_key(name)
            if not self.wait_for_task_status(key, description):
                self.fail(f"[{name}] no warning surfaced for zone {zone!r}")
            if not self.task_is_running(key):
                self.fail(f"[{name}] stopped after the warning for zone {zone!r}")

    def _verify_time_format_invalid(self) -> None:
        self.log("=== time.format with invalid inputs ===")
        names = [
            self.load_arc(ARC_FORMAT_FN, name_prefix="FormatFn"),
            self.load_arc(ARC_FORMAT_FLOW, name_prefix="FormatFlow"),
        ]
        for case in FORMAT_INVALID_CASES:
            self.log(f"[{case.label}] {case.fmt!r} in {case.zone!r}")
            self.writer.write("edge_fmt_ts_in", FORMAT_STAMP)
            self.writer.write("edge_fmt_in", case.fmt)
            self.writer.write("edge_fmt_tz_in", case.zone)
            self._expect_format_warning(names, case.zone)
        for odd in FORMAT_ODD_CASES:
            self.log(f"[{odd.label}] {odd.fmt!r} in {odd.zone!r}")
            self.writer.write("edge_fmt_ts_in", FORMAT_STAMP)
            self.writer.write("edge_fmt_in", odd.fmt)
            self.writer.write("edge_fmt_tz_in", odd.zone)
            self.wait_for_eq("edge_fmt_fn_out", odd.expected)
            self.wait_for_eq("edge_fmt_flow_out", odd.expected)
        self.log("[LiteralZone] Mars/Olympus as a literal")
        literal = [
            self.load_arc(ARC_FORMAT_LITERAL_ZONE_FN, name_prefix="FormatLitFn"),
            self.load_arc(ARC_FORMAT_LITERAL_ZONE_FLOW, name_prefix="FormatLitFlow"),
        ]
        self.writer.write("edge_fmt_ts_in", FORMAT_STAMP)
        self._expect_format_warning(literal, "Mars/Olympus")

    def _verify_import_cases(self) -> None:
        self.log("=== Import diagnostic detection ===")
        for case in IMPORT_CASES:
            self.log(f"[{case.label}] Testing import diagnostic")
            self._assert_configure_error(
                case.source,
                name_prefix=f"Import{case.label}",
                substr=case.wait_substr,
            )

    def _verify_type_mismatch_cases(self) -> None:
        self.log("=== Type mismatch detection ===")
        for case in TYPE_MISMATCH_CASES:
            self.log(f"[{case.label}] Testing type mismatch")
            self._assert_configure_error(
                case.source,
                name_prefix=f"TypeMismatch{case.label}",
                substr=case.wait_substr,
            )

    def _verify_routing_table_cases(self) -> None:
        self.log("=== Transition into routing detection ===")
        for case in ROUTING_TABLE_CASES:
            self.log(f"[{case.label}] Testing transition into routing")
            self._assert_configure_error(
                case.source,
                name_prefix=f"RoutingTable{case.label}",
                substr=case.wait_substr,
            )

    def _verify_read_only_monitor(self) -> None:
        self.log("=== Read-only monitor (no write channels) ===")
        name = self.load_arc(
            ARC_READ_ONLY_MONITOR,
            name_prefix="ReadOnly",
            trigger="start_read_only_monitor_cmd",
        )

        self.writer.write("edge_monitor_input", 50.0)
        self.log("Waiting for pressure status notification...")
        assert self.wait_for_notification("Pressure high"), (
            "No pressure status notification found"
        )

        assert self.task_is_running(self.task_key(name)), (
            "Read-only Arc stopped unexpectedly"
        )
        self.log("Read-only Arc with no write channels running successfully")

    def verify_sequence_execution(self) -> None:
        self._verify_channel_edge_cases()
        self._verify_circular_cases()
        self._verify_guarded_cases()
        self._verify_exec_context_cases()
        self._verify_wasm_in_flow()
        self._verify_time_format_invalid()
        self._verify_type_mismatch_cases()
        self._verify_routing_table_cases()
        self._verify_import_cases()
        self._verify_read_only_monitor()

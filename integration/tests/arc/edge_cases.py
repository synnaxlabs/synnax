#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass, field

import synnax as sy
from examples.simulators import PressSimDAQ
from x import random_name

from tests.arc.arc_case import ArcConsoleCase

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
func chan_write(ch chan f32) {
    ch = 77.0
}
func test_chan_param_write() {
    chan_write(edge_chan_basic)
}

// Edge case 7: Multi-level chain through chan params.
func chan_chain_leaf(ch chan f32) {
    ch = 88.0
}
func chan_chain_mid(ch chan f32) {
    chan_chain_leaf(ch)
}
func test_chan_chain() {
    chan_chain_mid(edge_chan_chain)
}

// Edge case 8: Same function called with different chan args.
func chan_set(ch chan f32) {
    ch = 33.0
}
func test_chan_diff_args() {
    chan_set(edge_chan_arg_a)
    chan_set(edge_chan_arg_b)
}

// Edge case 9: Multiple chan params written in one function.
func chan_multi_write(a chan f32, b chan f32) {
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
func chan_fwd_callee(ch chan f32) {
    ch = 66.0
}

interval{period=100ms} -> test_same_channel_write{}
interval{period=100ms} -> test_diamond{}
interval{period=100ms} -> test_multi_callee{}
interval{period=100ms} -> test_chain{}
interval{period=100ms} -> test_fwd_ref{}
interval{period=100ms} -> test_chan_param_write{}
interval{period=100ms} -> test_chan_chain{}
interval{period=100ms} -> test_chan_diff_args{}
interval{period=100ms} -> test_chan_multi_param{}
interval{period=100ms} -> test_chan_fwd_ref{}
"""

# ── Circular dependency sources (invalid, caught at configure time) ──
# Comprehensive topology coverage is in the Go unit tests
# (arc/go/analyzer/analyzer_test.go). These integration tests verify that
# circular detection works end-to-end through the console UI.

# a -> a
ARC_SELF_REC = """
func self_rec() {
    ch1 = 1.0
    self_rec()
}
interval{period=100ms} -> self_rec{}
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
func pong() { ping() }
interval{period=100ms} -> ping{}
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
interval{period=100ms} -> init_seq{}
"""

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
interval{period=100ms} -> self_rec{}
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
interval{period=100ms} -> init_seq{}
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


@dataclass
class CircularCase:
    label: str
    source: str
    wait_substr: str
    expect: list[str] = field(default_factory=list)


CIRCULAR_CASES = [
    CircularCase("SelfRec", ARC_SELF_REC, "self_rec -> self_rec"),
    CircularCase(
        "AllBranches",
        ARC_ALL_BRANCHES,
        "circular function call",
        expect=["ping", "pong"],
    ),
    CircularCase(
        "TangledWeb",
        ARC_TANGLED_WEB,
        "circular function call",
        expect=["init_seq"],
    ),
]


@dataclass
class GuardedCase:
    label: str
    source: str


GUARDED_CASES = [
    GuardedCase("SelfRec", ARC_GUARDED_SELF_REC),
    GuardedCase("TangledWeb", ARC_GUARDED_TANGLED_WEB),
]


class EdgeCases(ArcConsoleCase):
    """Test channel propagation edge cases at runtime and circular dependency
    detection at configure time. Comprehensive circular/guarded topology
    coverage (mutual, chain, diamond, buried, overlap) is in the Go unit tests
    at arc/go/analyzer/analyzer_test.go. This file focuses on end-to-end
    verification through the console UI."""

    arc_source = ARC_CHANNEL_EDGE_CASES_SOURCE
    arc_name_prefix = "ArcEdgeCases"
    start_cmd_channel = "start_edge_case_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = CHANNEL_VIRTUAL
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self._extra_arcs: list[str] = []
        for ch in CHANNEL_VIRTUAL:
            self.client.channels.create(
                name=ch,
                data_type=sy.DataType.FLOAT32,
                virtual=True,
                retrieve_if_name_exists=True,
            )
        super().setup()

    def _verify_channel_edge_cases(self) -> None:
        self.log("=== Channel propagation edge cases ===")

        self.log("[SameChannel] last write wins -> 2.0")
        self.wait_for_near("edge_same_ch", 2.0, tolerance=0.01, is_virtual=True)

        self.log("[Diamond] diamond_leaf -> 42.0")
        self.wait_for_near("edge_diamond", 42.0, tolerance=0.01, is_virtual=True)

        self.log("[MultiCallee] write_a -> 10.0, write_b -> 20.0")
        self.wait_for_near("edge_multi_a", 10.0, tolerance=0.01, is_virtual=True)
        self.wait_for_near("edge_multi_b", 20.0, tolerance=0.01, is_virtual=True)

        self.log("[Chain] chain_leaf -> 99.0")
        self.wait_for_near("edge_chain", 99.0, tolerance=0.01, is_virtual=True)

        self.log("[FwdRef] fwd_callee -> 55.0")
        self.wait_for_near("edge_fwd", 55.0, tolerance=0.01, is_virtual=True)

        self.log("[ChanParam] chan_write -> 77.0")
        self.wait_for_near("edge_chan_basic", 77.0, tolerance=0.01, is_virtual=True)

        self.log("[ChanChain] chan_chain_leaf -> 88.0")
        self.wait_for_near("edge_chan_chain", 88.0, tolerance=0.01, is_virtual=True)

        self.log("[ChanDiffArgs] chan_set -> 33.0 for both")
        self.wait_for_near("edge_chan_arg_a", 33.0, tolerance=0.01, is_virtual=True)
        self.wait_for_near("edge_chan_arg_b", 33.0, tolerance=0.01, is_virtual=True)

        self.log("[ChanMultiParam] chan_multi_write -> 11.0, 22.0")
        self.wait_for_near("edge_chan_mp_a", 11.0, tolerance=0.01, is_virtual=True)
        self.wait_for_near("edge_chan_mp_b", 22.0, tolerance=0.01, is_virtual=True)

        self.log("[ChanFwdRef] chan_fwd_callee -> 66.0")
        self.wait_for_near("edge_chan_fwd", 66.0, tolerance=0.01, is_virtual=True)

    def _assert_circular_error(self, case: CircularCase) -> None:
        arc_name = f"Circ{case.label}_{random_name()}"
        self.log(f"[{case.label}] Testing {arc_name}")

        self.console.arc.create(arc_name, case.source, mode="Text")
        self._extra_arcs.append(arc_name)
        assert self.rack is not None
        self.console.arc.select_rack(self.rack.name)

        self.console.arc.configure_no_wait()
        status = self.console.arc.wait_for_status(case.wait_substr)

        for name in case.expect:
            assert (
                name in status
            ), f"[{case.label}] Expected '{name}' in error, got: {status}"

        notifications = self.console.notifications.check(timeout=5)
        error_notifications = [n for n in notifications if n.get("type") == "error"]
        assert (
            len(error_notifications) > 0
        ), f"[{case.label}] Expected error notification, got: {notifications}"
        self.console.notifications.close_all()

    def _verify_circular_cases(self) -> None:
        self.log("=== Circular dependency detection ===")
        for case in CIRCULAR_CASES:
            self._assert_circular_error(case)

    def _assert_guarded_configures(self, case: GuardedCase) -> None:
        arc_name = f"Guard{case.label}_{random_name()}"
        self.log(f"[Guarded {case.label}] Testing {arc_name}")

        self.console.arc.create(arc_name, case.source, mode="Text")
        self._extra_arcs.append(arc_name)
        assert self.rack is not None
        self.console.arc.select_rack(self.rack.name)

        self.console.arc.configure()

    def _verify_guarded_cases(self) -> None:
        self.log("=== Guarded recursion (should configure successfully) ===")
        for case in GUARDED_CASES:
            self._assert_guarded_configures(case)

    def verify_sequence_execution(self) -> None:
        self._verify_channel_edge_cases()
        self._verify_circular_cases()
        self._verify_guarded_cases()

    def teardown(self) -> None:
        if self._extra_arcs:
            try:
                arcs = self.client.arcs.retrieve(names=self._extra_arcs)
                if arcs:
                    self.client.arcs.delete([a.key for a in arcs])
            except Exception as e:
                self.log(f"Cleanup failed for extra arcs: {e}")
        super().teardown()

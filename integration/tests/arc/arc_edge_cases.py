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

from framework.utils import get_random_name
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

interval{period=100ms} -> test_same_channel_write{}
interval{period=100ms} -> test_diamond{}
interval{period=100ms} -> test_multi_callee{}
interval{period=100ms} -> test_chain{}
interval{period=100ms} -> test_fwd_ref{}
"""

# ── Circular dependency sources (invalid, caught at configure time) ──

# a -> a
ARC_SELF_REC = """
func self_rec() {
    ch1 = 1.0
    self_rec()
}
interval{period=100ms} -> self_rec{}
"""

# a -> b -> a
ARC_MUTUAL = """
func ping() {
    ch1 = 1.0
    pong()
}
func pong() {
    ping()
}
interval{period=100ms} -> ping{}
"""

# a -> b -> c -> d -> a
ARC_CHAIN = """
func a() {
    ch1 = 1.0
    b()
}
func b() { c() }
func c() { d() }
func d() { a() }
interval{period=100ms} -> a{}
"""

# a -> b, a -> c, b -> a, c -> a (diamond with back edge)
ARC_DIAMOND = """
func root() {
    ch1 = 1.0
    left()
    right()
}
func left() { root() }
func right() { root() }
interval{period=100ms} -> root{}
"""

# Cycle buried in a larger acyclic tree
ARC_BURIED = """
func leaf1() { ch1 = 1.0 }
func leaf2() { ch2 = 2.0 }
func safe_mid() {
    leaf1()
    leaf2()
}
func cycle_a() {
    ch3 = 3.0
    cycle_b()
}
func cycle_b() { cycle_a() }
func top() {
    safe_mid()
    cycle_a()
}
interval{period=100ms} -> top{}
"""

# a self-recurses AND participates in a mutual cycle
ARC_OVERLAP = """
func a() {
    ch1 = 1.0
    a()
    b()
}
func b() { a() }
interval{period=100ms} -> a{}
"""

CHANNEL_VIRTUAL = [
    "edge_same_ch",
    "edge_diamond",
    "edge_multi_a",
    "edge_multi_b",
    "edge_chain",
    "edge_fwd",
    "ch1",
    "ch2",
    "ch3",
]


@dataclass
class CircularCase:
    label: str
    source: str
    wait_substr: str
    expect: list[str] = field(default_factory=list)
    reject: list[str] = field(default_factory=list)


CIRCULAR_CASES = [
    CircularCase("SelfRec", ARC_SELF_REC, "self_rec -> self_rec"),
    CircularCase(
        "Mutual", ARC_MUTUAL, "circular function call", expect=["ping", "pong"]
    ),
    CircularCase(
        "Chain", ARC_CHAIN, "circular function call", expect=["a", "b", "c", "d"]
    ),
    CircularCase("Diamond", ARC_DIAMOND, "circular function call", expect=["root"]),
    CircularCase(
        "Buried",
        ARC_BURIED,
        "circular function call",
        expect=["cycle_a", "cycle_b"],
        reject=["top", "safe_mid", "leaf1", "leaf2"],
    ),
    CircularCase("Overlap", ARC_OVERLAP, "a -> a"),
]


class ArcEdgeCases(ArcConsoleCase):
    """Test channel propagation edge cases at runtime and circular dependency
    detection at configure time."""

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

    def _assert_circular_error(self, case: CircularCase) -> None:
        arc_name = f"Circ{case.label}_{get_random_name()}"
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

        for name in case.reject:
            assert (
                name not in status
            ), f"[{case.label}] '{name}' should NOT be in error, got: {status}"

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

    def verify_sequence_execution(self) -> None:
        self._verify_channel_edge_cases()
        self._verify_circular_cases()

    def teardown(self) -> None:
        for name in reversed(self._extra_arcs):
            try:
                if self.console.arc.is_running():
                    self.console.arc.stop()
                self.console.arc.open(name)
                self.console.arc.delete(name)
            except Exception as e:
                self.log(f"Cleanup failed for {name}: {e}")
        super().teardown()

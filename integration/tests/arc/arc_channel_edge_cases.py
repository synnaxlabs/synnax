#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.simulators import PressSimDAQ

from tests.arc.arc_case import ArcConsoleCase

# Edge cases for channel propagation through function calls.
#
# Note: Recursive edge cases (mutual recursion, self-recursion, circular chains)
# are tested at the analyzer level only (analyzer_test.go) since they would cause
# infinite loops at runtime. The cases below test non-recursive patterns that
# verify runtime channel write behavior.
ARC_CHANNEL_EDGE_CASES_SOURCE = """
// Edge case 1: Multiple callees write the same channel.
// set_to_one() runs first, then set_to_two() runs second.
// Verifies which value "wins" (first vs last write semantics).
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

// Edge case 2: Diamond dependency.
// a calls b and c, both b and c call d which writes to a channel.
// Verifies the value propagates correctly without duplication issues.
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
// caller calls two helpers, each writing to a distinct channel.
// Verifies both channels receive their respective values.
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
// Verifies transitive propagation through multiple call levels.
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
// test_fwd_ref calls fwd_callee which is declared after it.
// Verifies forward references work correctly at runtime.
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

VIRTUAL_CHANNELS = [
    "edge_same_ch",
    "edge_diamond",
    "edge_multi_a",
    "edge_multi_b",
    "edge_chain",
    "edge_fwd",
]


class ArcChannelEdgeCases(ArcConsoleCase):
    """Test runtime behavior of channel propagation edge cases.

    Covers edge cases for the channel accumulation fixpoint loop
    (propagateCallChannels in analyzer.go):
    1. Multiple callees writing the same channel (last write wins)
    2. Diamond dependency (value through two paths)
    3. Multiple callees writing different channels
    4. Multi-level transitive chain
    5. Forward reference (caller before callee)
    """

    arc_source = ARC_CHANNEL_EDGE_CASES_SOURCE
    arc_name_prefix = "ArcChannelEdgeCases"
    start_cmd_channel = "start_edge_case_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = VIRTUAL_CHANNELS
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        for ch in VIRTUAL_CHANNELS:
            self.client.channels.create(
                name=ch,
                data_type=sy.DataType.FLOAT32,
                virtual=True,
                retrieve_if_name_exists=True,
            )
        super().setup()

    def verify_sequence_execution(self) -> None:
        # --- Edge case 1: Multiple callees write the same channel ---
        # set_to_one() writes 1.0, then set_to_two() writes 2.0.
        # In sequential execution, the last write wins -> expect 2.0.
        self.log("Edge case 1: multiple callees write same channel")
        self.wait_for_near("edge_same_ch", 2.0, tolerance=0.01, is_virtual=True)
        actual = self.read_tlm("edge_same_ch")
        self.log(f"  edge_same_ch = {actual} (expected 2.0, last write wins)")

        # --- Edge case 2: Diamond dependency ---
        # diamond_leaf writes 42.0, called through two paths (left and right).
        self.log("Edge case 2: diamond dependency")
        self.wait_for_near("edge_diamond", 42.0, tolerance=0.01, is_virtual=True)

        # --- Edge case 3: Multiple callees write different channels ---
        # write_a writes 10.0 to edge_multi_a, write_b writes 20.0 to edge_multi_b.
        self.log("Edge case 3: multiple callees, different channels")
        self.wait_for_near("edge_multi_a", 10.0, tolerance=0.01, is_virtual=True)
        self.wait_for_near("edge_multi_b", 20.0, tolerance=0.01, is_virtual=True)

        # --- Edge case 4: Multi-level chain ---
        # test_chain -> chain_mid -> chain_leaf writes 99.0.
        self.log("Edge case 4: multi-level chain")
        self.wait_for_near("edge_chain", 99.0, tolerance=0.01, is_virtual=True)

        # --- Edge case 5: Forward reference ---
        # test_fwd_ref (declared first) calls fwd_callee (declared second)
        # which writes 55.0.
        self.log("Edge case 5: forward reference")
        self.wait_for_near("edge_fwd", 55.0, tolerance=0.01, is_virtual=True)

        self.log("All channel propagation edge cases passed")

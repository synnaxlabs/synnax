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

ARC_STAGELESS_WORKFLOW_SOURCE = """
authority 200

sequence main {
    1 -> sw_a
    1 -> sw_b
    1 -> sw_c
    wait{500ms}
    sw_pressure > 50
    0 -> sw_a
    0 -> sw_b
    0 -> sw_c
}

sw_start_cmd => main
"""


class StagelessWorkflow(ArcConsoleCase):
    """A bare sequence exercising the full stageless feature set.

    The single sequence covers every stageless construct in one workflow:
    write cascading (three consecutive writes that fire on a single
    tick), a ``wait{}`` gate that blocks progression until a duration
    elapses, a bare expression gate that blocks until a channel crosses
    a threshold, and a final cascade of three writes after the gates
    open.

    Phases:
      1. Trigger the sequence. Verify all three "high" channels reach 1.
         The runtime must cascade three immediately-truthy writes through
         on the same scheduler cycle.
      2. After the wait elapses, the sequence holds at the bare
         ``sw_pressure > 50`` gate. The "high" channels stay at 1 because
         no progression past the gate has occurred.
      3. Drive ``sw_pressure`` above 50. The gate becomes truthy, the
         sequence advances, and the final three writes cascade to flip
         all channels back to 0.
    """

    arc_source = ARC_STAGELESS_WORKFLOW_SOURCE
    arc_name_prefix = "ArcStagelessWorkflow"
    start_cmd_channel = "sw_start_cmd"
    subscribe_channels = [
        "sw_a",
        "sw_b",
        "sw_c",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "sw_a", sy.DataType.UINT8)
        create_virtual_channel(self.client, "sw_b", sy.DataType.UINT8)
        create_virtual_channel(self.client, "sw_c", sy.DataType.UINT8)
        create_virtual_channel(self.client, "sw_pressure", sy.DataType.FLOAT32)
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for first cascade (sw_a = 1)...")
        self.wait_for_eq("sw_a", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True)
        self.log(
            "sw_a=1 observed; verifying sw_b and sw_c cascaded on same cycle "
            "(all three writes must be visible within 100ms of sw_a)..."
        )
        self.wait_for_eq(
            "sw_b", 1, timeout=100 * sy.TimeSpan.MILLISECOND, is_virtual=True
        )
        self.wait_for_eq(
            "sw_c", 1, timeout=100 * sy.TimeSpan.MILLISECOND, is_virtual=True
        )
        self.log("First cascade observed on a single tick")

        self.log(
            "Driving sw_pressure=10 (below gate threshold) and waiting for "
            "wait{500ms} to elapse without advancing"
        )
        self.writer.write("sw_pressure", 10.0)
        sy.sleep(1.5)
        a_value = self.read_tlm("sw_a")
        if a_value is None:
            self.fail(
                "sw_a has no buffered value; cannot verify gate held. "
                "Streamer did not deliver any reading after trigger."
            )
            return
        if a_value != 1:
            self.fail(
                f"sw_a={a_value} after wait elapsed but expression gate "
                "should still be blocking; sequence should not have advanced"
            )
            return
        self.log("Sequence correctly held at expression gate")

        self.log("Driving sw_pressure=75 (above gate threshold)")
        self.writer.write("sw_pressure", 75.0)
        self.log("Waiting for second cascade (sw_a = 0)...")
        self.wait_for_eq("sw_a", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True)
        self.log("sw_a=0 observed; verifying sw_b and sw_c cascaded on same cycle...")
        self.wait_for_eq(
            "sw_b", 0, timeout=100 * sy.TimeSpan.MILLISECOND, is_virtual=True
        )
        self.wait_for_eq(
            "sw_c", 0, timeout=100 * sy.TimeSpan.MILLISECOND, is_virtual=True
        )
        self.log("Second cascade fired on a single tick after gate opened")

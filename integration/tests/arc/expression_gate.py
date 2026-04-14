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

ARC_EXPRESSION_GATE_SOURCE = """
authority 200

sequence main {
    1 -> gate_press_cmd
    gate_pressure > 50
    0 -> gate_press_cmd
}

gate_start_cmd => main
"""


class ExpressionGate(ArcConsoleCase):
    """A bare expression appearing as a sequence item acts as a gate.

    Validates that a bare comparison expression in a sequence body blocks
    progression until it evaluates to truthy. The first write fires
    immediately on entry, then the sequence holds at the comparison until
    the channel value crosses the threshold, after which the final write
    fires.

    Sequence:
      1. Write ``gate_press_cmd = 1`` (immediate)
      2. Block on ``gate_pressure > 50`` until pressure crosses
      3. Write ``gate_press_cmd = 0``
    """

    arc_source = ARC_EXPRESSION_GATE_SOURCE
    arc_name_prefix = "ArcExpressionGate"
    start_cmd_channel = "gate_start_cmd"
    subscribe_channels = [
        "gate_press_cmd",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "gate_press_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "gate_pressure", sy.DataType.FLOAT32)
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for gate_press_cmd=1 (first write)...")
        self.wait_for_eq(
            "gate_press_cmd", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )

        self.log("Driving pressure=10 (below gate threshold)")
        self.writer.write("gate_pressure", 10.0)
        # Hold here briefly to give the runtime a chance to (incorrectly)
        # advance past the gate. If it does, gate_press_cmd would flip to 0.
        sy.sleep(1.0)

        self.log("Driving pressure=75 (above gate threshold)")
        self.writer.write("gate_pressure", 75.0)
        self.log("Waiting for gate_press_cmd=0 (final write after gate)...")
        self.wait_for_eq(
            "gate_press_cmd", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.log("Sequence advanced past gate after threshold crossed")

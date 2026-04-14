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

ARC_BACKWARD_JUMP_SOURCE = """
authority 200

sequence main {
    stage pressurize {
        1 -> bj_press_cmd,
        bj_pressure > 50 => hold
    }
    stage hold {
        0 -> bj_press_cmd,
        bj_pressure < 30 => pressurize
    }
}

bj_start_cmd => main
"""


class BackwardJump(ArcConsoleCase):
    """A sequence transitions forward to a named stage, then a backward
    ``=> name`` jump returns to an earlier stage which resets and runs
    again.

    The test cycles through pressurize -> hold -> pressurize:
      1. Enter pressurize: writes ``bj_press_cmd = 1``
      2. Drive ``bj_pressure`` above 50: transitions to hold
      3. hold writes ``bj_press_cmd = 0``
      4. Drive ``bj_pressure`` below 30: backward jump to pressurize
      5. pressurize re-runs and writes ``bj_press_cmd = 1`` again
    """

    arc_source = ARC_BACKWARD_JUMP_SOURCE
    arc_name_prefix = "ArcBackwardJump"
    start_cmd_channel = "bj_start_cmd"
    subscribe_channels = [
        "bj_press_cmd",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "bj_press_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "bj_pressure", sy.DataType.FLOAT32)
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for bj_press_cmd=1 (pressurize entered)...")
        self.wait_for_eq(
            "bj_press_cmd", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )

        self.log("Driving bj_pressure=75 (above pressurize -> hold threshold)")
        self.writer.write("bj_pressure", 75.0)
        self.log("Waiting for bj_press_cmd=0 (hold entered)...")
        self.wait_for_eq(
            "bj_press_cmd", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )

        self.log("Driving bj_pressure=10 (below hold -> pressurize threshold)")
        self.writer.write("bj_pressure", 10.0)
        self.log(
            "Waiting for bj_press_cmd=1 (backward jump re-entered pressurize)..."
        )
        self.wait_for_eq(
            "bj_press_cmd", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.log("Backward jump re-activated pressurize and re-fired its write")

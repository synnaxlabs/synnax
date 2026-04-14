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

ARC_INLINE_STAGE_SOURCE = """
authority 200

sequence main {
    1 -> inline_stage_ox_cmd
    stage {
        inline_stage_pressure < 15 => next,
    }
    0 -> inline_stage_ox_cmd
}

inline_stage_start_cmd => main
"""


class InlineStageInSequence(ArcConsoleCase):
    """A sequence body containing an anonymous inline ``stage {}`` block.

    Validates that a sequence can mix sequential writes with a reactive
    stage block, and that the stage's ``=> next`` transition resumes the
    parent sequence at the next step.

    Sequence:
      1. Write ``inline_stage_ox_cmd = 1`` (immediate)
      2. Enter inline stage; block until pressure drops below 15
      3. Write ``inline_stage_ox_cmd = 0``
    """

    arc_source = ARC_INLINE_STAGE_SOURCE
    arc_name_prefix = "ArcInlineStageInSequence"
    start_cmd_channel = "inline_stage_start_cmd"
    subscribe_channels = [
        "inline_stage_ox_cmd",
    ]

    def setup(self) -> None:
        create_virtual_channel(
            self.client, "inline_stage_ox_cmd", sy.DataType.UINT8
        )
        create_virtual_channel(
            self.client, "inline_stage_pressure", sy.DataType.FLOAT32
        )
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for ox_cmd=1 (first sequential write)...")
        self.wait_for_eq(
            "inline_stage_ox_cmd", 1, timeout=5 * sy.TimeSpan.SECOND,
            is_virtual=True,
        )

        self.log("Driving pressure=100 (above stage exit threshold)")
        self.writer.write("inline_stage_pressure", 100.0)
        sy.sleep(1.0)

        self.log("Driving pressure=10 (below stage exit threshold)")
        self.writer.write("inline_stage_pressure", 10.0)
        self.log("Waiting for ox_cmd=0 (sequence resumed past inline stage)...")
        self.wait_for_eq(
            "inline_stage_ox_cmd", 0, timeout=5 * sy.TimeSpan.SECOND,
            is_virtual=True,
        )
        self.log("Sequence resumed and executed final write")

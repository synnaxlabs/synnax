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

ARC_INLINE_SEQUENCE_IN_STAGE_SOURCE = """
authority 200

sequence main {
    stage fire {
        sequence {
            1 -> iss_ox_cmd
        },
        iss_pressure < 15 => exit,
    }
    stage exit {
        0 -> iss_ox_cmd,
        1 -> iss_vent_cmd,
        iss_pressure > 100 => fire,
    }
}

iss_start_cmd => main
"""


class InlineSequenceInStage(ArcConsoleCase):
    """A stage body containing an inline anonymous ``sequence {}`` that runs
    alongside the stage's reactive flows, and resets when the parent stage
    is re-entered.

    Phase 1 - inline sub-sequence runs alongside reactive exit:
      The fire stage activates an inline sub-sequence that writes
      ``iss_ox_cmd = 1`` while a parallel reactive transition watches
      ``iss_pressure < 15``. When pressure drops, the stage transitions to
      exit which writes ``iss_ox_cmd = 0`` and ``iss_vent_cmd = 1``.

    Phase 2 - sub-sequence resets when fire is re-entered:
      Drive ``iss_pressure`` above 100 to fire the exit -> fire backward
      transition. The fire stage activates again, which must re-trigger
      its inline sub-sequence from step 0. We observe by ``iss_ox_cmd``
      flipping back to 1.
    """

    arc_source = ARC_INLINE_SEQUENCE_IN_STAGE_SOURCE
    arc_name_prefix = "ArcInlineSequenceInStage"
    start_cmd_channel = "iss_start_cmd"
    subscribe_channels = [
        "iss_ox_cmd",
        "iss_vent_cmd",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "iss_ox_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "iss_vent_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "iss_pressure", sy.DataType.FLOAT32)
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for iss_ox_cmd=1 (inline sub-sequence first write)...")
        self.wait_for_eq(
            "iss_ox_cmd", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )

        self.log("Driving iss_pressure=100 (above stage exit threshold)")
        self.writer.write("iss_pressure", 100.0)
        sy.sleep(1.0)

        self.log("Driving iss_pressure=10 (below stage exit threshold)")
        self.writer.write("iss_pressure", 10.0)
        self.log("Waiting for iss_ox_cmd=0 and iss_vent_cmd=1 (exit stage)...")
        self.wait_for_eq(
            "iss_ox_cmd", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.wait_for_eq(
            "iss_vent_cmd", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.log("Sequence transitioned to exit stage")

        self.log(
            "Driving iss_pressure=150 (above exit -> fire backward jump threshold)"
        )
        self.writer.write("iss_pressure", 150.0)
        self.log(
            "Waiting for iss_ox_cmd=1 (fire re-entered, inline sub-sequence "
            "reset and re-fired its first write)..."
        )
        self.wait_for_eq(
            "iss_ox_cmd", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.log("Sub-sequence reset on parent stage re-entry")

        self.log(
            "Verifying exit's iss_vent_cmd=1 write does not re-apply after "
            "fire re-entry (exit stage must fully deactivate)..."
        )
        sy.sleep(1.0)
        self.writer.write("iss_vent_cmd", 0)
        sy.sleep(1.0)
        vent_value = self.read_tlm("iss_vent_cmd")
        if vent_value is None:
            self.fail(
                "iss_vent_cmd has no buffered value; cannot verify exit "
                "stage deactivated."
            )
            return
        if vent_value != 0:
            self.fail(
                f"iss_vent_cmd={vent_value} after manual reset to 0 while "
                "in fire stage. The exit stage's write is still applying, "
                "which means the stage did not fully deactivate on the "
                "backward transition."
            )
            return
        self.log("Exit stage's writes correctly stopped applying after re-entry")

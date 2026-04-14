#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_INLINE_STAGE_SOURCE = """
authority 200

sequence main {
    1 -> inline_stage_ox_cmd
    stage {
        inline_stage_pressure < 15 => next,
        wait{10s} => next,
    }
    0 -> inline_stage_ox_cmd
    1 -> inline_stage_vent_cmd
    stage {
        inline_stage_pressure > 9000 => next,
        wait{2s} => next,
    }
    0 -> inline_stage_vent_cmd
}

inline_stage_start_cmd => main
"""


class InlineStageInSequence(ArcConsoleCase):
    """A sequence body containing two anonymous inline ``stage {}`` blocks,
    each with a condition exit and a ``wait{}`` backstop exit. The two
    stages exercise the complementary winners of the multi-exit race.

    Phase A - condition wins:
      The first inline stage has ``pressure < 15 => next`` and a
      ``wait{10s}`` backstop. The test holds pressure high, asserts the
      sequence blocks (``ox_cmd`` stays 1), then drives pressure low and
      asserts the sequence resumes well before the 10s backstop. This
      proves the condition exit fires promptly and, by timing, that the
      backstop did not win.

    Phase B - wait wins:
      The second inline stage has ``pressure > 9000 => next`` (an
      unreachable threshold given how the test drives the channel) and a
      ``wait{2s}`` backstop. The test asserts the sequence blocks for
      ~1s with ``vent_cmd`` at 1, then asserts the sequence resumes
      around t=2s when the wait fires. This proves the wait exit fires
      on deadline even when the condition exit never triggers.
    """

    arc_source = ARC_INLINE_STAGE_SOURCE
    arc_name_prefix = "ArcInlineStageInSequence"
    start_cmd_channel = "inline_stage_start_cmd"
    subscribe_channels = [
        "inline_stage_ox_cmd",
        "inline_stage_vent_cmd",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "inline_stage_ox_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "inline_stage_vent_cmd", sy.DataType.UINT8)
        create_virtual_channel(
            self.client, "inline_stage_pressure", sy.DataType.FLOAT32
        )
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        # Phase A: condition wins
        self.log("Phase A: waiting for ox_cmd=1 (inline stage A entered)...")
        self.wait_for_eq(
            "inline_stage_ox_cmd",
            1,
            timeout=5 * sy.TimeSpan.SECOND,
            is_virtual=True,
        )

        self.log("Driving pressure=100 and asserting stage A blocks")
        self.writer.write("inline_stage_pressure", 100.0)
        sy.sleep(0.5)
        ox_while_blocked = self.read_tlm("inline_stage_ox_cmd")
        if ox_while_blocked != 1:
            self.fail(
                f"ox_cmd={ox_while_blocked} while pressure=100; inline "
                "stage A should still be blocking the sequence"
            )
            return

        self.log("Driving pressure=10; condition exit should fire fast")
        t_drive = time.monotonic()
        self.writer.write("inline_stage_pressure", 10.0)
        self.wait_for_eq(
            "inline_stage_ox_cmd",
            0,
            timeout=2 * sy.TimeSpan.SECOND,
            is_virtual=True,
        )
        elapsed_a = time.monotonic() - t_drive
        self.log(f"Phase A resumed in {elapsed_a:.2f}s (condition won)")

        # Phase B: wait wins
        self.log("Phase B: waiting for vent_cmd=1 (inline stage B entered)...")
        self.wait_for_eq(
            "inline_stage_vent_cmd",
            1,
            timeout=5 * sy.TimeSpan.SECOND,
            is_virtual=True,
        )
        t_entry = time.monotonic()

        self.log(
            "Asserting stage B blocks for 1s (condition is unreachable; "
            "only the 2s wait backstop should ever fire)"
        )
        sy.sleep(1.0)
        vent_while_blocked = self.read_tlm("inline_stage_vent_cmd")
        if vent_while_blocked != 1:
            self.fail(
                f"vent_cmd={vent_while_blocked} 1s into stage B; the wait"
                "{2s} backstop should not have fired yet"
            )
            return

        self.log("Waiting for wait{2s} backstop to fire (vent_cmd=0)...")
        self.wait_for_eq(
            "inline_stage_vent_cmd",
            0,
            timeout=3 * sy.TimeSpan.SECOND,
            is_virtual=True,
        )
        elapsed_b = time.monotonic() - t_entry
        self.log(f"Phase B resumed in {elapsed_b:.2f}s (wait backstop fired)")
        if elapsed_b < 1.5:
            self.fail(
                f"Phase B resumed in {elapsed_b:.2f}s but wait{{2s}} should "
                "not fire before ~2s; the condition exit may have fired "
                "spuriously"
            )
            return

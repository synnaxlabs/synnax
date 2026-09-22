#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Arc silent channel reads

A body read of a channel with no value yet must not evaluate as zero. The runtime
skips the pass, warns once, and retries when a value arrives:

- A polled transition holds while either channel is silent, even when the other
  operand would short-circuit the comparison.
- A one-shot wait transition, already fired while silent, evaluates as soon as the
  last silent channel delivers.
- A plain transition waits for every channel to deliver once.
- A skipped pass runs no side effects: a stateful counter sequenced before the
  silent read counts only the pass that evaluated.
"""

import synnax as sy
from framework.utils import create_virtual_channels
from tests.arc.arc import ArcCase

ARC_SOURCE = """
import time

func count_ticks{} (tick u8) i64 {
    n i64 $= 0
    n = n + 1
    return n
}

func cold_count{} (trigger u8) i64 {
    n i64 $= 0
    n = n + 1
    if temp_c < 4.0 {
        return n
    }
    return 0
}

// ──── polled transition ────
sequence polled {
    stage wait_cold {
        time.interval{100ms} -> count_ticks{} -> polled_ticks
        time.interval{100ms} -> temp_a < 4.0 and temp_b < 5.0 => hold
    }
    stage hold {
        1 -> polled_reached
    }
}

// ──── one-shot wait ────
sequence one_shot {
    stage wait_cold {
        time.wait{200ms} -> temp_d < 4.0 and temp_e < 5.0 => hold
    }
    stage hold {
        1 -> wait_reached
    }
}

// ──── plain transition ────
sequence plain {
    stage wait_cold {
        temp_d < 4.0 and temp_e < 5.0 => hold
    }
    stage hold {
        1 -> plain_reached
    }
}

// ──── side effects ────
sequence effects {
    stage wait_cold {
        time.wait{200ms} -> cold_count{} -> cold_counted
    }
}

silent_start => polled
silent_start => one_shot
silent_start => plain
silent_start => effects
"""

TEMPS = ["temp_a", "temp_b", "temp_c", "temp_d", "temp_e"]
FLAGS = ["polled_reached", "wait_reached", "plain_reached"]
COUNTS = ["polled_ticks", "cold_counted"]
OUTPUTS = FLAGS + COUNTS


class SilentReads(ArcCase):
    """Reads of a channel with no value yet skip the pass instead of reading zero."""

    arc_source = ARC_SOURCE
    arc_name_prefix = "ArcSilentReads"
    start_cmd_channel = "silent_start"
    subscribe_channels = OUTPUTS
    collect_task_status = True

    def setup(self) -> None:
        create_virtual_channels(
            self.client,
            [(name, sy.DataType.FLOAT32) for name in TEMPS]
            + [(name, sy.DataType.UINT8) for name in FLAGS]
            + [(name, sy.DataType.INT64) for name in COUNTS],
        )
        super().setup()

    def verify_sequence_execution(self) -> None:
        self._verify_polled_holds_while_silent()
        self._verify_warm_operand_does_not_pass()
        self._verify_half_cold_holds()
        self._verify_plain_waits_for_every_channel()
        self._verify_transitions_fire_once_values_arrive()
        self._verify_skipped_passes_left_no_trace()

    def _wait_ticks(self, count: int) -> None:
        """Waits for count more polls of the silent transition."""
        current = self.read_tlm("polled_ticks", 0)
        self.wait_for_ge("polled_ticks", int(current) + count)

    def _assert_unreached(self, flag: str, when: str) -> None:
        """The streamer seeds a subscribed channel with 0 until it is written."""
        value = self.read_tlm(flag, 0)
        if value != 0:
            self.fail(f"{flag} was written {when}: {value!r}")

    def _verify_polled_holds_while_silent(self) -> None:
        self.log("Polling with both channels silent...")
        self._wait_ticks(3)
        self._assert_unreached("polled_reached", "while both channels were silent")
        if not self.wait_for_task_status(
            self.task_key(self.arc_name), "has no value yet"
        ):
            self.fail("task never warned about the silent channel")

    def _verify_warm_operand_does_not_pass(self) -> None:
        self.log("Warm temp_a, silent temp_b...")
        self.writer.write("temp_a", 12.0)
        self._wait_ticks(3)
        self._assert_unreached("polled_reached", "with temp_b silent")

    def _verify_half_cold_holds(self) -> None:
        self.log("Cold temp_b, warm temp_a...")
        self.writer.write("temp_b", 3.0)
        self._wait_ticks(3)
        self._assert_unreached("polled_reached", "while temp_a was warm")

    def _verify_plain_waits_for_every_channel(self) -> None:
        self.log("Cold temp_d, silent temp_e...")
        self.writer.write("temp_d", 3.0)
        self._wait_ticks(3)
        self._assert_unreached("plain_reached", "with temp_e silent")
        self._assert_unreached("wait_reached", "with temp_e silent")

    def _verify_transitions_fire_once_values_arrive(self) -> None:
        self.log("Cold temp_a and temp_e...")
        self.writer.write("temp_a", 3.0)
        self.wait_for_eq("polled_reached", 1)
        self.writer.write("temp_e", 3.0)
        self.wait_for_eq("plain_reached", 1)
        self.wait_for_eq("wait_reached", 1)

    def _verify_skipped_passes_left_no_trace(self) -> None:
        self.log("Cold temp_c after the wait already fired...")
        self._assert_unreached("cold_counted", "while temp_c was silent")
        self.writer.write("temp_c", 3.0)
        self.wait_for_eq("cold_counted", 1)

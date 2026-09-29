#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
from examples.simulators import can_frames

import synnax as sy
from synnax import can, library
from tests.driver.bus_task import (
    Bound,
    Collector,
    assert_equal,
    assert_increasing,
    assert_indexed,
)
from tests.driver.can_task import CANCase

SINE_SCALE = 2**-14
TEMPERATURE_SCALE = 0.5
TEMPERATURE_OFFSET = -10.0
# Matches the slow frame's extended identifier on every source address, as J1939
# messages match by parameter group number.
SLOW_MASK = 0x1FFFFF00


class CANRead(CANCase):
    """Decodes a standard and an extended CAN frame into channels with an index each.
    The fast frame covers signed, big-endian, scaled, and unaligned fields, and the
    slow frame matches through an identifier mask and carries a float. The simulator
    also sends a frame with an unknown identifier, which the task must ignore."""

    prefix = "can_read"
    fast: Bound
    slow: Bound

    def create_entries(self) -> list[library.Entry]:
        return [
            library.MessageEntry(
                name="fast",
                payload=library.BinaryPayload(
                    identifier=library.CanIdentifier(id=can_frames.FAST_ID),
                    length=8,
                    fields=[
                        library.BinaryField(name="count", start_bit=0, bit_length=16),
                        library.BinaryField(
                            name="sine",
                            start_bit=16,
                            bit_length=16,
                            signed=True,
                            scale=SINE_SCALE,
                        ),
                        library.BinaryField(
                            name="temperature",
                            start_bit=32,
                            bit_length=12,
                            signed=True,
                            scale=TEMPERATURE_SCALE,
                            offset=TEMPERATURE_OFFSET,
                        ),
                        library.BinaryField(name="state", start_bit=44, bit_length=4),
                        library.BinaryField(
                            name="speed",
                            start_bit=55,
                            bit_length=16,
                            byte_order="big_endian",
                        ),
                    ],
                ),
            ),
            library.MessageEntry(
                name="slow",
                payload=library.BinaryPayload(
                    identifier=library.CanIdentifier(
                        id=can_frames.SLOW_ID & SLOW_MASK, extended=True, mask=SLOW_MASK
                    ),
                    length=8,
                    fields=[
                        library.BinaryField(name="count", start_bit=0, bit_length=32),
                        library.BinaryField(
                            name="ratio", start_bit=32, bit_length=32, float=True
                        ),
                    ],
                ),
            ),
        ]

    def create_task(self, device: sy.Device, library_key: library.Key) -> can.ReadTask:
        fast_msg, self.fast = self.bind_read(
            "fast",
            ["count", "sine", "temperature", "state", "speed"],
            {
                "count": sy.DataType.UINT16,
                "state": sy.DataType.UINT8,
                "speed": sy.DataType.UINT16,
            },
        )
        slow_msg, self.slow = self.bind_read(
            "slow", ["count", "ratio"], {"count": sy.DataType.UINT32}
        )
        return can.ReadTask(
            name="CAN Read",
            device=device.key,
            library=library_key,
            messages=[fast_msg, slow_msg],
        )

    def run(self) -> None:
        with self.statuses() as statuses:
            with self.collect([self.fast, self.slow]) as samples, self.task.run():
                samples.wait(count=10)
            self.assert_no_problems(statuses)
        assert_indexed("fast", self.fast, samples)
        assert_indexed("slow", self.slow, samples)
        fast, slow = samples.count(self.fast), samples.count(self.slow)
        if fast < 2 * slow:
            raise AssertionError(
                f"Expected at least twice as many fast messages as slow ones, got "
                f"{fast} fast and {slow} slow"
            )
        self._verify_fast(samples)
        self._verify_slow(samples)

    def _verify_fast(self, samples: Collector) -> None:
        counts = samples[self.fast.fields["count"]].astype(np.int64)
        assert_increasing("fast.count", counts)
        assert_equal(
            "fast.sine",
            samples[self.fast.fields["sine"]],
            [can_frames.sine(int(n)) * SINE_SCALE for n in counts],
        )
        raw = [can_frames.temperature(int(n)) for n in counts]
        assert_equal(
            "fast.temperature",
            samples[self.fast.fields["temperature"]],
            [t * TEMPERATURE_SCALE + TEMPERATURE_OFFSET for t in raw],
        )
        if min(raw) >= 0:
            raise AssertionError("fast.temperature: expected negative raw samples")
        assert_equal(
            "fast.state", samples[self.fast.fields["state"]], [n % 16 for n in counts]
        )
        assert_equal(
            "fast.speed",
            samples[self.fast.fields["speed"]],
            [can_frames.speed(int(n)) for n in counts],
        )

    def _verify_slow(self, samples: Collector) -> None:
        counts = samples[self.slow.fields["count"]].astype(np.int64)
        assert_increasing("slow.count", counts)
        assert_equal(
            "slow.ratio",
            samples[self.slow.fields["ratio"]],
            [int(n) * 0.25 for n in counts],
        )

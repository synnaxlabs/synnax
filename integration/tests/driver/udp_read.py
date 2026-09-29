#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
from examples.simulators import udp_telemetry
from examples.simulators.udp_telemetry import UDPTelemetrySim

import synnax as sy
from synnax import library, udp
from tests.driver.bus_task import (
    Bound,
    BusCase,
    Collector,
    assert_equal,
    assert_increasing,
    assert_indexed,
    typed_message,
)

SINE_SCALE = 2**-14
TEMPERATURE_SCALE = 0.5
TEMPERATURE_OFFSET = 20.0


class UDPRead(BusCase):
    """Decodes two datagram types, told apart by a type byte, into channels with an
    index each. The fast datagram covers signed, big-endian, float, and scaled fields,
    and the slow one covers fields narrower than a byte. The simulator also sends a
    datagram of an unknown type, which the task must ignore."""

    sim_classes = [UDPTelemetrySim]
    prefix = "udp_read"
    fast: Bound
    slow: Bound

    def create_entries(self) -> list[library.Entry]:
        return [
            typed_message(
                "fast",
                type_value=udp_telemetry.FAST_TYPE,
                type_start_bit=0,
                length=13,
                fields=[
                    library.BinaryField(name="count", start_bit=8, bit_length=32),
                    library.BinaryField(
                        name="sine",
                        start_bit=40,
                        bit_length=16,
                        signed=True,
                        scale=SINE_SCALE,
                    ),
                    library.BinaryField(
                        name="temperature",
                        start_bit=63,
                        bit_length=16,
                        byte_order="big_endian",
                        signed=True,
                        scale=TEMPERATURE_SCALE,
                        offset=TEMPERATURE_OFFSET,
                    ),
                    library.BinaryField(
                        name="ratio", start_bit=72, bit_length=32, float=True
                    ),
                ],
            ),
            typed_message(
                "slow",
                type_value=udp_telemetry.SLOW_TYPE,
                type_start_bit=0,
                length=4,
                fields=[
                    library.BinaryField(name="count", start_bit=8, bit_length=16),
                    library.BinaryField(name="mode", start_bit=24, bit_length=3),
                    library.BinaryField(name="odd", start_bit=27, bit_length=1),
                ],
            ),
        ]

    def create_task(self, device: sy.Device, library_key: library.Key) -> udp.ReadTask:
        fast_msg, self.fast = self.bind_read(
            "fast",
            ["count", "sine", "temperature", "ratio"],
            {"count": sy.DataType.UINT32},
        )
        slow_msg, self.slow = self.bind_read(
            "slow",
            ["count", "mode", "odd"],
            {
                "count": sy.DataType.UINT16,
                "mode": sy.DataType.UINT8,
                "odd": sy.DataType.UINT8,
            },
        )
        return udp.ReadTask(
            name="UDP Read",
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
            [udp_telemetry.sine(int(n)) * SINE_SCALE for n in counts],
        )
        raw = [udp_telemetry.temperature(int(n)) for n in counts]
        assert_equal(
            "fast.temperature",
            samples[self.fast.fields["temperature"]],
            [t * TEMPERATURE_SCALE + TEMPERATURE_OFFSET for t in raw],
        )
        if min(raw) >= 0:
            raise AssertionError("fast.temperature: expected negative raw samples")
        assert_equal(
            "fast.ratio",
            samples[self.fast.fields["ratio"]],
            [int(n) * 0.25 for n in counts],
        )

    def _verify_slow(self, samples: Collector) -> None:
        counts = samples[self.slow.fields["count"]].astype(np.int64)
        assert_increasing("slow.count", counts)
        assert_equal(
            "slow.mode", samples[self.slow.fields["mode"]], [n % 8 for n in counts]
        )
        assert_equal(
            "slow.odd", samples[self.slow.fields["odd"]], [n % 2 for n in counts]
        )

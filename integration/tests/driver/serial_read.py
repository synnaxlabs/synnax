#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
from examples.simulators import serial_frames

import synnax as sy
from synnax import library, serial
from tests.driver.bus_task import (
    Bound,
    Collector,
    assert_equal,
    assert_increasing,
    assert_indexed,
    typed_message,
)
from tests.driver.serial_task import SerialCase

PRESSURE_SCALE = 0.0625
PRESSURE_OFFSET = 14.7
FLOW_SCALE = 0.5
# The type byte follows the two sync bytes and the length byte.
TYPE_START_BIT = 24


class SerialRead(SerialCase):
    """Splits a byte stream into sync-and-length frames with a CRC-16/MODBUS
    checksum and decodes two frame types, told apart by a type byte after the
    header, into channels with an index each."""

    prefix = "serial_read"
    telemetry: Bound
    status: Bound

    def create_entries(self) -> list[library.Entry]:
        return [
            typed_message(
                "telemetry",
                type_value=serial_frames.TELEMETRY_TYPE,
                type_start_bit=TYPE_START_BIT,
                length=10,
                fields=[
                    library.BinaryField(name="count", start_bit=32, bit_length=16),
                    library.BinaryField(
                        name="pressure",
                        start_bit=48,
                        bit_length=16,
                        signed=True,
                        scale=PRESSURE_SCALE,
                        offset=PRESSURE_OFFSET,
                    ),
                    library.BinaryField(
                        name="flow",
                        start_bit=71,
                        bit_length=16,
                        byte_order="big_endian",
                        scale=FLOW_SCALE,
                    ),
                ],
            ),
            typed_message(
                "status",
                type_value=serial_frames.STATUS_TYPE,
                type_start_bit=TYPE_START_BIT,
                length=6,
                fields=[
                    library.BinaryField(name="count", start_bit=32, bit_length=8),
                    library.BinaryField(
                        name="trim", start_bit=40, bit_length=8, signed=True
                    ),
                ],
            ),
        ]

    def create_task(
        self, device: sy.Device, library_key: library.Key
    ) -> serial.ReadTask:
        telemetry_msg, self.telemetry = self.bind_read(
            "telemetry",
            ["count", "pressure", "flow"],
            {"count": sy.DataType.UINT16},
        )
        status_msg, self.status = self.bind_read(
            "status",
            ["count", "trim"],
            {"count": sy.DataType.UINT8, "trim": sy.DataType.INT8},
        )
        return serial.ReadTask(
            name="Serial Read",
            device=device.key,
            library=library_key,
            messages=[telemetry_msg, status_msg],
            framing=self.framing(),
        )

    def run(self) -> None:
        with self.statuses() as statuses:
            with self.collect([self.telemetry, self.status]) as samples:
                with self.task.run():
                    samples.wait(count=10)
            self.assert_no_problems(statuses)
        assert_indexed("telemetry", self.telemetry, samples)
        assert_indexed("status", self.status, samples)
        self._verify_telemetry(samples)
        self._verify_status(samples)

    def _verify_telemetry(self, samples: Collector) -> None:
        counts = samples[self.telemetry.fields["count"]].astype(np.int64)
        assert_increasing("telemetry.count", counts)
        assert_equal(
            "telemetry.pressure",
            samples[self.telemetry.fields["pressure"]],
            [
                serial_frames.pressure(int(n)) * PRESSURE_SCALE + PRESSURE_OFFSET
                for n in counts
            ],
        )
        assert_equal(
            "telemetry.flow",
            samples[self.telemetry.fields["flow"]],
            [serial_frames.flow(int(n)) * FLOW_SCALE for n in counts],
        )

    def _verify_status(self, samples: Collector) -> None:
        counts = samples[self.status.fields["count"]].astype(np.int64)
        trim = samples[self.status.fields["trim"]]
        assert_equal("status.trim", trim, [serial_frames.trim(int(n)) for n in counts])
        if not np.any(trim < 0):
            raise AssertionError("status.trim: expected negative samples")

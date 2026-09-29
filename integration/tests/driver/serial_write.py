#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import struct

from examples.simulators import serial_frames
from examples.simulators.serial_frames import SerialFrameSim

import synnax as sy
from synnax import library, serial
from tests.driver.serial_task import SerialCase

SETPOINT_SCALE = 0.25

# Setpoint and trim commands, and the raw values each round must encode after the
# header: a little-endian uint16 at a quarter unit per count and an int8.
ROUNDS = [
    (12.5, -3.0, 50, -3),
    (100.25, 42.0, 401, 42),
]


class SerialWrite(SerialCase):
    """Sends commands as sync-and-length frames and checks that the device receives
    each frame with its header and CRC-16/MODBUS checksum in place."""

    prefix = "serial_write"
    commands: dict[str, int]

    def create_entries(self) -> list[library.Entry]:
        return [
            library.MessageEntry(
                name="command",
                length=6,
                fields=[
                    library.BinaryField(
                        name="setpoint",
                        start_bit=24,
                        bit_length=16,
                        scale=SETPOINT_SCALE,
                    ),
                    library.BinaryField(
                        name="trim", start_bit=40, bit_length=8, signed=True
                    ),
                ],
            )
        ]

    def create_task(
        self, device: sy.Device, library_key: library.Key
    ) -> serial.WriteTask:
        message, self.commands = self.bind_write("command", ["setpoint", "trim"])
        return serial.WriteTask(
            name="Serial Write",
            device=device.key,
            library=library_key,
            messages=[message],
            framing=self.framing(),
        )

    def run(self) -> None:
        sim = self.simulator(SerialFrameSim)
        with self.statuses() as statuses:
            with self.task.run():
                for setpoint, trim, raw_setpoint, raw_trim in ROUNDS:
                    frame = serial_frames.frame(
                        struct.pack("<Hb", raw_setpoint, raw_trim)
                    )
                    self.command_until(
                        {
                            self.commands["setpoint"]: setpoint,
                            self.commands["trim"]: trim,
                        },
                        lambda: sim.wait_for(frame),
                    )
            self.assert_no_problems(statuses)

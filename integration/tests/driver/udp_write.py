#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators.udp_telemetry import UDPTelemetrySim

import synnax as sy
from synnax import library, udp
from tests.driver.bus_task import BusCase

# Setpoint, trim, and mode commands, and the datagram each round must send. The
# setpoint is a little-endian uint16 at half a unit per count, the trim a big-endian
# int16, the mode a uint8, and the last byte is a field no command channel drives.
ROUNDS = [
    (12.5, -3.0, 2.0, bytes.fromhex("1900fffd0200")),
    (100.0, 300.0, 7.0, bytes.fromhex("c800012c0700")),
]


class UDPWrite(BusCase):
    """Sends commands as one datagram and checks the bytes the device receives."""

    sim_classes = [UDPTelemetrySim]
    prefix = "udp_write"
    commands: dict[str, int]

    def create_entries(self) -> list[library.Entry]:
        return [
            library.MessageEntry(
                name="command",
                payload=library.BinaryPayload(
                    length=6,
                    fields=[
                        library.BinaryField(
                            name="setpoint", start_bit=0, bit_length=16, scale=0.5
                        ),
                        library.BinaryField(
                            name="trim",
                            start_bit=23,
                            bit_length=16,
                            byte_order="big_endian",
                            signed=True,
                        ),
                        library.BinaryField(name="mode", start_bit=32, bit_length=8),
                        library.BinaryField(name="spare", start_bit=40, bit_length=8),
                    ],
                ),
            )
        ]

    def create_task(self, device: sy.Device, library_key: library.Key) -> udp.WriteTask:
        message, self.commands = self.bind_write(
            "command", ["setpoint", "trim", "mode"]
        )
        return udp.WriteTask(
            name="UDP Write",
            device=device.key,
            library=library_key,
            messages=[message],
        )

    def run(self) -> None:
        sim = self.simulator(UDPTelemetrySim)
        with self.statuses() as statuses:
            with self.task.run():
                for setpoint, trim, mode, datagram in ROUNDS:
                    self.command_until(
                        {
                            self.commands["setpoint"]: setpoint,
                            self.commands["trim"]: trim,
                            self.commands["mode"]: mode,
                        },
                        lambda: sim.wait_for(datagram),
                    )
            self.assert_no_problems(statuses)

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators.can_frames import CANFrameSim, Frame

import synnax as sy
from synnax import can, library
from tests.driver.can_task import CANCase

COMMAND_ID = 0x200
HEARTBEAT_ID = 0x201
HEARTBEAT_PERIOD = 100 * sy.TimeSpan.MILLISECOND
# The heartbeats to wait for after each command, and the window to wait in.
HEARTBEATS = 5
HEARTBEAT_WINDOW = 3 * sy.TimeSpan.SECOND
# How long the case watches the bus to show that the task sends nothing.
QUIET_WINDOW = 0.5

# Setpoint, trim, enabled, and beat commands, and the command and heartbeat payloads
# each round must send. The setpoint is a little-endian uint16 at half a unit per
# count, the trim an int8, and enabled one bit.
ROUNDS = [
    (12.5, -3.0, 1.0, 7.0, bytes.fromhex("1900fd0100000000"), bytes.fromhex("0700")),
    (100.0, 42.0, 0.0, 9.0, bytes.fromhex("c8002a0000000000"), bytes.fromhex("0900")),
]


class CANWrite(CANCase):
    """Sends an on-change command frame and a periodic heartbeat frame. The heartbeat
    waits for its command channel to have a value, then repeats on its period with
    the latest value, and stops when the task stops."""

    prefix = "can_write"
    commands: dict[str, int]
    beat: dict[str, int]

    def create_entries(self) -> list[library.Entry]:
        return [
            library.MessageEntry(
                name="command",
                payload=library.BinaryPayload(
                    identifier=library.CanIdentifier(id=COMMAND_ID),
                    length=8,
                    fields=[
                        library.BinaryField(
                            name="setpoint", start_bit=0, bit_length=16, scale=0.5
                        ),
                        library.BinaryField(
                            name="trim", start_bit=16, bit_length=8, signed=True
                        ),
                        library.BinaryField(name="enabled", start_bit=24, bit_length=1),
                    ],
                ),
            ),
            library.MessageEntry(
                name="heartbeat",
                period=HEARTBEAT_PERIOD,
                payload=library.BinaryPayload(
                    identifier=library.CanIdentifier(id=HEARTBEAT_ID),
                    length=2,
                    fields=[
                        library.BinaryField(name="beat", start_bit=0, bit_length=16)
                    ],
                ),
            ),
        ]

    def create_task(self, device: sy.Device, library_key: library.Key) -> can.WriteTask:
        command, self.commands = self.bind_write(
            "command", ["setpoint", "trim", "enabled"]
        )
        heartbeat, self.beat = self.bind_write("heartbeat", ["beat"])
        return can.WriteTask(
            name="CAN Write",
            device=device.key,
            library=library_key,
            messages=[command, heartbeat],
        )

    def run(self) -> None:
        sim = self.simulator(CANFrameSim)
        with self.statuses() as statuses:
            with self.task.run():
                sy.sleep(QUIET_WINDOW)
                if sim.received():
                    raise AssertionError(
                        f"Expected no frames before any command, got {sim.received()}"
                    )
                for setpoint, trim, enabled, beat, command, heartbeat in ROUNDS:
                    self.command_until(
                        {
                            self.commands["setpoint"]: setpoint,
                            self.commands["trim"]: trim,
                            self.commands["enabled"]: enabled,
                            self.beat["beat"]: beat,
                        },
                        lambda: sim.wait_for(Frame(COMMAND_ID, False, command)),
                    )
                    sim.wait_for(
                        Frame(HEARTBEAT_ID, False, heartbeat),
                        count=HEARTBEATS,
                        timeout=HEARTBEAT_WINDOW,
                    )
            self.assert_no_problems(statuses)
        sy.sleep(QUIET_WINDOW)
        stopped = len(sim.received())
        sy.sleep(QUIET_WINDOW)
        if len(sim.received()) != stopped:
            raise AssertionError(
                f"Expected no frames after the task stopped, got "
                f"{sim.received()[stopped:]}"
            )

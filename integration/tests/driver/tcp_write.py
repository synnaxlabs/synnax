#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators.scpi import SCPISim

import synnax as sy
from synnax import bus, library, tcp
from tests.driver.bus_task import BusCase

CURRENT_SCALE = 0.5

# Voltage and current commands, and the line each must put on the wire.
ROUNDS = [
    (3.25, 1.5, "VOLT 3.25;CURR 3"),
    (12.0, 0.75, "VOLT 12;CURR 1.5"),
]


class TCPWrite(BusCase):
    """Sends setpoints as one SCPI line of tagged fields and checks the line the
    instrument receives and the setpoints it reports back."""

    sim_classes = [SCPISim]
    prefix = "tcp_write"
    commands: dict[str, int]

    def create_entries(self) -> list[library.Entry]:
        return [
            library.MessageEntry(
                name="setpoints",
                format="text",
                delimiter=";",
                fields=[
                    library.TaggedField(name="voltage", tag="VOLT "),
                    library.TaggedField(
                        name="current", tag="CURR ", scale=CURRENT_SCALE
                    ),
                ],
            )
        ]

    def create_task(self, device: sy.Device, library_key: library.Key) -> tcp.WriteTask:
        message, self.commands = self.bind_write("setpoints", ["voltage", "current"])
        return tcp.WriteTask(
            name="TCP Write",
            device=device.key,
            library=library_key,
            messages=[message],
            framing=bus.DelimiterFraming(delimiter="\n"),
        )

    def run(self) -> None:
        sim = self.simulator(SCPISim)
        with self.statuses() as statuses:
            with self.task.run():
                for voltage, current, line in ROUNDS:
                    self.command_until(
                        {
                            self.commands["voltage"]: voltage,
                            self.commands["current"]: current,
                        },
                        lambda: sim.wait_for(line),
                    )
                    self._verify_setpoint(sim, "VOLT?", voltage)
                    self._verify_setpoint(sim, "CURR?", current / CURRENT_SCALE)
            self.assert_no_problems(statuses)

    @staticmethod
    def _verify_setpoint(sim: SCPISim, query: str, expected: float) -> None:
        actual = float(sim.query(query))
        if actual != expected:
            raise AssertionError(f"{query} expected {expected}, got {actual}")

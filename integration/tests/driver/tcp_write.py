#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from functools import partial

from examples.simulators.scpi import SCPISim

import synnax as sy
from synnax import bus, icd, tcp
from tests.driver.bus_task import BusCase

CURRENT_SCALE = 0.5
MAX_VOLTAGE = 30.0

# Voltage and current commands, and the line each must put on the wire.
ROUNDS = [
    (3.25, 1.5, "VOLT 3.25;CURR 3"),
    (12.0, 0.75, "VOLT 12;CURR 1.5"),
]


def verify_setpoint(sim: SCPISim, query: str, expected: float) -> None:
    """Asserts that the instrument answers query with expected."""
    actual = float(sim.query(query))
    if actual != expected:
        raise AssertionError(f"{query} expected {expected}, got {actual}")


class SetpointsCase(BusCase):
    """Runs a write task that sends a voltage and a current setpoint as one SCPI line
    of tagged fields. The voltage accepts 0 to MAX_VOLTAGE."""

    sim_classes = [SCPISim]
    commands: dict[str, int]

    def create_entries(self) -> list[icd.Entry]:
        return [
            icd.MessageEntry(
                name="setpoints",
                payload=icd.TextPayload(
                    delimiter=";",
                    fields=[
                        icd.TaggedTextField(
                            name="voltage", tag="VOLT ", min=0, max=MAX_VOLTAGE
                        ),
                        icd.TaggedTextField(
                            name="current", tag="CURR ", scale=CURRENT_SCALE
                        ),
                    ],
                ),
            )
        ]

    def create_task(self, device: sy.Device, icd_key: icd.Key) -> tcp.WriteTask:
        message, self.commands = self.bind_write("setpoints", ["voltage", "current"])
        return tcp.WriteTask(
            name=type(self).__name__,
            device=device.key,
            icd=icd_key,
            messages=[message],
            framing=bus.DelimiterFraming(delimiter="\n"),
        )


class TCPWrite(SetpointsCase):
    """Sends setpoints and checks the line the instrument receives and the setpoints
    it reports back."""

    prefix = "tcp_write"

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
                        partial(sim.wait_for, line),
                    )
                    verify_setpoint(sim, "VOLT?", voltage)
                    verify_setpoint(sim, "CURR?", current / CURRENT_SCALE)
            self.assert_no_problems(statuses)


class TCPWriteWaits(SetpointsCase):
    """Commands only the voltage. The task warns that the message waits for the
    current and sends nothing until the current has a value too."""

    prefix = "tcp_write_waits"

    def run(self) -> None:
        sim = self.simulator(SCPISim)
        with self.statuses() as statuses, self.task.run():
            self.command_until(
                {self.commands["voltage"]: 3.25},
                partial(self._expect_warning, statuses, "setpoints waits for current"),
            )
            if received := sim.received():
                raise AssertionError(f"Expected no line yet, got {received!r}")
            self.command_until(
                {self.commands["current"]: 1.5},
                partial(sim.wait_for, "VOLT 3.25;CURR 3"),
            )
            self.wait_for_status(statuses, "success")

    def _expect_warning(self, statuses: sy.Streamer, text: str) -> None:
        message = self.wait_for_status(
            statuses, "warning", containing=text, timeout=2 * sy.TimeSpan.SECOND
        )
        self.log(f"Warning: {message}")


class TCPWriteOutOfRange(SetpointsCase):
    """Commands a voltage above its max. The task refuses it with a warning, and the
    next line carries the last voltage the task accepted."""

    prefix = "tcp_write_range"

    def run(self) -> None:
        sim = self.simulator(SCPISim)
        voltage, current = self.commands["voltage"], self.commands["current"]
        with self.statuses() as statuses, self.task.run():
            self.command_until(
                {voltage: 3.25, current: 1.5},
                partial(sim.wait_for, "VOLT 3.25;CURR 3"),
            )
            self.command({voltage: 50.0})
            message = self.wait_for_status(
                statuses,
                "warning",
                containing="setpoints.voltage: 50 is outside the range",
            )
            self.log(f"Warning: {message}")
            self.command({current: 2.0})
            sim.wait_for("VOLT 3.25;CURR 4")
            self.wait_for_status(statuses, "success")
        lines = sim.received()
        if any(line.startswith("VOLT 50") for line in lines):
            raise AssertionError(f"The refused voltage went out: {lines!r}")
        verify_setpoint(sim, "VOLT?", 3.25)

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Button mode semantics and the activation-delay hold guard on controls."""

import synnax as sy
from console.case import ConsoleCase
from console.schematic import Button, Valve
from console.schematic.schematic import Schematic
from x import random_name

# Configured activation delay and the hold that must satisfy it. The hold is
# comfortably past the delay so the timer fires while the mouse is still down.
DELAY_MS = 500
HOLD = sy.TimeSpan.MILLISECOND * 1200


class HoldToActuate(ConsoleCase):
    """Fire, momentary, and pulse button modes, and activation-delay guards on
    a button and a valve."""

    def _create_cmd_channel(self, name: str) -> None:
        idx = self.client.channels.create(
            name=f"{name}_idx", is_index=True, retrieve_if_name_exists=True
        )
        self.client.channels.create(
            name=name,
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        suffix = random_name()
        self.start = sy.TimeStamp.now()
        self.fire_ch = f"btn_fire_{suffix}"
        self.momentary_ch = f"btn_momentary_{suffix}"
        self.pulse_ch = f"btn_pulse_{suffix}"
        self.delay_ch = f"btn_delay_{suffix}"
        self.valve_cmd_ch = f"valve_cmd_{suffix}"
        self.valve_state_ch = f"valve_state_{suffix}"
        for name in (
            self.fire_ch,
            self.momentary_ch,
            self.pulse_ch,
            self.delay_ch,
            self.valve_cmd_ch,
            self.valve_state_ch,
        ):
            self._create_cmd_channel(name)

        self.log("Creating the schematic and its control symbols")
        schematic = self.console.pages.create(Schematic, f"hold_{suffix}")
        self._cleanup_pages.append(schematic.page_name)

        self.fire_btn = schematic.create_symbol(
            Button(label=self.fire_ch, channel_name=self.fire_ch, mode="Fire")
        )
        self.fire_btn.move(delta_x=-320, delta_y=-140)
        self.momentary_btn = schematic.create_symbol(
            Button(
                label=self.momentary_ch,
                channel_name=self.momentary_ch,
                mode="Momentary",
            )
        )
        self.momentary_btn.move(delta_x=320, delta_y=-140)
        self.pulse_btn = schematic.create_symbol(
            Button(label=self.pulse_ch, channel_name=self.pulse_ch, mode="Pulse")
        )
        self.pulse_btn.move(delta_x=-320, delta_y=140)
        self.delay_btn = schematic.create_symbol(
            Button(
                label=self.delay_ch,
                channel_name=self.delay_ch,
                mode="Fire",
                activation_delay=DELAY_MS,
            )
        )
        self.delay_btn.move(delta_x=320, delta_y=140)
        self.valve = schematic.create_symbol(
            Valve(
                label=self.valve_cmd_ch,
                state_channel=self.valve_state_ch,
                command_channel=self.valve_cmd_ch,
                activation_delay=DELAY_MS,
            )
        )
        self.valve.move(delta_x=0, delta_y=140)

        self.test_fire_button()
        self.test_momentary_button()
        self.test_pulse_button()
        self.test_delay_button()
        self.test_delay_valve()

    def test_fire_button(self) -> None:
        """Fire mode writes true on release."""
        self.log("Pressing the fire button")
        self.fire_btn.press()
        self.wait_for_eq(self.fire_ch, 1)

    def test_momentary_button(self) -> None:
        """Momentary mode writes true on press and false on release."""
        self.log("Holding the momentary button")
        with self.momentary_btn.hold():
            self.wait_for_eq(self.momentary_ch, 1)
        self.wait_for_eq(self.momentary_ch, 0)

    def test_pulse_button(self) -> None:
        """Pulse mode writes true then false on press."""
        self.log("Pressing the pulse button")
        self.pulse_btn.press()
        self.wait_for_eq(self.pulse_ch, 0)
        series = self.client.read(
            sy.TimeRange(self.start, sy.TimeStamp.now()), self.pulse_ch
        )
        values = list(series)
        assert values == [1.0, 0.0], f"pulse should write [1, 0], got {values}"

    def test_delay_button(self) -> None:
        """An activation delay swallows a short click and fires after a hold."""
        self.log("Short-clicking the delayed button")
        self.delay_btn.press()
        sy.sleep(1.0)
        value = self.get_value(self.delay_ch)
        assert value is None, (
            f"a click shorter than the activation delay must not write, got {value}"
        )
        self.log("Holding the delayed button past its activation delay")
        self.delay_btn.press_and_hold(HOLD)
        self.wait_for_eq(self.delay_ch, 1)

    def test_delay_valve(self) -> None:
        """A valve with an activation delay only actuates after a hold."""
        self.log("Short-clicking the delayed valve")
        self.valve.press()
        sy.sleep(1.0)
        value = self.get_value(self.valve_cmd_ch)
        assert value is None, (
            f"a click shorter than the activation delay must not actuate, got {value}"
        )
        self.log("Holding the delayed valve past its activation delay")
        self.valve.press_and_hold(HOLD)
        self.wait_for_eq(self.valve_cmd_ch, 1)

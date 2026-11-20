#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.case import ConsoleCase
from console.schematic import (
    Light,
    Schematic,
    Select,
    Setpoint,
    StateIndicator,
    Symbol,
    Value,
)
from x import random_name

# The default stale timeout is 5 s, and the sweep reports a transition up to 1 s late.
STALE_WAIT = 15 * sy.TimeSpan.SECOND


class StaleIndicator(ConsoleCase):
    """Show the stale indicator on state symbols, a value, and a setpoint read-back.

    Writes one sample to each channel, waits past the stale timeout, and checks that
    each symbol shows the stale indicator. A new sample clears it.
    """

    def run(self) -> None:
        client = self.client
        suffix = random_name()
        index = client.channels.create(
            name=f"stale_idx_{suffix}", is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        light_ch = client.channels.create(
            name=f"stale_light_{suffix}", data_type=sy.DataType.UINT8, index=index.key
        )
        state_ch = client.channels.create(
            name=f"stale_state_{suffix}",
            data_type=sy.DataType.FLOAT64,
            index=index.key,
        )
        value_ch = client.channels.create(
            name=f"stale_value_{suffix}",
            data_type=sy.DataType.FLOAT64,
            index=index.key,
        )
        command_ch = client.channels.create(
            name=f"stale_cmd_{suffix}", data_type=sy.DataType.FLOAT64, index=index.key
        )
        setpoint_ch = client.channels.create(
            name=f"stale_sp_cmd_{suffix}",
            data_type=sy.DataType.FLOAT64,
            index=index.key,
        )
        options = [
            sy.schematic.StateMapping(key="open", name="Open", value=1),
            sy.schematic.StateMapping(key="closed", name="Closed", value=2),
        ]

        schematic = self.console.pages.create(Schematic, f"stale_indicator_{suffix}")
        self._cleanup_pages.append(schematic.page_name)
        light = schematic.create_symbol(
            Light(label="Stale Light", channel_name=light_ch.name)
        )
        light.move(delta_x=-250, delta_y=0)
        indicator = schematic.create_symbol(
            StateIndicator(
                label="Stale State", channel_name=state_ch.name, options=options
            )
        )
        select = schematic.create_symbol(
            Select(
                label="Stale Select",
                channel_name=command_ch.name,
                state_channel_name=state_ch.name,
                options=options,
            )
        )
        select.move(delta_x=250, delta_y=0)
        value = schematic.create_symbol(
            Value(label="Stale Value", channel_name=value_ch.name)
        )
        value.move(delta_x=0, delta_y=200)
        setpoint = schematic.create_symbol(
            Setpoint(
                label="Stale Setpoint",
                channel_name=setpoint_ch.name,
                state_channel_name=state_ch.name,
            )
        )
        setpoint.move(delta_x=250, delta_y=200)
        symbols: list[Symbol] = [light, indicator, select, value, setpoint]

        channels = [light_ch.key, state_ch.key, value_ch.key]
        self.log("Writing one sample to each channel")
        self._write(index.key, channels)
        self.log("Waiting for each symbol to show the stale indicator")
        for symbol in symbols:
            self._assert_stale(symbol, True)

        self.log("Writing a new sample to clear the stale indicators")
        self._write(index.key, channels)
        for symbol in symbols:
            self._assert_stale(symbol, False)

    def _write(self, index: int, channels: list[int]) -> None:
        with self.client.open_writer(sy.TimeStamp.now(), [index, *channels]) as writer:
            writer.write({index: sy.TimeStamp.now(), **dict.fromkeys(channels, 1)})

    def _assert_stale(self, symbol: Symbol, stale: bool) -> None:
        indicator = symbol.stale_indicator

        def settled() -> bool | None:
            return True if indicator.is_visible() == stale else None

        sy.poll(settled, timeout=STALE_WAIT, interval=0.25)
        assert indicator.is_visible() == stale, (
            f"{symbol.label}: expected the stale indicator "
            f"{'shown' if stale else 'hidden'} within {STALE_WAIT}"
        )

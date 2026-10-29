#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import expect

import synnax as sy
from console.case import ConsoleCase
from console.schematic import Schematic, Symbol, Valve

CHANNEL_NAME = "change_symbol_channel"
INDEX_NAME = "change_symbol_index"


class ChangeSymbol(ConsoleCase):
    """Change placed symbols to another symbol from the toolbar header."""

    def setup(self) -> None:
        super().setup()
        index_ch = self.client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.UINT8,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        schematic = self.console.pages.create(Schematic, "change_symbol_schematic")
        self._cleanup_pages.append(schematic.page_name)

        valve_a = schematic.create_symbol(
            Valve(
                label="change_symbol_valve_a",
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_a.move(delta_x=-150, delta_y=0)
        valve_b = schematic.create_symbol(
            Valve(
                label="change_symbol_valve_b",
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_b.move(delta_x=150, delta_y=0)
        schematic.connect_symbols(
            source_symbol=valve_a,
            source_handle="right",
            target_symbol=valve_b,
            target_handle="left",
        )
        assert schematic.get_edge_count() == 1, "Expected one edge between the valves"

        self.test_change_keeps_settings(schematic, valve=valve_a)
        self.test_change_selection(schematic, valve_a=valve_a, valve_b=valve_b)
        self.test_undo_restores_selection(schematic, valve_a=valve_a, valve_b=valve_b)

    def test_change_keeps_settings(
        self, schematic: Schematic, *, valve: Symbol
    ) -> None:
        """Test that a changed symbol keeps its label, channels, and edges."""
        self.log("Testing that a change keeps the symbol's settings")

        schematic.select([valve])
        expect(schematic.symbol_name).to_have_text("Valve")
        schematic.change_symbol("Solenoid")
        expect(schematic.symbol_name).to_have_text("Solenoid")
        assert schematic.get_symbol_count() == 2, "Expected a change to add no symbol"

        expect(valve.locator).to_contain_text("change_symbol_valve_a")
        props = valve.get_properties()
        assert props["state_channel"] == CHANNEL_NAME, (
            f"State channel mismatch! Actual: {props['state_channel']}"
        )
        assert props["command_channel"] == CHANNEL_NAME, (
            f"Command channel mismatch! Actual: {props['command_channel']}"
        )
        assert schematic.get_edge_count() == 1, "Expected the edge to survive"

    def test_change_selection(
        self, schematic: Schematic, *, valve_a: Symbol, valve_b: Symbol
    ) -> None:
        """Test that a change applies to every selected symbol."""
        self.log("Testing a change of several selected symbols")

        schematic.select([valve_a, valve_b])
        expect(schematic.symbol_name).to_have_text("Mixed")
        schematic.change_symbol("Ball")
        expect(schematic.symbol_name).to_have_text("Ball valve")
        assert schematic.get_edge_count() == 1, "Expected the edge to survive"

    def test_undo_restores_selection(
        self, schematic: Schematic, *, valve_a: Symbol, valve_b: Symbol
    ) -> None:
        """Test that one undo restores every symbol a change touched."""
        self.log("Testing that one undo restores the changed symbols")

        schematic._click_canvas()
        self.console.layout.press_key("ControlOrMeta+Z")

        schematic.select([valve_a])
        expect(schematic.symbol_name).to_have_text("Solenoid")
        schematic.select([valve_b])
        expect(schematic.symbol_name).to_have_text("Valve")

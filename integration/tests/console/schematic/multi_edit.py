#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

from playwright.sync_api import expect

import synnax as sy
from console.case import ConsoleCase
from console.schematic import Schematic, Symbol, Value, Valve

CHANNEL_NAME = "multi_edit_channel"
INDEX_NAME = "multi_edit_index"

RED = (255, 0, 0)
BLUE = (0, 0, 255)
GREEN = (0, 255, 0)


def expect_stroke(symbol: Symbol, rgb: tuple[int, int, int]) -> None:
    """Wait until the symbol renders its stroke in the given color."""
    r, g, b = rgb
    pattern = re.compile(rf"--pluto-symbol-color:\s*{r},\s*{g},\s*{b},\s*1\b")
    expect(symbol.colored).to_have_attribute("style", pattern)


class MultiEdit(ConsoleCase):
    """Edit the colors and property groups of several selected symbols at once."""

    def setup(self) -> None:
        super().setup()
        index_ch = self.client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.FLOAT64,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        schematic = self.console.pages.create(Schematic, "multi_edit_schematic")
        self._cleanup_pages.append(schematic.page_name)

        valve_a = schematic.create_symbol(
            Valve(
                label="multi_edit_valve_a",
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_a.move(delta_x=-200, delta_y=-80)
        valve_b = schematic.create_symbol(
            Valve(
                label="multi_edit_valve_b",
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_b.move(delta_x=200, delta_y=-80)
        value_a = schematic.create_symbol(
            Value(label="multi_edit_value_a", channel_name=CHANNEL_NAME)
        )
        value_a.move(delta_x=-200, delta_y=80)
        value_b = schematic.create_symbol(
            Value(label="multi_edit_value_b", channel_name=CHANNEL_NAME)
        )
        value_b.move(delta_x=200, delta_y=80)

        self.test_groups_follow_selection(schematic, valve_a, valve_b, value_a, value_b)
        self.test_stroke_applies_to_selection(schematic, [valve_a, valve_b, value_a])
        self.test_mixed_and_selection_recolor(schematic, valve_a, valve_b, value_a)

    def test_groups_follow_selection(
        self,
        schematic: Schematic,
        valve_a: Symbol,
        valve_b: Symbol,
        value_a: Symbol,
        value_b: Symbol,
    ) -> None:
        """Test that a control or group shows only when a selected symbol has it."""
        self.log("Testing that groups follow the selection")
        colors = schematic.colors

        schematic.select([valve_a, valve_b])
        expect(colors.field("Stroke")).to_be_visible()
        expect(colors.field("Fill")).to_have_count(0)
        expect(colors.field("Text")).to_have_count(0)
        expect(schematic.layout.form_section("Symbol size")).to_be_visible()

        schematic.select([valve_a, value_a])
        expect(colors.field("Fill")).to_be_visible()
        expect(colors.field("Text")).to_be_visible()

        schematic.select([value_a, value_b])
        expect(colors.field("Stroke")).to_be_visible()
        expect(schematic.layout.form_section("Symbol size")).to_have_count(0)

    def test_stroke_applies_to_selection(
        self, schematic: Schematic, symbols: list[Symbol]
    ) -> None:
        """Test that a stroke set on the selection recolors every selected symbol
        and joins the Selection control as one swatch."""
        self.log("Testing a stroke set on the selection")
        colors = schematic.colors

        schematic.select(symbols)
        expect(colors.field("Stroke")).to_have_attribute("placeholder", "Auto")
        expect(colors.selection_swatches()).to_have_count(0)

        colors.set("Stroke", "ff0000")
        for symbol in symbols:
            expect_stroke(symbol, RED)
        expect(colors.selection_swatches()).to_have_count(1)

    def test_mixed_and_selection_recolor(
        self,
        schematic: Schematic,
        valve_a: Symbol,
        valve_b: Symbol,
        value_a: Symbol,
    ) -> None:
        """Test that differing strokes show Mixed, and that a Selection swatch
        recolors only the symbols that hold its color."""
        self.log("Testing Mixed and a Selection swatch recolor")
        colors = schematic.colors

        schematic.select([valve_a, value_a])
        colors.set("Stroke", "0000ff")
        expect_stroke(valve_a, BLUE)
        expect_stroke(value_a, BLUE)

        schematic.select([valve_a, valve_b, value_a])
        expect(colors.field("Stroke")).to_have_attribute("placeholder", "Mixed")
        expect(colors.selection_swatches()).to_have_count(2)
        red = colors.selection_colors().index("rgba(255, 0, 0, 1)")

        colors.recolor(colors.selection_swatches().nth(red), "00ff00")
        expect_stroke(valve_b, GREEN)
        expect_stroke(valve_a, BLUE)
        expect_stroke(value_a, BLUE)

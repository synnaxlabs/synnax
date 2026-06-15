#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test custom schematic symbol operations."""

from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

import synnax as sy
from console.case import ConsoleCase
from console.schematic import CustomSymbol, Schematic
from console.schematic.symbol_toolbar import SymbolToolbar
from framework.utils import get_fixture_path
from x import random_name

TEST_SYMBOL_SVG = get_fixture_path("test_symbol.svg")
STYLED_SYMBOL_SVG = get_fixture_path("styled_symbol.svg")


class CustomSymbols(ConsoleCase):
    """Test custom schematic symbol operations.

    Tests cover:
    - Symbol group operations (create, rename, delete)
    - Symbol operations (create, rename, edit, delete, export)
    - Using custom symbols as actuators in schematics
    """

    suffix: str
    test_group_name: str
    test_symbol_name: str
    schematic_name: str
    idx_channel_name: str
    cmd_channel_name: str
    custom_symbol: CustomSymbol | None = None

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()
        self.test_group_name = f"Test Group {self.suffix}"
        self.test_symbol_name = f"Test Symbol {self.suffix}"
        self.schematic_name = f"Symbol Test Schematic {self.suffix}"
        self.idx_channel_name = f"idx_{self.suffix}"
        self.cmd_channel_name = f"cmd_{self.suffix}"

        index_ch = self.client.channels.create(
            name=self.idx_channel_name,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.cmd_channel = self.client.channels.create(
            name=self.cmd_channel_name,
            data_type=sy.DataType.UINT8,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

    def teardown(self) -> None:
        try:
            toolbar = SymbolToolbar(self.console.layout)
            toolbar.show()
            if toolbar.group_exists(self.test_group_name):
                toolbar.delete_group(self.test_group_name)
        except PlaywrightTimeoutError:
            pass
        self._cleanup_pages.append(self.schematic_name)
        super().teardown()

    def run(self) -> None:
        """Run all custom symbol tests."""
        schematic = self.console.project.create_schematic(self.schematic_name)
        toolbar = SymbolToolbar(self.console.layout)
        toolbar.show()
        self.console.notifications.close_all()

        self.test_create_symbol_group(toolbar)
        self.test_rename_symbol_group(toolbar)

        self.test_styled_symbol_applies_to_preview(toolbar)

        self.test_create_symbol(schematic, toolbar)
        self.test_rename_symbol(toolbar)
        self.test_rename_symbol_via_editor(toolbar)

        self.test_custom_symbol_as_actuator()

        self.test_export_symbol(toolbar)

        self.test_delete_symbol(toolbar)
        self.test_delete_symbol_group(toolbar)

    def test_create_symbol_group(self, toolbar: SymbolToolbar) -> None:
        """Test creating a new symbol group."""
        self.log("Testing create symbol group")
        toolbar.create_group(self.test_group_name)
        assert toolbar.group_exists(self.test_group_name), (
            f"Group '{self.test_group_name}' should exist after creation"
        )

    def test_rename_symbol_group(self, toolbar: SymbolToolbar) -> None:
        """Test renaming a symbol group."""
        self.log("Testing rename symbol group")
        old_name = self.test_group_name
        new_name = f"Renamed Group {self.suffix}"
        toolbar.rename_group(old_name, new_name)
        toolbar.wait_for_group_hidden(old_name)
        assert toolbar.group_exists(new_name), (
            f"Group '{new_name}' should exist after rename"
        )
        self.test_group_name = new_name

    def test_styled_symbol_applies_to_preview(self, toolbar: SymbolToolbar) -> None:
        """Editing a styled symbol's colors and scale must update the live preview.

        Regression for SY-4217. Real-world logo SVGs define colors via inline
        style="fill:..." and <style> class rules, both of which outrank the
        presentation attribute the renderer writes - so color edits used to compute
        but never paint. Import now normalizes colors onto attributes (so overrides
        win), strips <style> blocks (so they cannot leak), and excludes clip-path /
        defs shapes from regions. This drives the editor exactly as a user would and
        asserts on the rendered preview's computed colors.
        """
        self.log("Testing styled symbol normalization, recolor, and scale")
        toolbar.select_group(self.test_group_name)
        styled_name = f"Styled Symbol {self.suffix}"

        editor = toolbar.create_symbol()
        editor.upload_svg(STYLED_SYMBOL_SVG)

        # Reading a preview shape waits for the preview to render before we inspect it.
        # The rect's blue fill comes from an inline style; overriding it must paint.
        assert editor.get_preview_fill("#body") != "rgb(255, 0, 0)"

        # The clip-path shape must not surface as a phantom region: only the rect and
        # the circle are painted.
        assert editor.region_count() == 2, (
            f"expected 2 regions (clip-path excluded), got {editor.region_count()}"
        )

        editor.set_region_fill_color("#FF0000", region_index=0)
        editor.assert_preview_fill("#body", "rgb(255, 0, 0)")

        # The circle's green fill comes from a <style> class rule; overriding it must
        # paint once the block is stripped and the color flattened.
        editor.set_region_fill_color("#FF00FF", region_index=1)
        editor.assert_preview_fill("#badge", "rgb(255, 0, 255)")

        # The default scale must resize the rendered preview (100 -> 50 at 50%).
        editor.set_default_scale(50)
        editor.assert_preview_width(50)

        editor.set_name(styled_name)
        editor.save()
        toolbar.delete_symbol(styled_name)

    def test_create_symbol(self, schematic: Schematic, toolbar: SymbolToolbar) -> None:
        """Test creating a custom symbol with all properties set in the editor."""
        self.log("Testing create symbol")
        toolbar.select_group(self.test_group_name)

        editor = toolbar.create_symbol()
        editor.upload_svg(TEST_SYMBOL_SVG)
        editor.set_name(self.test_symbol_name)
        editor.add_handle()
        editor.set_region_stroke_color("#FF0000", region_index=0)
        editor.set_default_scale(50)
        editor.set_state("Actuator")
        editor.save()

        assert toolbar.symbol_exists(
            self.test_symbol_name, select_group=self.test_group_name
        ), f"Symbol '{self.test_symbol_name}' should exist after creation"

        custom_symbol_config = CustomSymbol(
            label="Actuator Test",
            symbol_name=self.test_symbol_name,
            group_name=self.test_group_name,
            command_channel=self.cmd_channel_name,
        )
        created_symbol = schematic.create_symbol(custom_symbol_config)
        assert isinstance(created_symbol, CustomSymbol)
        self.custom_symbol = created_symbol
        assert self.custom_symbol.symbol_id is not None, (
            "Custom symbol should have an ID"
        )
        symbol_id = self.custom_symbol.symbol_id
        assert symbol_id.startswith("rf__node-"), (
            f"Symbol ID should start with 'rf__node-', got '{symbol_id}'"
        )

    def test_rename_symbol(self, toolbar: SymbolToolbar) -> None:
        """Test renaming a symbol via context menu."""
        self.log("Testing rename symbol via context menu")
        toolbar.select_group(self.test_group_name)
        old_name = self.test_symbol_name
        new_name = f"Renamed Symbol {self.suffix}"
        toolbar.rename_symbol(old_name, new_name)
        toolbar.wait_for_symbol_hidden(old_name)
        assert toolbar.symbol_exists(new_name), (
            f"Symbol '{new_name}' should exist after rename"
        )
        self.test_symbol_name = new_name

    def test_rename_symbol_via_editor(self, toolbar: SymbolToolbar) -> None:
        """Test renaming a symbol via the symbol editor."""
        self.log("Testing rename symbol via editor")
        toolbar.select_group(self.test_group_name)
        old_name = self.test_symbol_name
        new_name = f"Editor Renamed {self.suffix}"
        editor = toolbar.edit_symbol(old_name)
        editor.set_name(new_name)
        editor.save()
        toolbar.wait_for_symbol_hidden(old_name)
        assert toolbar.symbol_exists(new_name), (
            f"Symbol '{new_name}' should exist after editor rename"
        )
        self.test_symbol_name = new_name

    def test_custom_symbol_as_actuator(self) -> None:
        """Test pressing the custom symbol actuator to control a channel."""
        self.log("Testing custom symbol as actuator")
        assert self.custom_symbol is not None, "Custom symbol should exist"

        self.log("Pressing custom symbol actuator (expecting 1)")
        self.custom_symbol.press()
        self.wait_for_eq(self.cmd_channel_name, 1)
        self.log("Received expected value: 1")

        self.log("Pressing custom symbol actuator again (expecting 0)")
        self.custom_symbol.press()
        self.wait_for_eq(self.cmd_channel_name, 0)
        self.log("Received expected value: 0")

    def test_export_symbol(self, toolbar: SymbolToolbar) -> None:
        """Test exporting a symbol via context menu."""
        self.log("Testing export symbol")
        assert self.custom_symbol is not None
        self.custom_symbol._enable_edit_mode()
        toolbar.select_group(self.test_group_name)
        exported = toolbar.export_symbol(self.test_symbol_name)

        assert "key" in exported, "Exported symbol should contain 'key'"
        assert "name" in exported, "Exported symbol should contain 'name'"
        assert "data" in exported, "Exported symbol should contain 'data'"
        assert exported["name"] == self.test_symbol_name, (
            f"Exported symbol name should be '{self.test_symbol_name}'"
        )

    def test_delete_symbol(self, toolbar: SymbolToolbar) -> None:
        """Test deleting a symbol via context menu."""
        self.log("Testing delete symbol")
        toolbar.select_group(self.test_group_name)
        toolbar.delete_symbol(self.test_symbol_name)

    def test_delete_symbol_group(self, toolbar: SymbolToolbar) -> None:
        """Test deleting a symbol group via context menu."""
        self.log("Testing delete symbol group")
        toolbar.delete_group(self.test_group_name)

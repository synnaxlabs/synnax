#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Page
from pydantic import JsonValue

import synnax as sy
from console.case import ConsoleCase
from console.schematic import SCHEMATIC_VERSION, Button
from console.schematic.off_page_reference import OffPageReference
from console.schematic.schematic import Schematic
from framework.utils import assert_envelope, assert_link_format
from x import random_name


def assert_exported_json(exported: dict[str, JsonValue]) -> None:
    """Assert that the exported JSON has a valid structure.

    Validates:
    - Envelope shape: 'schematic' type, version >= SCHEMATIC_VERSION, key stripped
    - Required keys exist: nodes, edges, configs
    - The body parses as a generated schematic
    """
    assert_envelope(exported, envelope_type="schematic", min_version=SCHEMATIC_VERSION)
    required_keys = ["nodes", "edges", "configs"]
    for key in required_keys:
        assert key in exported, f"Exported JSON should contain '{key}'"
    sy.schematic.Schematic.model_validate(exported)


class SchematicLifecycle(ConsoleCase):
    """Test schematic lifecycle operations."""

    suffix: str
    idx_name: str
    cmd_name: str
    ctx_schematic_name: str | None
    ctx_schematic_copy_name: str | None
    main_schematic_name: str | None
    main_schematic_link: str
    off_page_schematic_names: list[str]

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()
        self.ctx_schematic_name = None
        self.ctx_schematic_copy_name = None
        self.main_schematic_name = None
        self.off_page_schematic_names = []

        ctx_schematic = self.console.pages.create(
            Schematic, f"Context Menu Test {self.suffix}"
        )
        self.ctx_schematic_name = ctx_schematic.page_name
        ctx_schematic.close()

    def teardown(self) -> None:
        for name in [
            self.ctx_schematic_name,
            self.ctx_schematic_copy_name,
            self.main_schematic_name,
            *self.off_page_schematic_names,
        ]:
            if name:
                self._cleanup_pages.append(name)

        super().teardown()

    def setup_channels(self) -> None:
        """Create all test channels."""
        self.idx_name = f"schematic_test_idx_{self.suffix}"
        self.cmd_name = f"schematic_test_cmd_{self.suffix}"

        index_ch = self.client.channels.create(
            name=self.idx_name,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=self.cmd_name,
            data_type=sy.DataType.UINT8,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        """Run all schematic lifecycle tests."""
        self.setup_channels()

        schematic = self.console.pages.create(
            Schematic, f"Schematic Test {self.suffix}"
        )
        schematic.create_symbol(Button(label="Test Button", channel_name=self.cmd_name))

        self.test_view_writers_in_control(schematic)
        self.test_copy_link(schematic)
        self.test_export_json(schematic)
        self.test_undo_redo(schematic)

        self.main_schematic_link = schematic.copy_link()
        self.main_schematic_name = schematic.page_name
        schematic.close()
        assert not schematic.is_open, "Schematic should be closed"

        self.test_open_schematic_from_resources()
        self.test_drag_schematic_onto_mosaic()
        self.test_open_schematic_from_search()

        self.test_ctx_operations()
        self.test_multi_schematic_operations()
        self.test_off_page_navigation()
        self.test_ctx_delete_operations()

        self.client.channels.delete([self.cmd_name, self.idx_name])

    def test_view_writers_in_control(self, schematic: Schematic) -> None:
        """Test that the control legend shows active writers."""
        self.log("Testing view writers in control")

        schematic.acquire_control()
        assert schematic.get_control_status() is True, (
            "Control status mismatch! Expected: True"
        )

        schematic.set_properties(show_control_legend=True)
        assert schematic.control_legend_visible() is True, (
            "Control legend should be visible"
        )

        entries = schematic.get_control_legend_entries()
        assert len(entries) > 0, "Control legend should have at least one entry"
        assert any(schematic.page_name in e for e in entries), (
            f"Expected writer '{schematic.page_name}' in control legend!\n"
            f"Actual entries: {entries}"
        )

        schematic.release_control()
        assert schematic.get_control_status() is False, (
            "Control status mismatch! Expected: False"
        )

    def test_copy_link(self, schematic: Schematic) -> None:
        """Test copying a link to the schematic via toolbar button."""
        self.log("Testing copy link to schematic")

        link = schematic.copy_link()

        assert_link_format(link, resource_type="schematic")

    def test_export_json(self, schematic: Schematic) -> None:
        """Test exporting the schematic as a JSON file via toolbar button."""
        self.log("Testing export schematic as JSON")

        exported = schematic.export_json()
        assert_exported_json(exported)

    def test_undo_redo(self, schematic: Schematic) -> None:
        """Test undo/redo of a node deletion via keyboard shortcuts."""
        self.log("Testing schematic undo/redo")

        schematic.enable_edit()
        schematic._click_canvas()
        baseline = schematic.get_symbol_count()
        assert baseline > 0, (
            f"Expected at least one symbol on the schematic before undo/redo, "
            f"got {baseline}"
        )

        button = schematic.find_symbol(
            Button(label=self.cmd_name, channel_name=self.cmd_name)
        )
        button.delete()
        after_delete = schematic.get_symbol_count()
        assert after_delete == baseline - 1, (
            f"Expected {baseline - 1} symbols after delete, got {after_delete}"
        )

        schematic._click_canvas()
        self.console.layout.press_key("ControlOrMeta+Z")
        schematic.wait_for_symbol_count(baseline)

        self.console.layout.press_key("ControlOrMeta+Shift+Z")
        schematic.wait_for_symbol_count(baseline - 1)

        schematic._click_canvas()
        self.console.layout.press_key("ControlOrMeta+Z")
        schematic.wait_for_symbol_count(baseline)

    def test_open_schematic_from_resources(self) -> None:
        """Test opening a schematic by double-clicking it in the project resources
        toolbar.
        """
        assert self.main_schematic_name is not None
        self.log("Testing open schematic from resources toolbar")

        schematic = self.console.pages.open(Schematic, self.main_schematic_name)

        assert schematic.is_pane_visible, (
            f"Schematic '{self.main_schematic_name}' pane not visible after opening "
            "from resources toolbar"
        )

        # Assert the link we used to open is the same link we get.
        opened_link = schematic.copy_link()
        assert opened_link == self.main_schematic_link, (
            f"Opened schematic link should match: expected {self.main_schematic_link}, "
            f"got {opened_link}"
        )

        schematic.close()

    def test_drag_schematic_onto_mosaic(self) -> None:
        """Test dragging a schematic from the resources toolbar onto the mosaic."""
        assert self.main_schematic_name is not None
        self.log("Testing drag schematic onto mosaic")

        schematic = self.console.pages.drag_to_mosaic(
            Schematic, self.main_schematic_name
        )

        assert schematic.is_pane_visible, (
            f"Schematic '{self.main_schematic_name}' pane not visible after dragging "
            "to mosaic"
        )

        opened_link = schematic.copy_link()
        assert opened_link == self.main_schematic_link, (
            f"Opened schematic link should match: expected {self.main_schematic_link}, "
            f"got {opened_link}"
        )

        schematic.close()

    def test_open_schematic_from_search(self) -> None:
        """Test opening a schematic by searching its name in the command palette."""
        assert self.main_schematic_name is not None
        self.log("Testing open schematic from search palette")

        schematic = self.console.pages.open_from_search(
            Schematic, self.main_schematic_name
        )

        assert schematic.is_pane_visible, (
            f"Schematic '{self.main_schematic_name}' pane not visible after opening "
            "from search"
        )

        opened_link = schematic.copy_link()
        assert opened_link == self.main_schematic_link, (
            f"Opened schematic link should match: expected {self.main_schematic_link}, "
            f"got {opened_link}"
        )

        schematic.close()

    def test_ctx_operations(self) -> None:
        """Test context menu operations using shared schematic."""
        assert self.ctx_schematic_name is not None, "ctx_schematic_name should be set"

        self.log("Testing copy link via context menu")
        link = self.console.pages.copy_link(self.ctx_schematic_name)
        assert_link_format(link, resource_type="schematic")

        self.log("Testing export schematic via context menu")
        exported = self.console.pages.export(self.ctx_schematic_name)
        assert_exported_json(exported)

        self.log("Testing rename schematic via context menu")
        new_name = f"Renamed Schematic {self.suffix}"
        self.console.pages.rename(self.ctx_schematic_name, new_name)
        assert self.console.pages.exists(new_name), (
            f"Schematic '{new_name}' should exist after rename"
        )
        self.ctx_schematic_name = new_name

        self.log("Testing duplicate schematic via context menu")
        copy_name = f"Copied Schematic {self.suffix}"
        self.console.pages.duplicate(self.ctx_schematic_name, copy_name)
        assert self.console.pages.exists(copy_name), (
            f"Copied schematic '{copy_name}' should exist"
        )
        assert self.console.pages.exists(self.ctx_schematic_name), (
            f"Original schematic '{self.ctx_schematic_name}' should still exist"
        )
        self.ctx_schematic_copy_name = copy_name

    def test_multi_schematic_operations(self) -> None:
        """Test multi-select operations: duplicate multiple, then delete multiple."""
        self.log("Creating schematics for multi-select operations")
        schematic_names: list[str] = []
        for i in range(3):
            schematic = self.console.pages.create(
                Schematic, f"Multi Test {i} {self.suffix}"
            )
            schematic_names.append(schematic.page_name)
            schematic.close()

        self.log("Testing duplicate multiple schematics via context menu")
        self.console.pages.duplicate_many(schematic_names)

        copy_names = [f"{name} (copy)" for name in schematic_names]
        for copy_name in copy_names:
            assert self.console.pages.exists(copy_name), (
                f"Copied schematic '{copy_name}' should exist"
            )

        self.log("Testing delete multiple schematics via context menu")
        all_names = schematic_names + copy_names
        self.console.pages.delete_many(all_names)

    def _dblclick_node(self, page: Page, label: str) -> None:
        """Double-click a react-flow node by label using mouse coordinates."""
        pane = page.locator(".react-flow__pane").first
        pane.click()
        node = page.locator(".react-flow__node").filter(has_text=label).first
        node.wait_for(state="visible", timeout=5000)
        box = node.bounding_box()
        if box is None:
            raise RuntimeError(f"Could not get bounding box for node '{label}'")
        cx = box["x"] + box["width"] / 2
        cy = box["y"] + box["height"] / 2
        page.mouse.dblclick(cx, cy)

    def test_off_page_navigation(self) -> None:
        """Test off-page reference navigation cycling through 3 linked schematics."""
        self.log("Testing off-page reference navigation cycle")
        page = self.console.layout.page

        names = [
            f"snap {self.suffix}",
            f"crackle {self.suffix}",
            f"pop {self.suffix}",
        ]

        for name in names:
            s = self.console.pages.create(Schematic, name)
            s.close()
        self.off_page_schematic_names = list(names)

        # Open snap, create off-page ref pointing to crackle, navigate
        schematic = self.console.pages.open(Schematic, names[0])
        ref = OffPageReference(label=f"Go to {names[1]}", page_name=names[1])
        schematic.create_symbol(ref)
        schematic.disable_edit()
        schematic.fit_view()
        self._dblclick_node(page, f"Go to {names[1]}")

        # Arrive at crackle, assert navigation succeeded
        self.console.layout.wait_for_tab(names[1])
        assert self.console.layout.get_tab(names[1]).is_visible(), (
            f"Expected to navigate to '{names[1]}'"
        )
        self.console.layout.get_tab(names[1]).click()
        schematic = Schematic.from_open_page(
            layout=self.console.layout, client=self.client, name=names[1]
        )
        ref2 = OffPageReference(label=f"Go to {names[2]}", page_name=names[2])
        schematic.create_symbol(ref2)
        schematic.disable_edit()
        schematic.fit_view()
        self._dblclick_node(page, f"Go to {names[2]}")

        # Arrive at pop, assert navigation succeeded
        self.console.layout.wait_for_tab(names[2])
        assert self.console.layout.get_tab(names[2]).is_visible(), (
            f"Expected to navigate to '{names[2]}'"
        )
        self.console.layout.get_tab(names[2]).click()
        schematic = Schematic.from_open_page(
            layout=self.console.layout, client=self.client, name=names[2]
        )
        ref3 = OffPageReference(label=f"Go to {names[0]}", page_name=names[0])
        schematic.create_symbol(ref3)
        schematic.disable_edit()
        schematic.fit_view()
        self._dblclick_node(page, f"Go to {names[0]}")

        # Verify we navigated back to snap
        self.console.layout.wait_for_tab(names[0])
        assert self.console.layout.get_tab(names[0]).is_visible(), (
            f"Expected to navigate back to '{names[0]}'"
        )
        self.log("Off-page navigation cycle verified")

        # Each off-page hop opens a new tab, so the cycle leaves one tab per schematic.
        for name in names:
            self.console.layout.close_tab(name)

        self.console.pages.delete_many(names)
        self.off_page_schematic_names = []

    def test_ctx_delete_operations(self) -> None:
        """Test delete operations using remaining schematics from earlier tests."""
        assert self.ctx_schematic_name is not None, "ctx_schematic_name should be set"

        self.log("Testing delete schematic via context menu")
        self.console.pages.delete(self.ctx_schematic_name)
        self.ctx_schematic_name = None

        self.log("Testing delete copied schematic via context menu")
        if self.ctx_schematic_copy_name:
            self.console.pages.delete(self.ctx_schematic_copy_name)
            self.ctx_schematic_copy_name = None

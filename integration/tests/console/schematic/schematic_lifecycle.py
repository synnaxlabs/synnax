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
from console.schematic import Button
from console.schematic.schematic import Schematic
from framework.utils import assert_link_format, get_random_name


class SchematicLifecycle(ConsoleCase):
    """Test schematic lifecycle operations."""

    suffix: str
    idx_name: str
    cmd_name: str

    def setup_channels(self) -> None:
        """Create all test channels."""
        self.suffix = get_random_name()
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

        schematic = Schematic(
            self.client, self.console, f"Schematic Test {self.suffix}"
        )
        schematic.create_symbol(Button(label="Test Button", channel_name=self.cmd_name))

        self.test_view_writers_in_control(schematic)
        self.test_copy_link(schematic)
        self.test_export_json(schematic)

        schematic_link = schematic.copy_link()
        schematic_name = schematic.page_name
        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

        self.test_open_schematic_from_resources(schematic_name, schematic_link)
        self.test_drag_schematic_onto_mosaic(schematic_name, schematic_link)

        self.test_open_schematic_from_search(schematic_name, schematic_link)

        # Resources Toolbar > Context Menu
        self.test_ctx_copy_link()
        self.test_ctx_export_json()
        self.test_ctx_rename_schematic()
        self.test_ctx_copy_schematic()
        self.test_ctx_copy_multiple_schematics()
        self.test_ctx_snapshot_schematic()
        self.test_ctx_snapshot_multiple_schematics()
        self.test_snapshot_rename_synchronization()
        self.test_ctx_delete_schematic()
        self.test_ctx_delete_multiple_schematics()

        self.client.channels.delete([self.cmd_name, self.idx_name])

    def test_view_writers_in_control(self, schematic: Schematic) -> None:
        """Test that the control legend shows active writers."""
        self.log("Testing view writers in control")

        schematic.acquire_control()
        schematic.assert_control_status(True)

        schematic.set_properties(show_control_legend=True)
        schematic.assert_control_legend_visible(True)

        entries = schematic.get_control_legend_entries()
        assert len(entries) > 0, "Control legend should have at least one entry"

        schematic.release_control()
        schematic.assert_control_status(False)

    def test_copy_link(self, schematic: Schematic) -> None:
        """Test copying a link to the schematic via toolbar button."""
        self.log("Testing copy link to schematic")

        link = schematic.copy_link()

        assert_link_format(link, "schematic")

    def test_export_json(self, schematic: Schematic) -> None:
        """Test exporting the schematic as a JSON file via toolbar button."""
        self.log("Testing export schematic as JSON")

        exported = schematic.export_json()

        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Schematic key should be a UUID"

    def test_open_schematic_from_resources(
        self, schematic_name: str, expected_link: str
    ) -> None:
        """Test opening a schematic by double-clicking it in the workspace resources toolbar."""
        self.log("Testing open schematic from resources toolbar")

        schematic = self.console.workspace.open_schematic(self.client, schematic_name)

        assert schematic.is_pane_visible(), "Schematic pane should be visible"

        opened_link = schematic.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened schematic link should match: expected {expected_link}, got {opened_link}"

        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

    def test_drag_schematic_onto_mosaic(
        self, schematic_name: str, expected_link: str
    ) -> None:
        """Test dragging a schematic from the resources toolbar onto the mosaic."""
        self.log("Testing drag schematic onto mosaic")

        schematic = self.console.workspace.drag_schematic_to_mosaic(
            self.client, schematic_name
        )

        assert schematic.is_pane_visible(), "Schematic pane should be visible"

        opened_link = schematic.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened schematic link should match: expected {expected_link}, got {opened_link}"

        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

    def test_open_schematic_from_search(
        self, schematic_name: str, expected_link: str
    ) -> None:
        """Test opening a schematic by searching its name in the command palette."""
        self.log("Testing open schematic from search palette")

        schematic = Schematic.open_from_search(
            self.client, self.console, schematic_name
        )

        assert schematic.is_pane_visible(), "Schematic pane should be visible"

        opened_link = schematic.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened schematic link should match: expected {expected_link}, got {opened_link}"

        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

    def test_ctx_copy_link(self) -> None:
        """Test copying a link to a schematic via context menu."""
        self.log("Testing copy link via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        schematic = Schematic(self.client, self.console, f"Copy Link Test {suffix}")
        schematic_name = schematic.page_name
        expected_link = schematic.copy_link()
        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

        link = self.console.workspace.copy_page_link(schematic_name)

        assert (
            link == expected_link
        ), f"Context menu link should match: expected {expected_link}, got {link}"

        self.console.workspace.delete_page(schematic_name)

    def test_ctx_export_json(self) -> None:
        """Test exporting a schematic as JSON via context menu."""
        self.log("Testing export schematic via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        schematic = Schematic(self.client, self.console, f"Export Test {suffix}")
        schematic_name = schematic.page_name
        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

        exported = self.console.workspace.export_page(schematic_name)

        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Schematic key should be a UUID"

        self.console.workspace.delete_page(schematic_name)

    def test_ctx_rename_schematic(self) -> None:
        """Test renaming a schematic via context menu in the workspace resources toolbar."""
        self.log("Testing rename schematic via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        schematic = Schematic(self.client, self.console, f"Rename Test {suffix}")
        old_name = schematic.page_name
        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

        new_name = f"Renamed Schematic {suffix}"
        self.console.workspace.rename_page(old_name, new_name)

        assert self.console.workspace.page_exists(
            new_name
        ), f"Schematic '{new_name}' should exist after rename"

        self.console.workspace.delete_page(new_name)

    def test_ctx_copy_schematic(self) -> None:
        """Test making a copy of a schematic via context menu."""
        self.log("Testing copy schematic via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        schematic = Schematic(self.client, self.console, f"Copy Source {suffix}")
        original_name = schematic.page_name
        schematic.close()

        copy_name = f"Copied Schematic {suffix}"
        self.console.workspace.copy_page(original_name, copy_name)

        assert self.console.workspace.page_exists(
            copy_name
        ), f"Copied schematic '{copy_name}' should exist"
        assert self.console.workspace.page_exists(
            original_name
        ), f"Original schematic '{original_name}' should still exist"

        self.console.workspace.delete_page(original_name)
        self.console.workspace.delete_page(copy_name)

    def test_ctx_copy_multiple_schematics(self) -> None:
        """Test copying multiple schematics via context menu."""
        self.log("Testing copy multiple schematics via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        schematic_names: list[str] = []

        for i in range(2):
            schematic = Schematic(self.client, self.console, f"Multi Copy {i} {suffix}")
            schematic_names.append(schematic.page_name)
            schematic.close()

        self.console.workspace.copy_pages(schematic_names)

        for name in schematic_names:
            copy_name = f"{name} (copy)"
            assert self.console.workspace.page_exists(
                copy_name
            ), f"Copied schematic '{copy_name}' should exist"

        all_names = schematic_names + [f"{name} (copy)" for name in schematic_names]
        self.console.workspace.delete_pages(all_names)

    def test_ctx_snapshot_schematic(self) -> None:
        """Test snapshotting a schematic to the active range via context menu."""
        self.log("Testing snapshot schematic to active range via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        range_name = f"Snapshot Range {suffix}"
        self.console.ranges.create(range_name, persisted=True)
        self.console.ranges.favorite(range_name)
        self.console.ranges.set_active(range_name)

        schematic = Schematic(self.client, self.console, f"Snapshot Source {suffix}")
        schematic_name = schematic.page_name
        schematic.close()

        self.console.workspace.snapshot_page_to_active_range(schematic_name, range_name)

        self.console.workspace.delete_page(schematic_name)
        self.console.ranges.open_explorer()
        self.console.ranges.delete_from_explorer(range_name)

    def test_ctx_snapshot_multiple_schematics(self) -> None:
        """Test snapshotting multiple schematics to the active range via context menu."""
        self.log(
            "Testing snapshot multiple schematics to active range via context menu"
        )
        self.console.close_nav_drawer()

        suffix = get_random_name()
        range_name = f"Multi Snapshot Range {suffix}"
        self.console.ranges.create(range_name, persisted=True)
        self.console.ranges.favorite(range_name)
        self.console.ranges.set_active(range_name)

        schematic_names: list[str] = []
        for i in range(2):
            schematic = Schematic(
                self.client, self.console, f"Multi Snapshot {i} {suffix}"
            )
            schematic_names.append(schematic.page_name)
            schematic.close()

        self.console.workspace.snapshot_pages_to_active_range(
            schematic_names, range_name
        )

        self.console.workspace.delete_pages(schematic_names)
        self.console.ranges.open_explorer()
        self.console.ranges.delete_from_explorer(range_name)

    def test_snapshot_rename_synchronization(self) -> None:
        """Test that renaming a schematic snapshot synchronizes across UI elements.

        Verifies synchronization across:
        - Mosaic Tab
        - Resources Toolbar
        - Visualization Toolbar
        - Range Details Overview
        """
        self.log("Testing schematic snapshot rename synchronization")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        range_name = f"Rename Sync Range {suffix}"
        self.console.ranges.create(range_name, persisted=True)
        self.console.ranges.favorite(range_name)
        self.console.ranges.set_active(range_name)

        original_schematic_name = f"Snapshot Original {suffix}"
        schematic = Schematic(self.client, self.console, original_schematic_name)
        schematic.close()
        self.console.workspace.snapshot_page_to_active_range(
            original_schematic_name, range_name
        )

        # Snapshot API automatically appends "(Snapshot)" suffix to the name
        snapshot_name = f"{original_schematic_name} (Snapshot)"

        self.console.ranges.open_from_search(range_name)
        self.console.ranges.wait_for_overview(range_name)

        snapshot_names = self.console.ranges.get_snapshot_names_in_overview()
        self.log(f"Snapshots found in overview: {snapshot_names}")
        assert self.console.ranges.snapshot_exists_in_overview(
            snapshot_name, timeout=10000
        ), f"Snapshot '{snapshot_name}' should exist in Range Details Overview. Found: {snapshot_names}"

        self.console.ranges.open_snapshot_from_overview(snapshot_name)
        self.console.layout.wait_for_tab(snapshot_name)

        new_name = f"Snapshot Renamed {suffix}"
        self.console.layout.rename_tab(old_name=snapshot_name, new_name=new_name)

        self.log("Verifying Mosaic Tab")
        self.console.ESCAPE
        self.console.layout.wait_for_tab(new_name)

        self.log("Verifying Visualization Toolbar")
        self.console.layout.show_visualization_toolbar()
        toolbar_title = self.console.layout.get_visualization_toolbar_title()
        assert (
            toolbar_title == new_name
        ), f"Visualization Toolbar should show '{new_name}', got '{toolbar_title}'"
        self.console.layout.hide_visualization_toolbar()

        self.log("Verifying Range Details Overview")
        self.console.ranges.open_from_search(range_name)
        self.console.ranges.wait_for_overview(range_name)
        assert self.console.ranges.snapshot_exists_in_overview(
            new_name
        ), f"Snapshot should be renamed to '{new_name}' in Range Details Overview"
        self.console.close_page(new_name)
        self.console.workspace.delete_page(original_schematic_name)
        self.console.ranges.open_explorer()
        self.console.ranges.delete_from_explorer(range_name)

    def test_ctx_delete_schematic(self) -> None:
        """Test deleting a schematic via context menu in the workspace resources toolbar."""
        self.log("Testing delete schematic via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        schematic = Schematic(self.client, self.console, f"Delete Test {suffix}")
        schematic_name = schematic.page_name
        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed after close()"

        assert self.console.workspace.page_exists(
            schematic_name
        ), f"Schematic '{schematic_name}' should exist before deletion"

        self.console.workspace.delete_page(schematic_name)

    def test_ctx_delete_multiple_schematics(self) -> None:
        """Test deleting multiple schematics via context menu."""
        self.log("Testing delete multiple schematics via context menu")
        self.console.close_nav_drawer()

        suffix = get_random_name()
        schematic_names: list[str] = []

        for i in range(3):
            schematic = Schematic(
                self.client, self.console, f"Multi Delete {i} {suffix}"
            )
            schematic_names.append(schematic.page_name)
            schematic.close()

        for name in schematic_names:
            assert self.console.workspace.page_exists(
                name
            ), f"Schematic '{name}' should exist before deletion"

        self.console.workspace.delete_pages(schematic_names)

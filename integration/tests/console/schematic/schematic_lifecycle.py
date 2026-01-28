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
    shared_range_name: str
    ctx_schematic_name: str | None
    ctx_schematic_copy_name: str | None
    main_schematic_name: str
    main_schematic_link: str

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.ctx_schematic_name = None
        self.ctx_schematic_copy_name = None

        self.shared_range_name = f"Shared Snapshot Range {self.suffix}"
        self.console.ranges.create(self.shared_range_name, persisted=True)
        self.console.ranges.favorite(self.shared_range_name)
        self.console.ranges.set_active(self.shared_range_name)

        ctx_schematic = Schematic(
            self.client, self.console, f"Context Menu Test {self.suffix}"
        )
        self.ctx_schematic_name = ctx_schematic.page_name
        ctx_schematic.close()

    def teardown(self) -> None:
        if hasattr(self, "shared_range_name"):
            self.console.ranges.open_explorer()
            if self.console.ranges.exists_in_explorer(self.shared_range_name):
                self.console.ranges.delete_from_explorer(self.shared_range_name)

        names_to_cleanup = [
            getattr(self, "ctx_schematic_name", None),
            getattr(self, "ctx_schematic_copy_name", None),
            getattr(self, "main_schematic_name", None),
        ]
        for name in names_to_cleanup:
            if name and self.console.workspace.page_exists(name):
                self.console.workspace.delete_page(name)

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

        schematic = Schematic(
            self.client, self.console, f"Schematic Test {self.suffix}"
        )
        schematic.create_symbol(Button(label="Test Button", channel_name=self.cmd_name))

        self.test_view_writers_in_control(schematic)
        self.test_copy_link(schematic)
        self.test_export_json(schematic)

        self.main_schematic_link = schematic.copy_link()
        self.main_schematic_name = schematic.page_name
        schematic.close()
        assert not schematic.is_open(), "Schematic should be closed"

        self.test_open_schematic_from_resources()
        self.test_drag_schematic_onto_mosaic()
        self.test_open_schematic_from_search()

        self.test_ctx_operations()
        self.test_multi_schematic_operations()
        self.test_snapshot_operations()
        self.test_ctx_delete_operations()

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

    def test_open_schematic_from_resources(self) -> None:
        """Test opening a schematic by double-clicking it in the workspace resources toolbar."""
        self.log("Testing open schematic from resources toolbar")

        schematic = self.console.workspace.open_schematic(
            self.client, self.main_schematic_name
        )

        assert schematic.is_pane_visible(), "Schematic pane should be visible"

        opened_link = schematic.copy_link()
        assert (
            opened_link == self.main_schematic_link
        ), f"Opened schematic link should match: expected {self.main_schematic_link}, got {opened_link}"

        schematic.close()

    def test_drag_schematic_onto_mosaic(self) -> None:
        """Test dragging a schematic from the resources toolbar onto the mosaic."""
        self.log("Testing drag schematic onto mosaic")

        schematic = self.console.workspace.drag_schematic_to_mosaic(
            self.client, self.main_schematic_name
        )

        assert schematic.is_pane_visible(), "Schematic pane should be visible"

        opened_link = schematic.copy_link()
        assert (
            opened_link == self.main_schematic_link
        ), f"Opened schematic link should match: expected {self.main_schematic_link}, got {opened_link}"

        schematic.close()

    def test_open_schematic_from_search(self) -> None:
        """Test opening a schematic by searching its name in the command palette."""
        self.log("Testing open schematic from search palette")

        schematic = Schematic.open_from_search(
            self.client, self.console, self.main_schematic_name
        )

        assert schematic.is_pane_visible(), "Schematic pane should be visible"

        opened_link = schematic.copy_link()
        assert (
            opened_link == self.main_schematic_link
        ), f"Opened schematic link should match: expected {self.main_schematic_link}, got {opened_link}"

        schematic.close()

    def test_ctx_operations(self) -> None:
        """Test context menu operations using shared schematic."""
        self.console.close_nav_drawer()

        self.log("Testing copy link via context menu")
        link = self.console.workspace.copy_page_link(self.ctx_schematic_name)
        assert_link_format(link, "schematic")

        self.log("Testing export schematic via context menu")
        exported = self.console.workspace.export_page(self.ctx_schematic_name)
        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Schematic key should be a UUID"

        self.log("Testing rename schematic via context menu")
        new_name = f"Renamed Schematic {self.suffix}"
        self.console.workspace.rename_page(self.ctx_schematic_name, new_name)
        assert self.console.workspace.page_exists(
            new_name
        ), f"Schematic '{new_name}' should exist after rename"
        self.ctx_schematic_name = new_name

        self.log("Testing copy schematic via context menu")
        copy_name = f"Copied Schematic {self.suffix}"
        self.console.workspace.copy_page(self.ctx_schematic_name, copy_name)
        assert self.console.workspace.page_exists(
            copy_name
        ), f"Copied schematic '{copy_name}' should exist"
        assert self.console.workspace.page_exists(
            self.ctx_schematic_name
        ), f"Original schematic '{self.ctx_schematic_name}' should still exist"
        self.ctx_schematic_copy_name = copy_name

    def test_multi_schematic_operations(self) -> None:
        """Test multi-select operations: copy multiple, then delete multiple."""
        self.console.close_nav_drawer()

        self.log("Creating schematics for multi-select operations")
        schematic_names: list[str] = []
        for i in range(3):
            schematic = Schematic(
                self.client, self.console, f"Multi Test {i} {self.suffix}"
            )
            schematic_names.append(schematic.page_name)
            schematic.close()

        self.log("Testing copy multiple schematics via context menu")
        self.console.workspace.copy_pages(schematic_names)

        copy_names = [f"{name} (copy)" for name in schematic_names]
        for copy_name in copy_names:
            assert self.console.workspace.page_exists(
                copy_name
            ), f"Copied schematic '{copy_name}' should exist"

        self.log("Testing delete multiple schematics via context menu")
        all_names = schematic_names + copy_names
        self.console.workspace.delete_pages(all_names)

    def test_snapshot_operations(self) -> None:
        """Test snapshot operations using shared range."""
        self.console.close_nav_drawer()

        self.log("Testing snapshot schematic to active range")
        single_snapshot_name = f"Snapshot Single {self.suffix}"
        schematic = Schematic(self.client, self.console, single_snapshot_name)
        schematic.close()
        self.console.workspace.snapshot_page_to_active_range(
            single_snapshot_name, self.shared_range_name
        )
        self.console.workspace.delete_page(single_snapshot_name)

        self.log("Testing snapshot multiple schematics to active range")
        multi_names: list[str] = []
        for i in range(2):
            schematic = Schematic(
                self.client, self.console, f"Snapshot Multi {i} {self.suffix}"
            )
            multi_names.append(schematic.page_name)
            schematic.close()
        self.console.workspace.snapshot_pages_to_active_range(
            multi_names, self.shared_range_name
        )
        self.console.workspace.delete_pages(multi_names)

        self.log("Testing schematic snapshot rename synchronization")
        self._test_snapshot_rename_synchronization()

    def _test_snapshot_rename_synchronization(self) -> None:
        """Test that renaming a snapshot synchronizes across UI elements."""
        original_name = f"Snapshot Original {self.suffix}"
        schematic = Schematic(self.client, self.console, original_name)
        schematic.close()
        self.console.workspace.snapshot_page_to_active_range(
            original_name, self.shared_range_name
        )

        # Snapshot API automatically appends "(Snapshot)" suffix to the name
        snapshot_name = f"{original_name} (Snapshot)"

        self.console.ranges.open_from_search(self.shared_range_name)
        self.console.ranges.wait_for_overview(self.shared_range_name)

        snapshot_names = self.console.ranges.get_snapshot_names_in_overview()
        self.log(f"Snapshots found in overview: {snapshot_names}")
        assert self.console.ranges.snapshot_exists_in_overview(
            snapshot_name, timeout=10000
        ), f"Snapshot '{snapshot_name}' should exist in Range Details Overview. Found: {snapshot_names}"

        self.console.ranges.open_snapshot_from_overview(snapshot_name)
        self.console.layout.wait_for_tab(snapshot_name)

        new_name = f"Snapshot Renamed {self.suffix}"
        self.console.layout.rename_tab(old_name=snapshot_name, new_name=new_name)

        self.console.ESCAPE
        self.console.layout.wait_for_tab(new_name)

        self.console.layout.show_visualization_toolbar()
        toolbar_title = self.console.layout.get_visualization_toolbar_title()
        assert (
            toolbar_title == new_name
        ), f"Visualization Toolbar should show '{new_name}', got '{toolbar_title}'"
        self.console.layout.hide_visualization_toolbar()

        self.console.ranges.open_from_search(self.shared_range_name)
        self.console.ranges.wait_for_overview(self.shared_range_name)
        assert self.console.ranges.snapshot_exists_in_overview(
            new_name
        ), f"Snapshot should be renamed to '{new_name}' in Range Details Overview"
        self.console.close_page(new_name)
        self.console.workspace.delete_page(original_name)

    def test_ctx_delete_operations(self) -> None:
        """Test delete operations using remaining schematics from earlier tests."""
        self.console.close_nav_drawer()

        self.log("Testing delete schematic via context menu")
        assert self.console.workspace.page_exists(
            self.ctx_schematic_name
        ), f"Schematic '{self.ctx_schematic_name}' should exist before deletion"
        self.console.workspace.delete_page(self.ctx_schematic_name)
        self.ctx_schematic_name = None

        self.log("Testing delete copied schematic via context menu")
        if self.ctx_schematic_copy_name:
            assert self.console.workspace.page_exists(
                self.ctx_schematic_copy_name
            ), f"Copied schematic '{self.ctx_schematic_copy_name}' should exist before deletion"
            self.console.workspace.delete_page(self.ctx_schematic_copy_name)
            self.ctx_schematic_copy_name = None

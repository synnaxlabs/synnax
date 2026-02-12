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
from framework.utils import get_random_name


class RangeDetails(ConsoleCase):
    """Test Range Details: child range context menu, metadata, and snapshot operations."""

    suffix: str
    parent_range_name: str
    child_1_name: str
    child_2_name: str
    child_3_name: str
    snapshot_name: str
    schematic_name: str

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.parent_range_name = f"DetailsParent_{self.suffix}"
        self.child_1_name = f"Child1_{self.suffix}"
        self.child_2_name = f"Child2_{self.suffix}"
        self.child_3_name = f"Child3_{self.suffix}"
        self.schematic_name = f"SnapshotSchematic_{self.suffix}"

        self.console.ranges.show_toolbar()

        self.console.ranges.create(self.parent_range_name, persisted=True)

        # Create a snapshot: make a schematic, favorite + activate the parent
        # range, snapshot the schematic to it, then clean up the original.
        schematic = self.console.workspace.create_schematic(self.schematic_name)
        schematic.close()
        self.console.ranges.favorite(self.parent_range_name)
        self.console.ranges.set_active(self.parent_range_name)
        self.console.workspace.snapshot_page_to_active_range(
            self.schematic_name, self.parent_range_name
        )
        self.console.workspace.delete_page(self.schematic_name)
        self.snapshot_name = f"{self.schematic_name} (Snapshot)"
        self.console.notifications.close_all()

        # Open parent overview BEFORE creating children. Child ranges show the
        # parent name in their explorer breadcrumb, so has_text filtering would
        # match children too if they already exist.
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.parent_range_name)
        self.console.ranges.wait_for_overview(self.parent_range_name)

        parent = self.client.ranges.retrieve(name=self.parent_range_name)
        now = sy.TimeStamp.now()
        for name in [self.child_1_name, self.child_2_name, self.child_3_name]:
            parent.create_child_range(
                name=name,
                time_range=sy.TimeRange(now - sy.TimeSpan.HOUR, now + sy.TimeSpan.HOUR),
            )

        for name in [self.child_1_name, self.child_2_name, self.child_3_name]:
            self.console.ranges.get_child_range_item(name).wait_for(
                state="visible", timeout=10000
            )

    def teardown(self) -> None:
        if self.console.layout.is_modal_open():
            self.page.keyboard.press("Escape")
        self.console.ranges.open_explorer()
        if self.console.ranges.exists_in_explorer(self.parent_range_name):
            self.console.ranges.delete_from_explorer(self.parent_range_name)
        super().teardown()

    def run(self) -> None:
        """Run all Range Details tests."""
        # Child Range Context Menu
        self.test_rename_child_range()
        self.test_favorite_multiple_child_ranges()
        self.test_unfavorite_multiple_child_ranges()
        self.test_copy_link_from_child_range()
        self.test_delete_child_range()
        self.test_delete_multiple_child_ranges()

        # Metadata
        self.test_set_metadata()
        self.test_update_metadata_value()
        self.test_copy_metadata_value()
        self.test_open_metadata_link()
        self.test_delete_metadata()

        # Snapshots
        self.test_navigate_to_snapshot()
        self.test_remove_snapshot()

    def test_rename_child_range(self) -> None:
        """Test renaming a child range via context menu."""
        self.log("Testing: Rename child range")
        new_name = f"RenamedChild1_{self.suffix}"
        self.console.ranges.rename_child_range(self.child_1_name, new_name)

        assert self.console.ranges.child_range_exists(
            new_name
        ), f"Renamed child range '{new_name}' should exist"

        rng = self.client.ranges.retrieve(name=new_name)
        assert rng.name == new_name, f"Server-side name should be '{new_name}'"
        self.child_1_name = new_name

    def test_favorite_multiple_child_ranges(self) -> None:
        """Test favoriting multiple child ranges via multi-select context menu."""
        self.log("Testing: Favorite multiple child ranges")
        self.console.ranges.favorite_child_ranges(
            [self.child_1_name, self.child_2_name]
        )

        assert self.console.ranges.exists_in_toolbar(
            self.child_1_name
        ), f"'{self.child_1_name}' should appear in toolbar after favoriting"
        assert self.console.ranges.exists_in_toolbar(
            self.child_2_name
        ), f"'{self.child_2_name}' should appear in toolbar after favoriting"

    def test_unfavorite_multiple_child_ranges(self) -> None:
        """Test unfavoriting multiple child ranges via multi-select context menu."""
        self.log("Testing: Unfavorite multiple child ranges")
        self._open_parent_overview()
        self.console.ranges.unfavorite_child_ranges(
            [self.child_1_name, self.child_2_name]
        )

        self.console.ranges.wait_for_removed_from_toolbar(self.child_1_name)
        self.console.ranges.wait_for_removed_from_toolbar(self.child_2_name)

    def test_copy_link_from_child_range(self) -> None:
        """Test copying a link to a child range via context menu."""
        self.log("Testing: Copy link from child range")
        self._open_parent_overview()
        self.console.ranges.copy_link_from_child_range(self.child_1_name)

        clipboard = self.console.layout.read_clipboard()
        assert len(clipboard) > 0, "Clipboard should not be empty after copying link"

    def test_delete_child_range(self) -> None:
        """Test deleting a single child range via context menu."""
        self.log("Testing: Delete child range")
        self._open_parent_overview()
        self.console.ranges.delete_child_range(self.child_3_name)

        self.console.ranges.wait_for_child_range_removed(self.child_3_name)

        try:
            self.client.ranges.retrieve(name=self.child_3_name)
            raise AssertionError(
                f"Range '{self.child_3_name}' should be deleted from server"
            )
        except sy.QueryError:
            pass

    def test_delete_multiple_child_ranges(self) -> None:
        """Test deleting multiple child ranges via multi-select context menu."""
        self.log("Testing: Delete multiple child ranges")
        self._open_parent_overview()
        self.console.ranges.delete_child_ranges([self.child_1_name, self.child_2_name])

        self.console.ranges.wait_for_child_range_removed(self.child_1_name)
        self.console.ranges.wait_for_child_range_removed(self.child_2_name)

        for name in [self.child_1_name, self.child_2_name]:
            try:
                self.client.ranges.retrieve(name=name)
                raise AssertionError(f"Range '{name}' should be deleted from server")
            except sy.QueryError:
                pass

    def test_set_metadata(self) -> None:
        """Test setting a new metadata key-value pair."""
        self.log("Testing: Set metadata")
        self._open_parent_overview()
        self.console.ranges.set_metadata("test_key", "test_value")

        assert self.console.ranges.metadata_exists(
            "test_key"
        ), "Metadata 'test_key' should exist"
        value = self.console.ranges.get_metadata_value("test_key")
        assert (
            value == "test_value"
        ), f"Metadata value should be 'test_value', got '{value}'"

    def test_update_metadata_value(self) -> None:
        """Test updating the value of existing metadata."""
        self.log("Testing: Update metadata value")
        self.console.ranges.update_metadata_value("test_key", "updated_value")

        value = self.console.ranges.get_metadata_value("test_key")
        assert (
            value == "updated_value"
        ), f"Metadata value should be 'updated_value', got '{value}'"

    def test_copy_metadata_value(self) -> None:
        """Test copying a metadata value to the clipboard."""
        self.log("Testing: Copy metadata value")
        self.console.ranges.copy_metadata_value("test_key")

        clipboard = self.console.layout.read_clipboard()
        assert (
            clipboard == "updated_value"
        ), f"Clipboard should contain 'updated_value', got '{clipboard}'"

    def test_open_metadata_link(self) -> None:
        """Test opening a URL metadata value in a new tab."""
        self.log("Testing: Open metadata link")
        self.console.ranges.update_metadata_value("test_key", "https://synnaxlabs.com")

        with self.page.expect_popup() as popup_info:
            self.console.ranges.open_metadata_link("test_key")

        popup = popup_info.value
        assert (
            "synnaxlabs.com" in popup.url
        ), f"Popup URL should contain 'synnaxlabs.com', got '{popup.url}'"
        popup.close()

    def test_delete_metadata(self) -> None:
        """Test deleting a metadata entry."""
        self.log("Testing: Delete metadata")
        self.console.ranges.delete_metadata("test_key")

        self.console.ranges.wait_for_metadata_removed("test_key")

    def test_navigate_to_snapshot(self) -> None:
        """Test navigating to a snapshot by clicking on it."""
        self.log("Testing: Navigate to snapshot")
        self._open_parent_overview()
        self.console.ranges.open_snapshot_from_overview(self.snapshot_name)
        self.console.layout.wait_for_tab(self.snapshot_name)
        self.console.layout.close_tab(self.snapshot_name)

    def test_remove_snapshot(self) -> None:
        """Test removing a snapshot from the range details page."""
        self.log("Testing: Remove snapshot")
        self._open_parent_overview()
        self.console.ranges.delete_snapshot_from_overview(self.snapshot_name)
        self.console.ranges.wait_for_snapshot_removed(self.snapshot_name)

    def _open_parent_overview(self) -> None:
        """Navigate to the parent range overview if not already showing."""
        if not self.console.ranges.is_overview_showing(self.parent_range_name):
            self.console.ranges.open_from_search(self.parent_range_name)

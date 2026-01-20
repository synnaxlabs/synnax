#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

from console.case import ConsoleCase


class RangeLifecycle(ConsoleCase):
    """Test the lifecycle of ranges in the Console UI."""

    def setup(self) -> None:
        super().setup()
        self.rand_suffix: int = random.randint(1000, 9999)

    def run(self) -> None:
        """Run all range lifecycle tests."""
        self.test_open_range_toolbar()
        self.test_create_label_for_range()
        self.test_create_local_range()
        self.test_create_persisted_range()
        self.test_create_range_with_stage()
        self.test_create_range_with_parent()
        self.test_create_range_with_labels()
        self.test_navigate_to_parent()
        self.test_open_range_explorer()
        self.test_range_exists_in_explorer()
        self.test_favorite_range()
        self.test_set_active_range()
        self.test_rename_range()
        self.test_delete_range()
        self.test_cleanup_ranges()

    def test_open_range_toolbar(self) -> None:
        """Test opening the ranges toolbar."""
        self.log("Testing: Open ranges toolbar")
        self.console.ranges.show_toolbar()
        toolbar_title = self.page.get_by_text("Ranges", exact=True).first
        assert toolbar_title.is_visible(), "Ranges toolbar should be visible"

    def test_create_local_range(self) -> None:
        """Test creating a local (non-persisted) range."""
        self.log("Testing: Create local range")
        local_range_name = f"LocalRange_{self.rand_suffix}"
        self.console.ranges.create(local_range_name, persisted=False)
        assert self.console.ranges.exists_in_toolbar(local_range_name), (
            "Local range should appear in toolbar"
        )
        self.local_range_name = local_range_name

    def test_create_persisted_range(self) -> None:
        """Test creating a persisted range."""
        self.log("Testing: Create persisted range")
        self.range_name = f"TestRange_{self.rand_suffix}"
        self.console.ranges.create(self.range_name, persisted=True)

    def test_create_range_with_stage(self) -> None:
        """Test creating a range with a specific stage."""
        self.log("Testing: Create range with stage")
        self.staged_range_name = f"CompletedRange_{self.rand_suffix}"
        self.console.ranges.create(
            self.staged_range_name, persisted=True, stage="Completed"
        )
        self.console.ranges.open_explorer()
        assert self.console.ranges.exists_in_explorer(self.staged_range_name), (
            "Staged range should exist in explorer"
        )

    def test_create_range_with_parent(self) -> None:
        """Test creating a range with a parent range."""
        self.log("Testing: Create range with parent")
        self.child_range_name = f"ChildRange_{self.rand_suffix}"
        self.console.ranges.create(
            self.child_range_name, persisted=True, parent=self.staged_range_name
        )
        self.console.ranges.open_explorer()
        assert self.console.ranges.exists_in_explorer(self.child_range_name), (
            "Child range should exist in explorer"
        )

    def test_navigate_to_parent(self) -> None:
        """Test navigating to parent range from child range overview."""
        self.log("Testing: Navigate to parent range")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.child_range_name)
        self.console.ranges.wait_for_overview(self.child_range_name)
        self.console.ranges.navigate_to_parent(self.staged_range_name)
        self.console.ranges.wait_for_overview(self.staged_range_name)
        assert self.console.ranges.is_overview_showing(self.staged_range_name), (
            "Should navigate to parent range overview"
        )

    def test_create_label_for_range(self) -> None:
        """Create a label to use when creating a range with labels."""
        self.log("Testing: Create label for range test")
        self.test_label_name = f"RangeLabel_{self.rand_suffix}"
        self.console.labels.open_edit_modal()
        self.console.labels.create(name=self.test_label_name)
        self.page.keyboard.press("Escape")
        self.console.labels.close_modal()

    def test_create_range_with_labels(self) -> None:
        """Test creating a range with labels attached."""
        self.log("Testing: Create range with labels")
        self.labeled_range_name = f"LabeledRange_{self.rand_suffix}"
        self.console.ranges.create(
            self.labeled_range_name, persisted=True, labels=[self.test_label_name]
        )
        self.console.ranges.open_explorer()
        assert self.console.ranges.exists_in_explorer(self.labeled_range_name), (
            "Labeled range should exist in explorer"
        )

    def test_open_range_explorer(self) -> None:
        """Test opening the Range Explorer."""
        self.log("Testing: Open Range Explorer")
        self.console.ranges.open_explorer()
        all_ranges_header = self.page.get_by_text("All Ranges")
        assert all_ranges_header.is_visible(), "Range Explorer should show 'All Ranges'"

    def test_range_exists_in_explorer(self) -> None:
        """Test that created range exists in the explorer."""
        self.log("Testing: Range exists in explorer")
        assert self.console.ranges.exists_in_explorer(self.range_name), (
            f"Range '{self.range_name}' should exist in explorer"
        )

    def test_favorite_range(self) -> None:
        """Test favoriting a range from the explorer."""
        self.log("Testing: Favorite range")
        self.console.ranges.favorite_from_explorer(self.range_name)
        assert self.console.ranges.exists_in_toolbar(self.range_name), (
            f"Range '{self.range_name}' should appear in toolbar after favoriting"
        )

    def test_set_active_range(self) -> None:
        """Test setting a range as active from the toolbar."""
        self.log("Testing: Set active range")
        self.console.ranges.set_active(self.range_name)
        item = self.console.ranges.get_toolbar_item(self.range_name)
        class_attr = item.get_attribute("class") or ""
        assert "pluto--selected" in class_attr, "Range should be marked as active"

    def test_rename_range(self) -> None:
        """Test renaming a range from the explorer."""
        self.log("Testing: Rename range")
        self.console.ranges.open_explorer()
        new_name = f"RenamedRange_{self.rand_suffix}"
        self.console.ranges.rename_from_explorer(self.range_name, new_name)
        assert self.console.ranges.exists_in_explorer(new_name), (
            "Range should exist with new name"
        )
        self.range_name = new_name

    def test_delete_range(self) -> None:
        """Test deleting a range from the explorer."""
        self.log("Testing: Delete range")
        self.console.ranges.delete_from_explorer(self.range_name)
        assert not self.console.ranges.exists_in_explorer(self.range_name), (
            "Range should be deleted"
        )

    def test_cleanup_ranges(self) -> None:
        """Clean up test ranges and labels."""
        self.log("Testing: Cleanup ranges")
        self.console.ranges.open_explorer()
        if self.console.ranges.exists_in_explorer(self.labeled_range_name):
            self.console.ranges.delete_from_explorer(self.labeled_range_name)
        if self.console.ranges.exists_in_explorer(self.child_range_name):
            self.console.ranges.delete_from_explorer(self.child_range_name)
        if self.console.ranges.exists_in_explorer(self.staged_range_name):
            self.console.ranges.delete_from_explorer(self.staged_range_name)
        self.console.labels.open_edit_modal()
        if self.test_label_name in self.console.labels.list_all():
            self.console.labels.delete(self.test_label_name)
        self.console.labels.close_modal()

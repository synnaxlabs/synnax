#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
from datetime import timezone

from console.case import ConsoleCase

import synnax as sy


class RangeLifecycle(ConsoleCase):
    """Test the lifecycle of ranges in the Console UI."""

    def setup(self) -> None:
        super().setup()
        self.rand_suffix: int = random.randint(1000, 9999)
        self.test_label_name = f"RangeLabel_{self.rand_suffix}"
        self.second_label_name = f"SecondLabel_{self.rand_suffix}"
        self.console.labels.create(name=self.test_label_name)
        self.console.labels.create(name=self.second_label_name)

    def teardown(self) -> None:
        ranges_to_delete = [
            getattr(self, "labeled_range_name", None),
            getattr(self, "new_child_range_name", None),
            getattr(self, "child_range_name", None),
            getattr(self, "staged_range_name", None),
        ]

        self.console.ranges.open_explorer()
        for range_name in ranges_to_delete:
            if range_name and self.console.ranges.exists_in_explorer(range_name):
                self.console.ranges.delete_from_explorer(range_name)

        if self.console.labels.exists(self.test_label_name):
            self.console.labels.delete(self.test_label_name)
        if self.console.labels.exists(self.second_label_name):
            self.console.labels.delete(self.second_label_name)

        super().teardown()

    def run(self) -> None:
        """Run all range lifecycle tests."""
        # Setup
        self.test_open_range_toolbar()

        # Range Creation
        self.test_create_local_range()
        self.test_create_persisted_range()
        self.test_create_range_with_stage()
        self.test_create_range_with_parent()
        self.test_create_range_with_labels()

        # Range Explorer
        self.test_open_range_explorer()
        self.test_range_exists_in_explorer()
        self.test_favorite_range()
        self.test_set_active_range()

        # Range Details
        self.test_navigate_to_parent()
        self.test_change_times_in_overview()
        self.test_change_stage_in_overview()
        self.test_add_label_in_overview()
        self.test_remove_label_in_overview()
        self.test_rename_range_from_tab()
        self.test_rename_range_from_overview()
        self.test_copy_python_code()
        self.test_copy_typescript_code()
        self.test_copy_link()
        self.test_download_csv()

        # Range Explorer Context Menu
        self.test_rename_range()
        self.test_delete_range()

        # Child Ranges
        self.test_navigate_to_child_range()
        self.test_create_child_range_from_overview()
        self.test_change_child_range_stage()
        self.test_favorite_child_range()
        self.test_unfavorite_child_range()

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
        assert self.console.ranges.exists_in_toolbar(
            local_range_name
        ), "Local range should appear in toolbar"
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
        assert self.console.ranges.exists_in_explorer(
            self.staged_range_name
        ), "Staged range should exist in explorer"

    def test_create_range_with_parent(self) -> None:
        """Test creating a range with a parent range."""
        self.log("Testing: Create range with parent")
        self.child_range_name = f"ChildRange_{self.rand_suffix}"
        self.console.ranges.create(
            self.child_range_name, persisted=True, parent=self.staged_range_name
        )
        self.console.ranges.open_explorer()
        assert self.console.ranges.exists_in_explorer(
            self.child_range_name
        ), "Child range should exist in explorer"

    def test_navigate_to_parent(self) -> None:
        """Test navigating to parent range from child range overview."""
        self.log("Testing: Navigate to parent range")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.child_range_name)
        self.console.ranges.wait_for_overview(self.child_range_name)
        self.console.ranges.navigate_to_parent(self.staged_range_name)
        self.console.ranges.wait_for_overview(self.staged_range_name)
        assert self.console.ranges.is_overview_showing(
            self.staged_range_name
        ), "Should navigate to parent range overview"

    def test_create_range_with_labels(self) -> None:
        """Test creating a range with labels attached."""
        self.log("Testing: Create range with labels")
        self.labeled_range_name = f"LabeledRange_{self.rand_suffix}"
        self.console.ranges.create(
            self.labeled_range_name, persisted=True, labels=[self.test_label_name]
        )
        self.console.ranges.open_explorer()
        assert self.console.ranges.exists_in_explorer(
            self.labeled_range_name
        ), "Labeled range should exist in explorer"

    def test_open_range_explorer(self) -> None:
        """Test opening the Range Explorer."""
        self.log("Testing: Open Range Explorer")
        self.console.ranges.open_explorer()
        all_ranges_header = self.page.get_by_text("All Ranges")
        assert all_ranges_header.is_visible(), "Range Explorer should show 'All Ranges'"

    def test_range_exists_in_explorer(self) -> None:
        """Test that created range exists in the explorer."""
        self.log("Testing: Range exists in explorer")
        assert self.console.ranges.exists_in_explorer(
            self.range_name
        ), f"Range '{self.range_name}' should exist in explorer"

    def test_favorite_range(self) -> None:
        """Test favoriting and unfavoriting a range."""
        self.log("Testing: Favorite range")
        self.console.ranges.favorite_from_explorer(self.range_name)
        assert self.console.ranges.exists_in_toolbar(
            self.range_name
        ), f"Range '{self.range_name}' should appear in toolbar after favoriting"

        self.log("Testing: Unfavorite range")
        # Will raise an error if the range is still in the toolbar
        self.console.ranges.unfavorite_from_toolbar(self.range_name)

        self.log("Testing: Re-favorite range for subsequent tests")
        self.console.ranges.open_explorer()
        self.console.ranges.favorite_from_explorer(self.range_name)
        assert self.console.ranges.exists_in_toolbar(
            self.range_name
        ), f"Range '{self.range_name}' should appear in toolbar after re-favoriting"

    def test_set_active_range(self) -> None:
        """Test setting a range as active from the toolbar."""
        self.log("Testing: Set active range")
        self.console.ranges.set_active(self.range_name)
        item = self.console.ranges.get_toolbar_item(self.range_name)
        class_attr = item.get_attribute("class") or ""
        assert "pluto--selected" in class_attr, "Range should be marked as active"

    def test_change_times_in_overview(self) -> None:
        """Test changing start and end times in the range overview."""
        self.log("Testing: Change times in overview")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        self.console.ranges.set_start_time_in_overview(
            year=2024, month="January", day=1, hour=0, minute=0, second=0
        )
        self.console.ranges.set_end_time_in_overview(
            year=2024, month="January", day=2, hour=0, minute=0, second=0
        )

        rng = self.client.ranges.retrieve(name=self.labeled_range_name)
        start_ts = sy.TimeStamp(rng.time_range.start)
        end_ts = sy.TimeStamp(rng.time_range.end)
        start_utc = start_ts.datetime(timezone.utc)
        end_utc = end_ts.datetime(timezone.utc)
        assert (
            start_utc.year == 2024
        ), f"Start year should be 2024, got {start_utc.year}"
        assert (
            start_utc.month == 1
        ), f"Start month should be January, got {start_utc.month}"
        assert start_utc.day == 1, f"Start day should be 1, got {start_utc.day}"
        assert end_utc.day == 2, f"End day should be 2, got {end_utc.day}"

    def test_change_stage_in_overview(self) -> None:
        """Test changing stage in the range overview (which also changes times)."""
        self.log("Testing: Change stage in overview")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        self.console.ranges.set_stage_in_overview("In Progress")

        rng = self.client.ranges.retrieve(name=self.labeled_range_name)
        now = int(sy.TimeStamp.now())
        assert rng.time_range.start < now, "Start should be in the past"
        assert rng.time_range.end > now, "End should be in the future"

    def test_add_label_in_overview(self) -> None:
        """Test adding a label to a range in the overview."""
        self.log("Testing: Add label in overview")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        self.console.ranges.add_label_in_overview(self.second_label_name)

        labels = self.console.ranges.get_labels_in_overview()
        assert (
            self.second_label_name in labels
        ), f"Label '{self.second_label_name}' should be in overview"

    def test_remove_label_in_overview(self) -> None:
        """Test removing a label from a range in the overview."""
        self.log("Testing: Remove label in overview")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        self.console.ranges.remove_label_in_overview(self.second_label_name)

        labels = self.console.ranges.get_labels_in_overview()
        assert (
            self.second_label_name not in labels
        ), f"Label '{self.second_label_name}' should be removed from overview"

    def test_rename_range_from_tab(self) -> None:
        """Test renaming a range from the tab name."""
        self.log("Testing: Rename range from tab")
        original_rng = self.client.ranges.retrieve(name=self.labeled_range_name)
        original_key = original_rng.key

        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        new_name = f"RenamedTab_{self.rand_suffix}"
        self.console.layout.rename_tab(
            old_name=self.labeled_range_name, new_name=new_name
        )

        rng = self.client.ranges.retrieve(name=new_name)
        assert rng.name == new_name, f"Range should be renamed to '{new_name}'"
        assert rng.key == original_key, "Range key should remain the same after rename"
        self.labeled_range_name = new_name

    def test_rename_range_from_overview(self) -> None:
        """Test renaming a range from the overview name field."""
        self.log("Testing: Rename range from overview")
        original_rng = self.client.ranges.retrieve(name=self.labeled_range_name)
        original_key = original_rng.key

        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        new_name = f"RenamedOverview_{self.rand_suffix}"
        self.console.ranges.rename_from_overview(new_name)

        rng = self.client.ranges.retrieve(name=new_name)
        assert rng.name == new_name, f"Range should be renamed to '{new_name}'"
        assert rng.key == original_key, "Range key should remain the same after rename"
        self.labeled_range_name = new_name

    def test_copy_python_code(self) -> None:
        """Test copying Python code from the range overview."""
        self.log("Testing: Copy Python code")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        self.console.ranges.copy_python_code_from_overview()
        notifications = self.console.notifications.check(timeout=2)
        messages = [n.get("message", "") for n in notifications]
        assert any(
            "Python code to retrieve" in msg for msg in messages
        ), "Should show Python code copied notification"
        self.console.notifications.close_all()

    def test_copy_typescript_code(self) -> None:
        """Test copying TypeScript code from the range overview."""
        self.log("Testing: Copy TypeScript code")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        self.console.ranges.copy_typescript_code_from_overview()
        notifications = self.console.notifications.check(timeout=2)
        messages = [n.get("message", "") for n in notifications]
        assert any(
            "TypeScript code to retrieve" in msg for msg in messages
        ), "Should show TypeScript code copied notification"
        self.console.notifications.close_all()

    def test_copy_link(self) -> None:
        """Test copying link from the range overview."""
        self.log("Testing: Copy link")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        self.console.ranges.copy_link_from_overview()
        notifications = self.console.notifications.check(timeout=2)
        messages = [n.get("message", "") for n in notifications]
        assert any(
            "link to" in msg.lower() for msg in messages
        ), "Should show link copied notification"
        self.console.notifications.close_all()

    def test_download_csv(self) -> None:
        """Test downloading CSV data from the range overview."""
        self.log("Testing: Download CSV")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.labeled_range_name)
        self.console.ranges.wait_for_overview(self.labeled_range_name)

        channel = "sy_node_1_metrics_time"
        csv_content = self.console.ranges.download_csv(self.labeled_range_name, channel)

        assert csv_content, "CSV content should not be empty"
        assert channel in csv_content, f"CSV should contain channel {channel}"
        lines = csv_content.strip().split("\n")
        assert len(lines) > 1, "CSV should have header and data rows"

    def test_rename_range(self) -> None:
        """Test renaming a range from the explorer."""
        self.log("Testing: Rename range")
        self.console.ranges.open_explorer()
        new_name = f"RenamedRange_{self.rand_suffix}"
        self.console.ranges.rename_from_explorer(self.range_name, new_name)
        assert self.console.ranges.exists_in_explorer(
            new_name
        ), f"Range should exist with name {new_name}"
        self.range_name = new_name

    def test_delete_range(self) -> None:
        """Test deleting a range from the explorer."""
        self.log("Testing: Delete range")
        rng = self.client.ranges.retrieve(name=self.range_name)
        range_key = rng.key
        self.console.ranges.delete_from_explorer(self.range_name)
        try:
            self.client.ranges.retrieve(key=range_key)
            raise AssertionError("Range should be deleted but was found")
        except sy.NotFoundError:
            pass

    def test_navigate_to_child_range(self) -> None:
        """Test clicking on a child range to navigate to its overview."""
        self.log("Testing: Navigate to child range")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(self.staged_range_name)
        self.console.ranges.wait_for_overview(self.staged_range_name)

        self.console.ranges.click_child_range(self.child_range_name)
        self.console.ranges.wait_for_overview(self.child_range_name)
        assert self.console.ranges.is_overview_showing(
            self.child_range_name
        ), "Should navigate to child range overview"

    def test_create_child_range_from_overview(self) -> None:
        """Test creating a child range from the parent overview."""
        self.log("Testing: Create child range from overview")
        self.console.ranges.navigate_to_parent(self.staged_range_name)
        self.console.ranges.wait_for_overview(self.staged_range_name)

        self.console.ranges.create_child_range_from_overview()
        self.new_child_range_name = f"NewChild_{self.rand_suffix}"
        name_input = self.page.locator(
            f"input[placeholder='{self.console.ranges.NAME_INPUT_PLACEHOLDER}']"
        )
        name_input.fill(self.new_child_range_name)
        save_button = self.page.get_by_role("button", name="Save to Synnax")
        save_button.click(timeout=2000)
        modal = self.page.locator(self.console.ranges.CREATE_MODAL_SELECTOR)
        modal.wait_for(state="hidden", timeout=5000)

        self.console.ranges.open_explorer()
        assert self.console.ranges.exists_in_explorer(
            self.new_child_range_name
        ), "New child range should appear in explorer"

    def test_change_child_range_stage(self) -> None:
        """Test changing the stage of a child range from the parent overview."""
        self.log("Testing: Change child range stage")
        self.console.ranges.open_overview_from_explorer(self.staged_range_name)
        self.console.ranges.wait_for_overview(self.staged_range_name)

        self.console.ranges.set_child_range_stage(self.child_range_name, "In Progress")

        rng = self.client.ranges.retrieve(name=self.child_range_name)
        now = int(sy.TimeStamp.now())
        assert rng.time_range.start < now, "Child range start should be in the past"
        assert rng.time_range.end > now, "Child range end should be in the future"

    def test_favorite_child_range(self) -> None:
        """Test favoriting a child range from the parent overview."""
        self.log("Testing: Favorite child range")
        self.console.ranges.favorite_child_range(self.child_range_name)

        assert self.console.ranges.exists_in_toolbar(
            self.child_range_name
        ), "Child range should appear in toolbar after favoriting"

    def test_unfavorite_child_range(self) -> None:
        """Test unfavoriting a child range from the parent overview."""
        self.log("Testing: Unfavorite child range")
        # Will raise an error if the range is still in the toolbar
        self.console.ranges.unfavorite_child_range(self.child_range_name)

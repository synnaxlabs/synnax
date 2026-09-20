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

import synnax as sy
from console.case import ConsoleCase
from console.plot import Plot


class RangeLifecycle(ConsoleCase):
    """Test the lifecycle of ranges in the Console UI."""

    rand_suffix: int
    test_label_name: str
    second_label_name: str
    local_range_name: str | None = None
    range_name: str | None = None
    staged_range_name: str | None = None
    child_range_name: str | None = None
    labeled_range_name: str | None = None
    new_child_range_name: str | None = None

    def setup(self) -> None:
        super().setup()
        self.rand_suffix = random.randint(1000, 9999)
        self.test_label_name = f"RangeLabel_{self.rand_suffix}"
        self.second_label_name = f"SecondLabel_{self.rand_suffix}"

        self.console.labels.create(name=self.test_label_name)
        self.console.labels.create(name=self.second_label_name)

    def teardown(self) -> None:
        ranges_to_delete = [
            self.local_range_name,
            self.labeled_range_name,
            self.new_child_range_name,
            self.child_range_name,
            self.staged_range_name,
        ]

        with self._try_to("delete ranges"):
            self.console.ranges.explorer.open()
            for range_name in ranges_to_delete:
                if range_name and self.console.ranges.explorer.exists(range_name):
                    self.console.ranges.explorer.delete(range_name)

        with self._try_to("delete labels"):
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
        self.test_save_local_range()
        self.test_create_persisted_range()
        self.test_create_range_with_stage()
        self.test_create_range_with_parent()
        self.test_create_range_with_labels()

        # Range Explorer
        self.test_open_range_explorer()
        self.test_range_exists_in_explorer()
        self.test_unfavorite_favorite_range()
        self.test_set_active_range()
        self.test_remove_active_range()

        # Range Toolbar - Plot Operations
        self.test_add_to_new_plot()
        self.test_add_to_active_plot()

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

    def test_open_range_toolbar(self) -> None:
        """Test opening the ranges toolbar."""
        self.log("Testing: Open ranges toolbar")
        self.console.ranges.toolbar.show()
        toolbar_title = self.page.get_by_text("Ranges", exact=True).first
        assert toolbar_title.is_visible(), "Ranges toolbar should be visible"

    def test_create_local_range(self) -> None:
        """Test creating a local (non-persisted) range."""
        self.log("Testing: Create local range")
        local_range_name = f"LocalRange_{self.rand_suffix}"
        self.console.ranges.create(local_range_name, persisted=False)
        assert self.console.ranges.toolbar.exists(local_range_name), (
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
        self.console.ranges.explorer.open()
        assert self.console.ranges.explorer.exists(self.staged_range_name), (
            "Staged range should exist in explorer"
        )

    def test_create_range_with_parent(self) -> None:
        """Test creating a range with a parent range."""
        self.log("Testing: Create range with parent")
        self.child_range_name = f"ChildRange_{self.rand_suffix}"
        self.console.ranges.create(
            self.child_range_name, persisted=True, parent=self.staged_range_name
        )
        self.console.ranges.explorer.open()
        assert self.console.ranges.explorer.exists(self.child_range_name), (
            "Child range should exist in explorer"
        )

    def test_navigate_to_parent(self) -> None:
        """Test navigating to parent range from child range overview."""
        assert self.child_range_name is not None
        assert self.staged_range_name is not None
        self.log("Testing: Navigate to parent range")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.child_range_name)
        self.console.ranges.overview.wait_for(self.child_range_name)
        self.console.ranges.overview.navigate_to_parent(self.staged_range_name)
        self.console.ranges.overview.wait_for(self.staged_range_name)
        assert self.console.ranges.overview.is_showing(self.staged_range_name), (
            "Should navigate to parent range overview"
        )

    def test_create_range_with_labels(self) -> None:
        """Test creating a range with labels attached."""
        self.log("Testing: Create range with labels")
        self.labeled_range_name = f"LabeledRange_{self.rand_suffix}"
        self.console.ranges.create(
            self.labeled_range_name, persisted=True, labels=[self.test_label_name]
        )
        self.console.ranges.explorer.open()
        assert self.console.ranges.explorer.exists(self.labeled_range_name), (
            "Labeled range should exist in explorer"
        )

    def test_open_range_explorer(self) -> None:
        """Test opening the Range Explorer."""
        self.log("Testing: Open Range Explorer")
        self.console.ranges.explorer.open()
        static_view_name = self.console.ranges.explorer.views.static_view_name
        header = self.page.get_by_text(static_view_name)
        assert header.is_visible(), f"Range Explorer should show '{static_view_name}'"

    def test_range_exists_in_explorer(self) -> None:
        """Test that created range exists in the explorer."""
        assert self.range_name is not None
        self.log("Testing: Range exists in explorer")
        assert self.console.ranges.explorer.exists(self.range_name), (
            f"Range '{self.range_name}' should exist in explorer"
        )

    def test_unfavorite_favorite_range(self) -> None:
        """Test unfavoriting and favoriting a range from the explorer."""
        assert self.range_name is not None
        self.log("Testing: Unfavorite range")
        self.console.ranges.explorer.unfavorite(self.range_name)

        self.log("Testing: Favorite range")
        self.console.ranges.explorer.favorite(self.range_name)
        assert self.console.ranges.toolbar.exists(self.range_name), (
            f"Range '{self.range_name}' should appear in toolbar after favoriting"
        )

    def test_set_active_range(self) -> None:
        """Test setting a range as active from the toolbar."""
        assert self.range_name is not None
        self.log("Testing: Set active range")
        self.console.ranges.set_active(self.range_name)
        item = self.console.ranges.toolbar.get_item(self.range_name)
        class_attr = item.get_attribute("class") or ""
        assert "pluto--selected" in class_attr, "Range should be marked as active"

    def test_remove_active_range(self) -> None:
        """Test removing an active range from the toolbar via context menu."""
        assert self.range_name is not None
        self.log("Testing: Remove active range from toolbar")
        self.console.ranges.toolbar.unfavorite(self.range_name)

        self.log("Re-favoriting and re-activating range for subsequent tests")
        self.console.ranges.explorer.open()
        self.console.ranges.explorer.favorite(self.range_name)
        self.console.ranges.set_active(self.range_name)

    def test_add_to_new_plot(self) -> None:
        """Test adding a range to a new line plot via toolbar context menu."""
        assert self.range_name is not None
        self.log("Testing: Add range to new plot")
        self.console.ranges.toolbar.add_to_new_plot(self.range_name)
        expected_tab = f"Plot for {self.range_name}"
        self._cleanup_pages.append(expected_tab)
        self.console.layout.wait_for_tab(expected_tab)
        self.console.layout.close_tab(expected_tab)

    def test_add_to_active_plot(self) -> None:
        """Test adding a range to the active line plot via toolbar context menu."""
        assert self.range_name is not None
        self.log("Testing: Add range to active plot")
        plot_name = f"TestPlot_{self.rand_suffix}"
        plot = self.console.pages.create(Plot, plot_name)
        self._cleanup_pages.append(plot.page_name)
        plot.add_channels("Y1", "sy_node_1_metrics_cpu_percentage")
        self.console.layout.hide_visualization_toolbar()
        self.console.ranges.toolbar.add_to_active_plot(self.range_name)
        self.console.layout.close_tab(plot_name)

    def test_save_local_range(self) -> None:
        """Test saving a local range to Synnax via toolbar context menu."""
        assert self.local_range_name is not None
        self.log("Testing: Save local range to Synnax")
        self.console.ranges.toolbar.save_to_synnax(self.local_range_name)
        rng = self.client.ranges.retrieve(name=self.local_range_name)
        assert rng.name == self.local_range_name, (
            f"Range '{self.local_range_name}' should be persisted to Synnax"
        )

    def test_change_times_in_overview(self) -> None:
        """Test changing start and end times in the range overview."""
        assert self.labeled_range_name is not None
        self.log("Testing: Change times in overview")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        self.console.ranges.overview.set_start_time(
            year=2024, month="January", day=1, hour=0, minute=0, second=0
        )
        self.console.ranges.overview.set_end_time(
            year=2024, month="January", day=2, hour=0, minute=0, second=0
        )

        # The overview form autosaves on a debounce, so poll until the last
        # edit reaches the server before asserting.
        for _ in range(10):
            rng = self.client.ranges.retrieve(name=self.labeled_range_name)
            start_utc = sy.TimeStamp(rng.time_range.start).datetime(timezone.utc)
            end_utc = sy.TimeStamp(rng.time_range.end).datetime(timezone.utc)
            if end_utc.day == 2:
                break
            sy.sleep(0.5)
        assert start_utc.year == 2024, (
            f"Start year should be 2024, got {start_utc.year}"
        )
        assert start_utc.month == 1, (
            f"Start month should be January, got {start_utc.month}"
        )
        assert start_utc.day == 1, f"Start day should be 1, got {start_utc.day}"
        assert end_utc.day == 2, f"End day should be 2, got {end_utc.day}"

        self.log("Verifying time sync to Ranges Toolbar")
        self.console.ranges.favorite(self.labeled_range_name)
        toolbar_time = self.console.ranges.toolbar.get_item_time(
            self.labeled_range_name
        )
        assert "Jan" in toolbar_time, (
            f"Toolbar should show Jan time, got '{toolbar_time}'"
        )

        self.log("Verifying time sync to Range Explorer")
        self.console.ranges.explorer.open()
        explorer_time = self.console.ranges.explorer.get_item_time(
            self.labeled_range_name
        )
        assert "Jan" in explorer_time, (
            f"Explorer should show Jan time, got '{explorer_time}'"
        )

        self.console.ranges.toolbar.unfavorite(self.labeled_range_name)

    def retrieve_renamed(self, name: str) -> sy.Range:
        """Retrieve a range by its new name, polling while the debounced
        autosave of the rename lands."""
        for _ in range(10):
            try:
                return self.client.ranges.retrieve(name=name)
            except sy.QueryError:
                sy.sleep(0.5)
        return self.client.ranges.retrieve(name=name)

    def assert_range_spans_now(self, name: str) -> None:
        """Poll until the range's time range spans now.

        Stage edits autosave on a debounce, so an immediate read races them.
        """
        now = int(sy.TimeStamp.now())
        for _ in range(10):
            rng = self.client.ranges.retrieve(name=name)
            now = int(sy.TimeStamp.now())
            if rng.time_range.start < now < rng.time_range.end:
                break
            sy.sleep(0.5)
        assert rng.time_range.start < now, f"'{name}' start should be in the past"
        assert rng.time_range.end > now, f"'{name}' end should be in the future"

    def test_change_stage_in_overview(self) -> None:
        """Test changing stage in the range overview (which also changes times)."""
        assert self.labeled_range_name is not None
        self.log("Testing: Change stage in overview")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        self.console.ranges.overview.set_stage("In progress")
        self.assert_range_spans_now(self.labeled_range_name)

    def test_add_label_in_overview(self) -> None:
        """Test adding a label to a range in the overview."""
        assert self.labeled_range_name is not None
        self.log("Testing: Add label in overview")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        self.console.ranges.overview.add_label(self.second_label_name)

        labels = self.console.ranges.overview.get_labels()
        assert self.second_label_name in labels, (
            f"Label '{self.second_label_name}' should be in overview"
        )

    def test_remove_label_in_overview(self) -> None:
        """Test removing a label from a range in the overview."""
        assert self.labeled_range_name is not None
        self.log("Testing: Remove label in overview")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        self.console.ranges.overview.remove_label(self.second_label_name)

        labels = self.console.ranges.overview.get_labels()
        assert self.second_label_name not in labels, (
            f"Label '{self.second_label_name}' should be removed from overview"
        )

    def test_rename_range_from_tab(self) -> None:
        """Test renaming a range from the tab name."""
        assert self.labeled_range_name is not None
        self.log("Testing: Rename range from tab")
        original_rng = self.client.ranges.retrieve(name=self.labeled_range_name)
        original_key = original_rng.key

        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        new_name = f"RenamedTab_{self.rand_suffix}"
        self.console.layout.rename_tab(
            old_name=self.labeled_range_name, new_name=new_name
        )

        rng = self.retrieve_renamed(new_name)
        assert rng.name == new_name, f"Range should be renamed to '{new_name}'"
        assert rng.key == original_key, "Range key should remain the same after rename"
        self.labeled_range_name = new_name

    def test_rename_range_from_overview(self) -> None:
        """Test renaming a range from the overview name field."""
        assert self.labeled_range_name is not None
        self.log("Testing: Rename range from overview")
        original_rng = self.client.ranges.retrieve(name=self.labeled_range_name)
        original_key = original_rng.key

        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        new_name = f"RenamedOverview_{self.rand_suffix}"
        self.console.ranges.overview.rename(new_name)

        rng = self.retrieve_renamed(new_name)
        assert rng.name == new_name, f"Range should be renamed to '{new_name}'"
        assert rng.key == original_key, "Range key should remain the same after rename"
        self.labeled_range_name = new_name

    def test_copy_python_code(self) -> None:
        """Test copying Python code from the range overview."""
        assert self.labeled_range_name is not None
        self.log("Testing: Copy Python code")
        self.console.notifications.close_all()
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        self.console.ranges.overview.copy_python_code()
        notifications = self.console.notifications.check(timeout=10)
        messages = [n.get("message", "") for n in notifications]
        assert any("Python code for" in msg for msg in messages), (
            "Should show Python code copied notification"
        )
        self.console.notifications.close_all()

    def test_copy_typescript_code(self) -> None:
        """Test copying TypeScript code from the range overview."""
        assert self.labeled_range_name is not None
        self.log("Testing: Copy TypeScript code")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        self.console.ranges.overview.copy_typescript_code()
        notifications = self.console.notifications.check(timeout=2)
        messages = [n.get("message", "") for n in notifications]
        assert any("TypeScript code for" in msg for msg in messages), (
            "Should show TypeScript code copied notification"
        )
        self.console.notifications.close_all()

    def test_copy_link(self) -> None:
        """Test copying link from the range overview."""
        assert self.labeled_range_name is not None
        self.log("Testing: Copy link")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        self.console.ranges.overview.copy_link()
        notifications = self.console.notifications.check(timeout=2)
        messages = [n.get("message", "") for n in notifications]
        assert any("link to" in msg.lower() for msg in messages), (
            "Should show link copied notification"
        )
        self.console.notifications.close_all()

    def test_download_csv(self) -> None:
        """Test downloading CSV data from the range overview."""
        assert self.labeled_range_name is not None
        self.log("Testing: Download CSV")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.labeled_range_name)
        self.console.ranges.overview.wait_for(self.labeled_range_name)

        channel = "sy_node_1_metrics_time"
        csv_content = self.console.ranges.overview.download_csv(
            self.labeled_range_name, channel
        )

        assert csv_content, "CSV content should not be empty"
        assert channel in csv_content, f"CSV should contain channel {channel}"
        lines = csv_content.strip().split("\n")
        assert len(lines) > 1, "CSV should have header and data rows"

    def test_rename_range(self) -> None:
        """Test renaming a range from the explorer."""
        assert self.range_name is not None
        self.log("Testing: Rename range")
        self.console.ranges.explorer.open()
        new_name = f"RenamedRange_{self.rand_suffix}"
        self.console.ranges.explorer.rename(self.range_name, new_name)
        assert self.console.ranges.explorer.exists(new_name), (
            f"Range should exist with name {new_name}"
        )
        self.range_name = new_name

    def test_delete_range(self) -> None:
        """Test deleting a range from the explorer."""
        assert self.range_name is not None
        self.log("Testing: Delete range")
        rng = self.client.ranges.retrieve(name=self.range_name)
        range_key = rng.key
        self.console.ranges.explorer.delete(self.range_name)
        try:
            self.client.ranges.retrieve(key=range_key)
            raise AssertionError("Range should be deleted but was found")
        except sy.NotFoundError:
            pass

    def test_navigate_to_child_range(self) -> None:
        """Test clicking on a child range to navigate to its overview."""
        assert self.staged_range_name is not None
        assert self.child_range_name is not None
        self.log("Testing: Navigate to child range")
        self.console.ranges.explorer.open()
        self.console.ranges.overview.open(self.staged_range_name)
        self.console.ranges.overview.wait_for(self.staged_range_name)

        self.console.ranges.overview.click_child_range(self.child_range_name)
        self.console.ranges.overview.wait_for(self.child_range_name)
        assert self.console.ranges.overview.is_showing(self.child_range_name), (
            "Should navigate to child range overview"
        )

    def test_create_child_range_from_overview(self) -> None:
        """Test creating a child range from the parent overview."""
        assert self.staged_range_name is not None
        self.log("Testing: Create child range from overview")
        self.console.ranges.overview.navigate_to_parent(self.staged_range_name)
        self.console.ranges.overview.wait_for(self.staged_range_name)

        self.new_child_range_name = f"NewChild_{self.rand_suffix}"
        self.console.ranges.overview.create_child_range(self.new_child_range_name)

        self.console.ranges.explorer.open()
        assert self.console.ranges.explorer.exists(self.new_child_range_name), (
            "New child range should appear in explorer"
        )

    def test_change_child_range_stage(self) -> None:
        """Test changing the stage of a child range from the parent overview."""
        assert self.staged_range_name is not None
        assert self.child_range_name is not None
        self.log("Testing: Change child range stage")
        self.console.ranges.overview.open(self.staged_range_name)
        self.console.ranges.overview.wait_for(self.staged_range_name)

        self.console.ranges.overview.set_child_range_stage(
            self.child_range_name, "In progress"
        )
        self.assert_range_spans_now(self.child_range_name)

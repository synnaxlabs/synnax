#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase


class LabelLifecycle(ConsoleCase):
    """Test the lifecycle of labels."""

    def run(self) -> None:
        self.test_open_label_modal()
        self.test_create_label()
        self.test_rename_label()
        self.test_change_label_color()
        self.test_rename_label_syncs_with_range_toolbar()
        self.test_change_label_color_syncs_with_range_toolbar()
        self.test_delete_label()

    def test_open_label_modal(self) -> None:
        """Test opening the Edit Labels modal via command palette."""
        self.log("Testing: Open label modal")

        self.console.labels.open_edit_modal()

        modal = self.page.locator(".console-label__edit")
        assert modal.is_visible(), "Edit Labels modal should be visible"

        header = self.page.locator(".console-label__edit-header")
        assert header.is_visible(), "Edit header should be visible"

        add_button = self.page.locator(".console-label__add-btn")
        assert add_button.count() > 0, "Add Label button should be present"

        self.log("Successfully opened Edit Labels modal")

        self.console.labels.close_edit_modal()
        self.log("Successfully closed Edit Labels modal")

    def test_create_label(self) -> None:
        """Test creating a new label."""
        self.log("Testing: Create label")

        label_name = "TestLabel"

        self.console.labels.open_edit_modal()
        self.console.labels.create(name=label_name)

        label_item = self.page.locator(
            f".console-label__list-item:not(.console--create) input[value='{label_name}']"
        )
        assert label_item.count() > 0, f"Label '{label_name}' should appear in the list"

        self.log(f"Successfully created label: {label_name}")

        self.console.labels.close_edit_modal()

    def test_rename_label(self) -> None:
        """Test renaming an existing label."""
        self.log("Testing: Rename label")

        old_name = "TestLabel"
        new_name = "RenamedLabel"

        self.console.labels.open_edit_modal()

        labels_before = self.console.labels.list_all()
        assert old_name in labels_before, f"Label '{old_name}' should exist"

        self.console.labels.rename(old_name, new_name)

        self.page.wait_for_timeout(300)
        self.console.labels.open_edit_modal()

        label_item = self.page.locator(
            f".console-label__list-item:not(.console--create) input[value='{new_name}']"
        )
        assert label_item.count() > 0, f"Label should be renamed to '{new_name}'"

        self.log(f"Successfully renamed label from '{old_name}' to '{new_name}'")

        self.console.labels.close_edit_modal()

    def test_change_label_color(self) -> None:
        """Test changing the color of a label."""
        self.log("Testing: Change label color")

        label_name = "RenamedLabel"
        new_color = "#FF5733"  # Orange color

        self.console.labels.change_color(label_name, new_color)

        self.log(f"Successfully changed color of label '{label_name}' to {new_color}")

    def test_rename_label_syncs_with_range_toolbar(self) -> None:
        """Test that renaming a label updates the range toolbar."""
        self.log("Testing: Rename label syncs with range toolbar")

        label_name = "SyncTestLabel"
        range_name = "LabelSyncRange"

        self.console.labels.open_edit_modal()
        self.console.labels.create(name=label_name)
        self.console.labels.close_edit_modal()

        self.console.ranges.create(name=range_name, persisted=True, labels=[label_name])
        self.console.ranges.open_explorer()
        self.console.ranges.favorite_from_explorer(range_name)
        self.console.ranges.show_toolbar()

        assert self.console.ranges.exists_in_toolbar(
            range_name
        ), f"Range '{range_name}' should be in toolbar"
        assert self.console.ranges.label_exists_in_toolbar(range_name, label_name)
        self.log(f"  - Label '{label_name}' visible in toolbar")

        new_label_name = "RenamedSyncLabel"
        self.console.labels.rename(label_name, new_label_name)

        self.console.ranges.show_toolbar()
        assert self.console.ranges.label_exists_in_toolbar(
            range_name, new_label_name
        ), f"Label should be renamed to '{new_label_name}' in toolbar"
        assert not self.console.ranges.label_exists_in_toolbar(
            range_name, label_name
        ), f"Old label name '{label_name}' should not exist"

        self.log(f"  - Label renamed to '{new_label_name}' in toolbar")

    def test_change_label_color_syncs_with_range_toolbar(self) -> None:
        """Test that changing a label color updates the range toolbar."""
        self.log("Testing: Change label color syncs with range toolbar")

        label_name = "RenamedSyncLabel"
        range_name = "LabelSyncRange"

        self.console.ranges.show_toolbar()
        original_color = self.console.ranges.get_label_color_in_toolbar(
            range_name, label_name
        )
        self.log(f"  - Original color: {original_color}")

        new_color = "#00FF00"
        self.console.labels.change_color(label_name, new_color)

        self.console.ranges.show_toolbar()
        self.page.wait_for_timeout(500)
        updated_color = self.console.ranges.get_label_color_in_toolbar(
            range_name, label_name
        )
        assert updated_color is not None, "Updated color should not be None"
        self.log(f"  - Updated color: {updated_color}")

        assert updated_color != original_color, "Label color should have changed"
        assert (
            "0, 255, 0" in updated_color
        ), f"Expected green color, got: {updated_color}"

        self.log("  - Label color updated in toolbar")

        self._cleanup_sync_test(label_name, range_name)

    def _cleanup_sync_test(self, label_name: str, range_name: str) -> None:
        """Clean up resources created for sync tests."""
        self.console.labels.open_edit_modal()
        self.console.labels.delete(label_name)
        self.console.labels.close_edit_modal()

        self.console.ranges.open_explorer()
        for _ in range(5):
            if not self.console.ranges.exists_in_explorer(range_name):
                break
            self.console.ranges.delete_from_explorer(range_name)
            self.page.wait_for_timeout(500)

    def test_delete_label(self) -> None:
        """Test deleting a label (also cleans up the test label)."""
        self.log("Testing: Delete label")

        label_name = "RenamedLabel"

        self.console.labels.open_edit_modal()
        self.console.labels.delete(label_name)

        self.console.labels.close_edit_modal()
        self.page.wait_for_timeout(300)
        self.console.labels.open_edit_modal()

        label_item = self.page.locator(
            f".console-label__list-item:not(.console--create) input[value='{label_name}']"
        )
        assert label_item.count() == 0, f"Label '{label_name}' should be deleted"

        self.log(f"Successfully deleted label: {label_name}")
        self.console.labels.close_edit_modal()

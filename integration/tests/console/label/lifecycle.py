#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from framework.utils import get_random_name


class LabelLifecycle(ConsoleCase):
    """Test the lifecycle of labels."""

    def run(self) -> None:
        self.test_create_label()
        # self.test_rename_label()
        # self.test_change_label_color()
        # self.test_rename_label_syncs_with_range_toolbar()
        # self.test_change_label_color_syncs_with_range_toolbar()
        # self.test_delete_label()

    def test_create_label(self) -> None:
        """Test creating a new label."""
        self.log("Testing: Create label")
        label_name = get_random_name()
        self.console.labels.create(label_name, color="#F0FE00")
        assert self.console.labels.exists(
            label_name
        ), f"Label '{label_name}' should exist"
        label_color = self.console.labels.get_color(label_name)
        self.log(f"Label color: {label_color}")
        assert label_color is not None, f"Label '{label_name}' should have a color"
        assert (
            label_color == "#F0FE00"
        ), f"Label '{label_name}' should have color '#F0FE00'"
        self.log(f"Successfully created label: {label_name} with color '#F0FE00'")
        self.console.labels.delete(label_name)

    def test_rename_label(self) -> None:
        """Test renaming an existing label."""
        self.log("Testing: Rename label")

        old_name = "TestLabel"
        new_name = "RenamedLabel"

        labels_before = self.console.labels.list_all()
        assert old_name in labels_before, f"Label '{old_name}' should exist"

        self.console.labels.rename(old_name, new_name)

        self.console.labels.exists(new_name)
        self.log(f"Successfully renamed label from '{old_name}' to '{new_name}'")

    def test_change_label_color(self) -> None:
        """Test changing the color of a label."""
        self.log("Testing: Change label color")

        label_name = "RenamedLabel"
        new_color = "#FF5733"  # Orange color

        self.console.labels.change_color(label_name, new_color)

        ## Todo: test the new color

        self.log(f"Successfully changed color of label '{label_name}' to {new_color}")

    def test_rename_label_syncs_with_range_toolbar(self) -> None:
        """Test that renaming a label updates the range toolbar."""
        self.log("Testing: Rename label syncs with range toolbar")

        label_name = "SyncTestLabel"
        range_name = "LabelSyncRange"

        self.console.labels.create(name=label_name)

        self.console.ranges.create(name=range_name, persisted=True, labels=[label_name])
        self.console.ranges.open_explorer()
        self.console.ranges.favorite_from_explorer(range_name)

        assert self.console.ranges.exists_in_toolbar(
            range_name
        ), f"Range '{range_name}' should be in toolbar"
        assert self.console.ranges.label_exists_in_toolbar(range_name, label_name)
        self.log(f"  - Label '{label_name}' visible in toolbar")

        new_label_name = "RenamedSyncLabel"
        self.console.labels.rename(label_name, new_label_name)

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

        original_color = self.console.ranges.get_label_color_in_toolbar(
            range_name, label_name
        )
        self.log(f"  - Original color: {original_color}")

        new_color = "#00FF00"
        self.console.labels.change_color(label_name, new_color)

        updated_color = self.console.ranges.get_label_color_in_toolbar(
            range_name, label_name
        )
        assert updated_color is not None, "Updated color should not be None"
        self.log(f"  - Updated color: {updated_color}")

        assert (
            "0, 255, 0" in updated_color
        ), f"Expected green color, got: {updated_color}"

        self.log("  - Label color updated in toolbar")

        self._cleanup_sync_test(label_name, range_name)

    def _cleanup_sync_test(self, label_name: str, range_name: str) -> None:
        """Clean up resources created for sync tests."""
        self.console.labels.delete(label_name)

        self.console.ranges.open_explorer()
        for _ in range(5):
            if not self.console.ranges.exists_in_explorer(range_name):
                break
            self.console.ranges.delete_from_explorer(range_name)
            self.page.wait_for_timeout(100)

    def test_delete_label(self) -> None:
        """Test deleting a label (also cleans up the test label)."""
        self.log("Testing: Delete label")

        label_name = "RenamedLabel"

        self.console.labels.delete(label_name)

        self.page.wait_for_timeout(100)
        ## TODO: make a get label via name function on the client
        self.console.labels._open_edit_modal()

        label_item = self.page.locator(
            f".console-label__list-item:not(.console--create) input[value='{label_name}']"
        )
        assert label_item.count() == 0, f"Label '{label_name}' should be deleted"

        self.log(f"Successfully deleted label: {label_name}")
        self.console.labels._close_edit_modal()

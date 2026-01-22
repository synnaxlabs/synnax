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
        self.test_rename_label()
        self.test_change_label_color()
        # self.test_rename_label_syncs_with_range_toolbar()
        # self.test_change_label_color_syncs_with_range_toolbar()

    def test_create_label(self) -> None:
        """Test creating a new label."""
        self.log("Testing: Create label")
        name = get_random_name()
        color = "#F0FE00"
        self.console.labels.create(name, color=color)
        assert self.console.labels.exists(name), f"Label {name} should exist"
        label_color = self.console.labels.get_color(name)
        self.log(f"Label color: {label_color}")
        assert label_color == color, f"Label {name} should have color {color}"
        self.log(f"Successfully created label {name} with color '{color}'")
        self.console.labels.delete(name)

    def test_rename_label(self) -> None:
        """Test renaming an existing label."""
        self.log("Testing: Rename label")
        old_name = get_random_name()
        new_name = get_random_name()
        self.console.labels.create(old_name)
        self.console.labels.rename(old_name, new_name)
        assert self.console.labels.exists(new_name), f"Label {new_name} should exist"
        assert not self.console.labels.exists(
            old_name
        ), f"Label {old_name} should not exist"
        self.log(f"Successfully renamed label from {old_name} to {new_name}")
        self.console.labels.delete(new_name)

    def test_change_label_color(self) -> None:
        """Test changing the color of a label."""
        self.log("Testing: Change label color")
        name = get_random_name()
        self.console.labels.create(name, color="#F0FE00")
        new_color = "#FF5733"
        self.console.labels.change_color(name, new_color)
        changed_color = self.console.labels.get_color(name)
        assert changed_color == new_color, f"Label {name} should have color {new_color}"
        self.log(f"Successfully changed color of label {name} to {new_color}")
        self.console.labels.delete(name)

    def test_rename_label_syncs_with_range_toolbar(self) -> None:
        """Test that renaming a label updates the range toolbar."""
        ## TODO: Implement this test
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
        ## TODO: Implement this test
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
        ## TODO: Implement this function
        self.console.labels.delete(label_name)

        self.console.ranges.open_explorer()
        for _ in range(5):
            if not self.console.ranges.exists_in_explorer(range_name):
                break
            self.console.ranges.delete_from_explorer(range_name)
            self.page.wait_for_timeout(100)

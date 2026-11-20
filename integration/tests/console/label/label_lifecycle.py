#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from x import random_name


class LabelLifecycle(ConsoleCase):
    """Test the lifecycle of labels."""

    shared_label: str
    shared_range: str

    def setup(self) -> None:
        super().setup()
        self.shared_label = random_name()
        self.console.labels.create(self.shared_label, color="#F0FE00")
        self.shared_range = random_name()
        self.console.ranges.create(self.shared_range, labels=[self.shared_label])
        self.console.ranges.explorer.open()

    def teardown(self) -> None:
        self.page.keyboard.press("Escape")
        self.console.ranges.explorer.delete(self.shared_range)
        if self.console.labels.exists(self.shared_label):
            self.console.labels.delete(self.shared_label)
        super().teardown()

    def run(self) -> None:
        self.test_rename_label()
        self.test_change_label_color()
        self.test_rename_label_syncs_with_range_explorer()
        self.test_change_label_color_syncs_with_range_explorer()
        self.test_delete_label_syncs_with_range_explorer()

    def test_rename_label(self) -> None:
        """Test renaming an existing label."""
        self.log("Testing: Rename label")

        old_name = self.shared_label
        new_name = random_name()
        self.log(f"Renaming label {old_name}")

        self.console.labels.rename(old_name=old_name, new_name=new_name)
        self.log(f"Label {old_name} renamed to {new_name}")
        self.shared_label = new_name

    def test_change_label_color(self) -> None:
        """Test changing the color of a label."""
        self.log("Testing: Change label color")

        name = self.shared_label
        new_color = "#FF5733"
        self.console.labels.change_color(name=name, new_color=new_color)
        changed_color = self.console.labels.get_color(name)
        assert changed_color == new_color, f"Label {name} should have color {new_color}"
        self.log(f"Label {name} color changed to {new_color}")

    def test_rename_label_syncs_with_range_explorer(self) -> None:
        """Test that renaming a label updates the range explorer."""
        self.log("Testing: Rename label syncs with range explorer")

        old_label_name = self.shared_label
        range_name = self.shared_range
        explorer = self.console.ranges.explorer

        explorer.open()
        assert explorer.label_exists(range_name, label_name=old_label_name), (
            f"Label {old_label_name} should be visible in explorer for range "
            f"{range_name}"
        )
        self.log(f"Label {old_label_name} visible in explorer for range {range_name}")

        new_label_name = random_name()
        self.console.labels.rename(old_name=old_label_name, new_name=new_label_name)
        self.log(f"Label {old_label_name} renamed to {new_label_name}")

        explorer.open()
        assert explorer.label_exists(range_name, label_name=new_label_name), (
            f"Label {new_label_name} should be visible in explorer for range "
            f"{range_name}"
        )
        explorer.wait_for_label_removed(range_name, label_name=old_label_name)
        self.log(f"Label {new_label_name} visible in explorer for range {range_name}")
        self.shared_label = new_label_name

    def test_delete_label_syncs_with_range_explorer(self) -> None:
        """Test that deleting a label updates the range explorer."""
        self.log("Testing: Delete label syncs with range explorer")

        label_name = self.shared_label
        range_name = self.shared_range
        explorer = self.console.ranges.explorer

        explorer.open()
        assert explorer.label_exists(range_name, label_name=label_name), (
            f"Label {label_name} should be visible in explorer for range {range_name}"
        )
        self.log(f"Label {label_name} visible in explorer for range {range_name}")

        self.console.labels.delete(label_name)
        self.log(f"Label {label_name} deleted")

        explorer.open()
        explorer.wait_for_label_removed(range_name, label_name=label_name)
        self.log(f"Label {label_name} not visible in explorer for range {range_name}")

    def test_change_label_color_syncs_with_range_explorer(self) -> None:
        """Test that changing a label color updates the range explorer."""
        self.log("Testing: Change label color syncs with range explorer")

        label_name = self.shared_label
        range_name = self.shared_range
        explorer = self.console.ranges.explorer

        explorer.open()
        assert explorer.label_exists(range_name, label_name=label_name), (
            f"Label {label_name} should be visible in explorer for range {range_name}"
        )
        self.log(f"Label {label_name} visible in explorer for range {range_name}")

        new_color = "#00FF00"
        self.console.labels.change_color(name=label_name, new_color=new_color)
        self.log(f"Label {label_name} color changed to {new_color}")

        explorer.open()
        updated_color = explorer.get_label_color(range_name, label_name=label_name)
        assert updated_color == new_color, (
            f"Label {label_name} color in explorer should be {new_color}"
        )
        self.log(f"Label {label_name} color updated to {new_color} in explorer")

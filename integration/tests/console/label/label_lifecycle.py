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


class LabelLifecycle(ConsoleCase):
    """Test the lifecycle of labels."""

    def run(self) -> None:
        self.test_create_label()
        self.test_rename_label()
        self.test_change_label_color()
        self.test_rename_label_syncs_with_range_toolbar()
        self.test_delete_label_syncs_with_range_toolbar()
        self.test_change_label_color_syncs_with_range_toolbar()

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
        self.log(f"Label {name} created with color {color}")

        self.console.labels.delete(name)

    def test_rename_label(self) -> None:
        """Test renaming an existing label."""
        self.log("Testing: Rename label")

        old_name = get_random_name()
        new_name = get_random_name()
        self.console.labels.create(old_name)
        self.log(f"Label {old_name} created")

        self.console.labels.rename(old_name=old_name, new_name=new_name)
        assert self.console.labels.exists(new_name), f"Label {new_name} should exist"
        assert not self.console.labels.exists(
            old_name
        ), f"Label {old_name} should not exist"
        self.log(f"Label {old_name} renamed to {new_name}")

        self.console.labels.delete(new_name)

    def test_change_label_color(self) -> None:
        """Test changing the color of a label."""
        self.log("Testing: Change label color")

        name = get_random_name()
        old_color = "#F0FE00"
        self.console.labels.create(name, color=old_color)
        self.log(f"Label {name} created with color {old_color}")

        new_color = "#FF5733"
        self.console.labels.change_color(name=name, new_color=new_color)
        changed_color = self.console.labels.get_color(name)
        assert changed_color == new_color, f"Label {name} should have color {new_color}"
        self.log(f"Label {name} color changed to {new_color}")

        self.console.labels.delete(name)

    def test_rename_label_syncs_with_range_toolbar(self) -> None:
        """Test that renaming a label updates the range toolbar."""
        self.log("Testing: Rename label syncs with range toolbar")

        old_label_name = get_random_name()
        self.console.labels.create(old_label_name)
        range_name = get_random_name()
        self.console.ranges.create(range_name, persisted=True, labels=[old_label_name])
        self.log(f"Range {range_name} created with label {old_label_name}")

        self.console.ranges.open_explorer()
        print(f"[DEBUG_TEST_RENAME] Waiting for range '{range_name}' to appear in explorer")
        range_item = self.console.ranges.get_explorer_item(range_name)
        range_item.wait_for(state="visible", timeout=10000)
        print(f"[DEBUG_TEST_RENAME] Range '{range_name}' is now visible in explorer")

        self.console.ranges.favorite(range_name)
        print(f"[DEBUG_TEST_RENAME] Favorited range '{range_name}', checking if it appears in toolbar")

        self.console.ranges.show_toolbar()
        toolbar_range_item = self.console.ranges.get_toolbar_item(range_name)
        toolbar_range_item.wait_for(state="visible", timeout=5000)
        print(f"[DEBUG_TEST_RENAME] Range '{range_name}' is now visible in toolbar")
        assert self.console.ranges.label_exists_in_toolbar(
            range_name, old_label_name
        ), f"Label {old_label_name} should be visible in toolbar for range {range_name}"
        self.log(f"Label {old_label_name} visible in toolbar for range {range_name}")

        import synnax as sy
        print(f"[DEBUG_TEST_RENAME] BEFORE RENAME: Getting all labels for range '{range_name}'")
        labels_before = self.console.ranges.get_all_labels_in_toolbar(range_name)
        print(f"[DEBUG_TEST_RENAME] Labels before rename: {labels_before}")

        new_label_name = get_random_name()
        self.console.labels.rename(old_name=old_label_name, new_name=new_label_name)
        self.log(f"Label {old_label_name} renamed to {new_label_name}")

        print(f"[DEBUG_TEST_RENAME] AFTER RENAME: Getting all labels for range '{range_name}'")
        labels_after = self.console.ranges.get_all_labels_in_toolbar(range_name)
        print(f"[DEBUG_TEST_RENAME] Labels after rename: {labels_after}")

        assert self.console.ranges.label_exists_in_toolbar(
            range_name, new_label_name
        ), f"Label {new_label_name} should be visible in toolbar for range {range_name}"
        assert not self.console.ranges.label_exists_in_toolbar(
            range_name, old_label_name
        ), f"Label {old_label_name} should not be visible in toolbar for range {range_name}"
        self.log(f"Label {new_label_name} visible in toolbar for range {range_name}")

        self.console.labels.delete(new_label_name)
        self.console.ranges.delete_from_explorer(range_name)

    def test_delete_label_syncs_with_range_toolbar(self) -> None:
        """Test that deleting a label updates the range toolbar."""
        self.log("Testing: Delete label syncs with range toolbar")

        label_name = get_random_name()
        self.console.labels.create(label_name)
        range_name = get_random_name()
        self.console.ranges.create(range_name, persisted=True, labels=[label_name])
        self.log(f"Range {range_name} created with label {label_name}")

        self.console.ranges.open_explorer()
        print(f"[DEBUG_TEST] Waiting for range '{range_name}' to appear in explorer")
        range_item = self.console.ranges.get_explorer_item(range_name)
        range_item.wait_for(state="visible", timeout=10000)
        print(f"[DEBUG_TEST] Range '{range_name}' is now visible in explorer")

        self.console.ranges.favorite(range_name)
        print(f"[DEBUG_TEST] Favorited range '{range_name}', checking if it appears in toolbar")

        self.console.ranges.show_toolbar()
        toolbar_range_item = self.console.ranges.get_toolbar_item(range_name)
        toolbar_range_item.wait_for(state="visible", timeout=5000)
        print(f"[DEBUG_TEST] Range '{range_name}' is now visible in toolbar")

        assert self.console.ranges.label_exists_in_toolbar(
            range_name, label_name
        ), f"Label {label_name} should be visible in toolbar for range {range_name}"
        self.log(f"Label {label_name} visible in toolbar for range {range_name}")

        print(f"[DEBUG_TEST] BEFORE DELETE: Getting all labels for range '{range_name}'")
        labels_before = self.console.ranges.get_all_labels_in_toolbar(range_name)
        print(f"[DEBUG_TEST] Labels before delete: {labels_before}")

        self.console.labels.delete(label_name)
        print(f"[DEBUG_TEST] Label '{label_name}' deleted from modal")
        print(f"[DEBUG_TEST] Waiting 10 seconds for label to be removed from toolbar")

        print(f"[DEBUG_TEST] Checking UI state after delete:")
        label_modal = self.page.locator(".console-label__edit")
        label_modal_visible = label_modal.is_visible()
        print(f"[DEBUG_TEST] Label edit modal is visible: {label_modal_visible}")

        nav_drawer = self.page.locator(".pluto-nav-drawer").first
        nav_drawer_visible = nav_drawer.is_visible()
        print(f"[DEBUG_TEST] Navigation drawer is visible: {nav_drawer_visible}")

        ranges_header = self.page.get_by_text("Ranges", exact=True).first
        ranges_header_visible = ranges_header.is_visible()
        print(f"[DEBUG_TEST] Ranges toolbar header is visible: {ranges_header_visible}")

        all_range_items = self.page.locator(".console-range-list-item")
        range_count = all_range_items.count()
        print(f"[DEBUG_TEST] Number of ranges in toolbar: {range_count}")
        for i in range(range_count):
            range_text = all_range_items.nth(i).text_content()
            print(f"[DEBUG_TEST] Range {i+1}: {range_text}")

        print(f"[DEBUG_TEST] AFTER DELETE: Getting all labels for range '{range_name}'")
        labels_after = self.console.ranges.get_all_labels_in_toolbar(range_name)
        print(f"[DEBUG_TEST] Labels after delete: {labels_after}")

        print(f"[DEBUG_TEST] Checking UI state after waiting:")
        label_modal_visible_after = label_modal.is_visible()
        print(f"[DEBUG_TEST] Label edit modal is visible: {label_modal_visible_after}")

        nav_drawer_visible_after = nav_drawer.is_visible()
        print(f"[DEBUG_TEST] Navigation drawer is visible: {nav_drawer_visible_after}")

        ranges_header_visible_after = ranges_header.is_visible()
        print(f"[DEBUG_TEST] Ranges toolbar header is visible: {ranges_header_visible_after}")

        print(f"[DEBUG_TEST] Checking if label '{label_name}' still exists in label system:")
        label_exists_in_system = self.console.labels.exists(label_name)
        print(f"[DEBUG_TEST] Label '{label_name}' exists in label system: {label_exists_in_system}")

        assert not self.console.ranges.label_exists_in_toolbar(
            range_name, label_name
        ), f"Label {label_name} should not be visible in toolbar for range {range_name}"
        self.log(f"Label {label_name} not visible in toolbar for range {range_name}")

        self.console.ranges.delete_from_explorer(range_name)

    def test_change_label_color_syncs_with_range_toolbar(self) -> None:
        """Test that changing a label color updates the range toolbar."""
        self.log("Testing: Change label color syncs with range toolbar")

        label_name = get_random_name()
        old_color = "#F0FE00"
        self.console.labels.create(label_name, color=old_color)
        self.log(f"Label {label_name} created with color {old_color}")

        range_name = get_random_name()
        self.console.ranges.create(range_name, persisted=True, labels=[label_name])
        self.log(f"Range {range_name} created with label {label_name}")

        self.console.ranges.open_explorer()
        self.console.ranges.favorite_from_explorer(range_name)
        assert self.console.ranges.label_exists_in_toolbar(
            range_name, label_name
        ), f"Label {label_name} should be visible in toolbar for range {range_name}"
        self.log(f"Label {label_name} visible in toolbar for range {range_name}")

        new_color = "#00FF00"
        self.console.labels.change_color(name=label_name, new_color=new_color)
        self.log(f"Label {label_name} color changed to {new_color}")

        updated_color = self.console.ranges.get_label_color_in_toolbar(
            range_name, label_name
        )
        assert (
            updated_color == new_color
        ), f"Label {label_name} color in toolbar should be {new_color}"
        self.log(f"Label {label_name} color updated to {new_color} in toolbar")

        self.console.labels.delete(label_name)
        self.console.ranges.delete_from_explorer(range_name)

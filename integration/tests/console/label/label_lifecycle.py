#  Copyright 2025 Synnax Labs, Inc.
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
        """Run all label lifecycle tests."""
        self.test_open_label_modal()
        self.test_create_label()
        self.test_rename_label()
        self.test_change_label_color()
        self.test_delete_label()

    def test_open_label_modal(self) -> None:
        """Test opening the Edit Labels modal via command palette."""
        self.log("Testing: Open label modal")

        # Open the Edit Labels modal
        self.console.labels.open_edit_modal()

        # Verify the modal is visible
        modal = self.page.locator(".console-label__edit")
        assert modal.is_visible(), "Edit Labels modal should be visible"

        # Verify the header section is present (contains search and Add button)
        header = self.page.locator(".console-label__edit-header")
        assert header.is_visible(), "Edit header should be visible"

        # Verify the Add Label button is present
        add_button = self.page.locator(".console-label__add-btn")
        assert add_button.count() > 0, "Add Label button should be present"

        self.log("Successfully opened Edit Labels modal")

        # Close the modal
        self.console.labels.close_modal()
        self.log("Successfully closed Edit Labels modal")

    def test_create_label(self) -> None:
        """Test creating a new label."""
        self.log("Testing: Create label")

        label_name = "TestLabel"

        # Open the modal and create a label
        self.console.labels.open_edit_modal()
        self.console.labels.create(name=label_name)

        # Verify the label was created by checking it appears in the list
        label_item = self.page.locator(
            f".console-label__list-item:not(.console--create) input[value='{label_name}']"
        )
        assert label_item.count() > 0, f"Label '{label_name}' should appear in the list"

        self.log(f"Successfully created label: {label_name}")

        # Close the modal
        self.console.labels.close_modal()

    def test_rename_label(self) -> None:
        """Test renaming an existing label."""
        self.log("Testing: Rename label")

        old_name = "TestLabel"
        new_name = "RenamedLabel"

        # Open the modal
        self.console.labels.open_edit_modal()

        # Verify the label exists before renaming
        labels_before = self.console.labels.list_all()
        assert old_name in labels_before, f"Label '{old_name}' should exist"

        # Rename the label
        self.console.labels.rename(old_name, new_name)

        # Close and reopen to refresh the list and verify
        self.console.labels.close_modal()
        self.page.wait_for_timeout(300)
        self.console.labels.open_edit_modal()

        # Verify the label was renamed by checking the new name input exists
        label_item = self.page.locator(
            f".console-label__list-item:not(.console--create) input[value='{new_name}']"
        )
        assert label_item.count() > 0, f"Label should be renamed to '{new_name}'"

        self.log(f"Successfully renamed label from '{old_name}' to '{new_name}'")

        # Close the modal
        self.console.labels.close_modal()

    def test_change_label_color(self) -> None:
        """Test changing the color of a label."""
        self.log("Testing: Change label color")

        label_name = "RenamedLabel"
        new_color = "#FF5733"  # Orange color

        # Open the modal
        self.console.labels.open_edit_modal()

        # Change the label color
        self.console.labels.change_color(label_name, new_color)

        self.log(f"Successfully changed color of label '{label_name}' to {new_color}")

        # Close the modal
        self.console.labels.close_modal()

    def test_delete_label(self) -> None:
        """Test deleting a label (also cleans up the test label)."""
        self.log("Testing: Delete label")

        label_name = "RenamedLabel"

        # Open the modal
        self.console.labels.open_edit_modal()

        # Delete the label
        self.console.labels.delete(label_name)

        # Close and reopen to verify deletion
        self.console.labels.close_modal()
        self.page.wait_for_timeout(300)
        self.console.labels.open_edit_modal()

        # Verify the label was deleted
        label_item = self.page.locator(
            f".console-label__list-item:not(.console--create) input[value='{label_name}']"
        )
        assert label_item.count() == 0, f"Label '{label_name}' should be deleted"

        self.log(f"Successfully deleted label: {label_name}")

        # Close the modal
        self.console.labels.close_modal()

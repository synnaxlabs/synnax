#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase


class KeyboardShortcuts(ConsoleCase):
    """Test layout keyboard shortcuts."""

    def run(self) -> None:
        """Run all keyboard shortcut tests."""
        self.test_close_with_cmd_w()
        self.test_rename_with_cmd_e()
        self.test_new_tab_with_cmd_t()

    def test_close_with_cmd_w(self) -> None:
        """Should close active tab with Cmd+W."""
        self.log("test_close_with_cmd_w: Creating and closing a tab with Cmd+W")
        console = self.console

        # Create a page
        page_name = "Close Me"
        console.create_page("Line Plot", page_name)

        # Verify tab exists
        tab = console.layout.get_tab(page_name)
        assert tab.is_visible(), f"Tab '{page_name}' should be visible"

        # Close with Cmd+W
        self.page.keyboard.press("ControlOrMeta+w")
        self.page.wait_for_timeout(300)

        # Handle unsaved changes dialog if it appears
        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()
            self.page.wait_for_timeout(200)

        # Verify tab is gone
        tab_after = console.layout.get_tab(page_name)
        assert not tab_after.is_visible(), f"Tab '{page_name}' should be closed"

        self.log("test_close_with_cmd_w: PASSED")

    def test_rename_with_cmd_e(self) -> None:
        """Should enter rename mode with Cmd+E."""
        self.log("test_rename_with_cmd_e: Creating and renaming a tab with Cmd+E")
        console = self.console

        # Create a page
        original_name = "Rename Via Shortcut"
        console.create_page("Line Plot", original_name)

        # Verify tab exists
        tab = console.layout.get_tab(original_name)
        assert tab.is_visible(), f"Tab '{original_name}' should be visible"

        # Press Cmd+E to enter rename mode
        self.page.keyboard.press("ControlOrMeta+e")
        self.page.wait_for_timeout(200)

        # Type new name and confirm
        new_name = "Renamed Via Cmd E"
        self.page.keyboard.type(new_name)
        self.page.keyboard.press("Enter")
        self.page.wait_for_timeout(200)

        # Verify new name is visible
        new_tab = console.layout.get_tab(new_name)
        assert new_tab.is_visible(), f"Tab '{new_name}' should be visible after rename"

        # Clean up
        console.close_page(new_name)
        self.log("test_rename_with_cmd_e: PASSED")

    def test_new_tab_with_cmd_t(self) -> None:
        """Should create new tab/leaf with Cmd+T."""
        self.log("test_new_tab_with_cmd_t: Creating a new tab with Cmd+T")
        console = self.console

        # Count existing tabs/panes before
        plots_before = self.page.locator(".pluto-line-plot").count()

        # Press Cmd+T to create new tab
        self.page.keyboard.press("ControlOrMeta+t")
        self.page.wait_for_timeout(500)

        # Should see "New Component" or similar empty pane indicator
        new_component = self.page.get_by_text("New Component")
        assert new_component.count() > 0, "Cmd+T should create a 'New Component' pane"

        self.log("test_new_tab_with_cmd_t: PASSED")

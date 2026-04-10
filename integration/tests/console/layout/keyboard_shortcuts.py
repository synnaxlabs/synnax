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

    NAV_DRAWER_SELECTOR = (
        ".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)"
    )
    FLOATING_DRAWER_SELECTOR = ".console-nav__drawer.pluto--visible.console--hover:not(.pluto--location-bottom)"
    PINNED_DRAWER_SELECTOR = ".console-nav__drawer.pluto--visible:not(.console--hover):not(.pluto--location-bottom)"

    TOOLBAR_SHORTCUTS = [
        ("a", "arc", "Arc"),
        ("c", "channel", "Channels"),
        ("d", "device", "Devices"),
        ("r", "range", "Ranges"),
        ("s", "notification", "Statuses"),
        ("t", "task", "Tasks"),
        ("u", "user", "Users"),
        ("w", "workspace", "Workspaces"),
    ]

    def run(self) -> None:
        """Run all keyboard shortcut tests."""
        self.test_close_with_cmd_w()
        self.test_rename_with_cmd_e()
        self.test_new_tab_with_cmd_t()
        self.test_toolbar_toggle_shortcuts()

    def test_close_with_cmd_w(self) -> None:
        """Should close active tab with Cmd+W."""
        self.log("test_close_with_cmd_w: Creating and closing a tab with Cmd+W")
        console = self.console

        # Create a page
        page_name = "Close Me"
        console.workspace.create_page("Line Plot", page_name)
        self._cleanup_pages.append(page_name)

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
        console.workspace.create_page("Line Plot", original_name)
        self._cleanup_pages.append(original_name)

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
        self._cleanup_pages.remove(original_name)
        self._cleanup_pages.append(new_name)

        # Verify new name is visible
        new_tab = console.layout.get_tab(new_name)
        assert new_tab.is_visible(), f"Tab '{new_name}' should be visible after rename"

        # Clean up
        console.workspace.close_page(new_name)
        self.log("test_rename_with_cmd_e: PASSED")

    def test_new_tab_with_cmd_t(self) -> None:
        """Should create new tab/leaf with Cmd+T."""
        self.log("test_new_tab_with_cmd_t: Creating a new tab with Cmd+T")
        # Press Cmd+T to create new tab
        self.page.keyboard.press("ControlOrMeta+t")
        self.page.wait_for_timeout(500)

        # Should see "New Component" or similar empty pane indicator
        new_component = self.page.get_by_text("New Component")
        assert new_component.count() > 0, "Cmd+T should create a 'New Component' pane"

        # Close the new tab to clean up
        self.page.keyboard.press("ControlOrMeta+w")
        self.page.wait_for_timeout(300)

        self.log("test_new_tab_with_cmd_t: PASSED")

    def test_toolbar_toggle_shortcuts(self) -> None:
        """Should show, pin, and hide each toolbar via its keyboard shortcut."""
        self.console.layout.close_left_toolbar()
        floating_drawer = self.page.locator(self.FLOATING_DRAWER_SELECTOR)
        pinned_drawer = self.page.locator(self.PINNED_DRAWER_SELECTOR)
        nav_drawer = self.page.locator(self.NAV_DRAWER_SELECTOR)

        for key, icon, label in self.TOOLBAR_SHORTCUTS:
            self.log(f"test_toolbar_toggle: {label} toolbar with '{key.upper()}'")

            btn = self.page.locator(
                f"button.console-main-nav__item:has(svg.pluto-icon--{icon})"
            )

            # Single press to show floating toolbar
            self.page.keyboard.press(key)
            floating_drawer.wait_for(state="visible", timeout=5000)
            assert "pluto--selected" in (btn.get_attribute("class") or ""), (
                f"{label} nav button should be selected after single press"
            )

            # Double press to pin the toolbar
            self.page.keyboard.press(key)
            self.page.keyboard.press(key)
            pinned_drawer.wait_for(state="visible", timeout=5000)
            floating_drawer.wait_for(state="hidden", timeout=5000)
            assert "pluto--selected" in (btn.get_attribute("class") or ""), (
                f"{label} nav button should be selected after double press"
            )

            # Click the nav button to close the pinned toolbar
            btn.click()
            nav_drawer.wait_for(state="hidden", timeout=5000)

            self.log(f"test_toolbar_toggle: {label} toolbar PASSED")

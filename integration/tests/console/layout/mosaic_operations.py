#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from console.layout import LayoutClient


class MosaicOperations(ConsoleCase):
    """Test layout mosaic operations: tabs, splits, focus, new windows."""

    shared_page_name: str | None = None

    def setup(self) -> None:
        super().setup()
        self.shared_page_name = "Shared Layout Plot"
        self.console.workspace.create_page("Line Plot", self.shared_page_name)

    def run(self) -> None:
        """Run all mosaic operation tests."""
        # Tests using shared page (non-destructive)
        self.test_toggle_color_theme()
        self.test_find_tab()
        self.test_focus_via_context_menu()
        self.test_focus_via_cmd_l()
        # Tests that create their own resources (destructive)
        self.test_rename_tab()
        self.test_split_horizontal()
        self.test_split_vertical()

    def test_find_tab(self) -> None:
        """Should find a tab by name after creating a page."""
        self.log("test_find_tab: Verifying shared page tab is visible")
        console = self.console

        tab = console.layout.get_tab(self.shared_page_name)
        assert tab.is_visible(), f"Tab '{self.shared_page_name}' should be visible"

    def test_rename_tab(self) -> None:
        """Should rename a tab by double-clicking and typing new name."""
        self.log("test_rename_tab: Creating and renaming a tab")
        console = self.console

        # Create a page
        original_name = "Original Tab Name"
        console.workspace.create_page("Line Plot", original_name)

        # Rename the tab
        new_name = "Renamed Tab"
        console.layout.rename_tab(old_name=original_name, new_name=new_name)

        # Verify the new name is visible
        new_tab = console.layout.get_tab(new_name)
        assert new_tab.is_visible(), f"Tab '{new_name}' should be visible after rename"

        # Clean up
        console.workspace.close_page(new_name)

    def test_split_horizontal(self) -> None:
        """Should split a leaf horizontally via context menu."""
        self.log("test_split_horizontal: Creating two plots and splitting")
        console = self.console

        # Create two pages
        left_name = "Left Plot"
        right_name = "Right Plot"
        console.workspace.create_page("Line Plot", left_name)
        console.workspace.create_page("Line Plot", right_name)

        # Split Left Plot horizontally
        console.layout.split_horizontal(left_name)
        self.page.wait_for_timeout(300)

        # Drag Right Plot tab to the right pane
        right_tab = console.layout.get_tab(right_name)
        right_tab_box = right_tab.bounding_box()
        assert right_tab_box is not None, "Right tab should have bounding box"

        # Get viewport for drop target calculation
        viewport = self.page.viewport_size
        assert viewport is not None, "Viewport should be available"

        # Drag to right side of screen
        self.page.mouse.move(
            right_tab_box["x"] + right_tab_box["width"] / 2,
            right_tab_box["y"] + right_tab_box["height"] / 2,
        )
        self.page.mouse.down()
        self.page.mouse.move(viewport["width"] - 100, viewport["height"] // 2, steps=10)
        self.page.wait_for_timeout(200)
        self.page.mouse.up()
        self.page.wait_for_timeout(500)

        # Get pane positions via the pluto-line-plot elements
        left_pane = self.page.locator(".pluto-line-plot").first
        right_pane = self.page.locator(".pluto-line-plot").last

        left_box = left_pane.bounding_box()
        right_box = right_pane.bounding_box()

        assert left_box is not None, "Left pane should have bounding box"
        assert right_box is not None, "Right pane should have bounding box"
        assert (
            right_box["x"] > left_box["x"]
        ), f"Right pane ({right_box['x']}) should be to the right of left pane ({left_box['x']})"

        # Clean up
        console.workspace.close_page(left_name)
        console.workspace.close_page(right_name)

    def test_split_vertical(self) -> None:
        """Should split a leaf vertically via context menu."""
        self.log("test_split_vertical: Creating two plots and splitting vertically")
        console = self.console

        # Create two pages
        top_name = "Top Plot"
        bottom_name = "Bottom Plot"
        console.workspace.create_page("Line Plot", top_name)
        console.workspace.create_page("Line Plot", bottom_name)

        # Split Top Plot vertically
        console.layout.split_vertical(top_name)
        self.page.wait_for_timeout(300)

        # Drag Bottom Plot tab to the bottom pane
        bottom_tab = console.layout.get_tab(bottom_name)
        bottom_tab_box = bottom_tab.bounding_box()
        assert bottom_tab_box is not None, "Bottom tab should have bounding box"

        # Get viewport for drop target calculation
        viewport = self.page.viewport_size
        assert viewport is not None, "Viewport should be available"

        # Drag to bottom of screen
        self.page.mouse.move(
            bottom_tab_box["x"] + bottom_tab_box["width"] / 2,
            bottom_tab_box["y"] + bottom_tab_box["height"] / 2,
        )
        self.page.mouse.down()
        self.page.mouse.move(viewport["width"] // 2, viewport["height"] - 100, steps=10)
        self.page.wait_for_timeout(200)
        self.page.mouse.up()
        self.page.wait_for_timeout(500)

        # Get pane positions via the pluto-line-plot elements
        top_pane = self.page.locator(".pluto-line-plot").first
        bottom_pane = self.page.locator(".pluto-line-plot").last

        top_box = top_pane.bounding_box()
        bottom_box = bottom_pane.bounding_box()

        assert top_box is not None, "Top pane should have bounding box"
        assert bottom_box is not None, "Bottom pane should have bounding box"
        assert (
            bottom_box["y"] > top_box["y"]
        ), f"Bottom pane ({bottom_box['y']}) should be below top pane ({top_box['y']})"

        # Clean up
        console.workspace.close_page(top_name)
        console.workspace.close_page(bottom_name)

    def test_focus_via_context_menu(self) -> None:
        """Should focus a leaf via context menu, showing a modal overlay."""
        self.log("test_focus_via_context_menu: Focusing a leaf via context menu")
        console = self.console

        modal = console.layout.locator(LayoutClient.MODAL_SELECTOR)

        # Focus via context menu
        console.layout.focus(self.shared_page_name)
        console.layout.wait_for_visible(modal)

        # Unfocus by pressing Cmd+L
        console.layout.press_key("ControlOrMeta+l")
        console.layout.wait_for_hidden(modal)

    def test_focus_via_cmd_l(self) -> None:
        """Should toggle focus modal with Cmd+L keyboard shortcut."""
        self.log("test_focus_via_cmd_l: Toggling focus with Cmd+L")
        console = self.console

        modal = console.layout.locator(LayoutClient.MODAL_SELECTOR)

        # Focus with Cmd+L
        console.layout.press_key("ControlOrMeta+l")
        console.layout.wait_for_visible(modal)

        # Toggle off with Cmd+L
        console.layout.press_key("ControlOrMeta+l")
        console.layout.wait_for_hidden(modal)

    def _get_theme_class(self) -> str:
        """Get the current pluto theme class from the <html> element."""
        classes = self.console.layout.page.evaluate("""
            Array.from(document.documentElement.classList)
                .find(c => c.startsWith('pluto-theme-')) || ''
            """)
        return classes

    def test_toggle_color_theme(self) -> None:
        """Should toggle the color theme via the command palette."""
        self.log("test_toggle_color_theme: Toggling color theme")
        console = self.console

        original_theme = self._get_theme_class()
        assert original_theme, "Should have an active pluto theme class"

        console.layout.command_palette("Toggle color theme")
        console.layout.page.wait_for_function(
            f"!document.documentElement.classList.contains('{original_theme}')",
            timeout=5000,
        )
        new_theme = self._get_theme_class()
        assert (
            new_theme != original_theme
        ), f"Theme should change. Before: '{original_theme}', After: '{new_theme}'"

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

    shared_page_name: str = ""

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

    def _split_and_drag(self, direction: str) -> None:
        """Split a leaf, drag a tab to the new pane, and verify positioning.

        Args:
            direction: "horizontal" or "vertical".
        """
        console = self.console
        horizontal = direction == "horizontal"
        first_name = "Left Plot" if horizontal else "Top Plot"
        second_name = "Right Plot" if horizontal else "Bottom Plot"

        console.workspace.create_page("Line Plot", first_name)
        console.workspace.create_page("Line Plot", second_name)

        if horizontal:
            console.layout.split_horizontal(first_name)
        else:
            console.layout.split_vertical(first_name)
        self.page.wait_for_timeout(300)

        tab = console.layout.get_tab(second_name)
        tab_box = tab.bounding_box()
        assert tab_box is not None, f"{second_name} tab should have bounding box"

        mosaic = self.page.locator(".pluto-mosaic").first
        mosaic_box = mosaic.bounding_box()
        assert mosaic_box is not None, "Mosaic should have bounding box"

        self.page.mouse.move(
            tab_box["x"] + tab_box["width"] / 2,
            tab_box["y"] + tab_box["height"] / 2,
        )
        self.page.mouse.down()
        self.page.mouse.move(
            mosaic_box["x"] + mosaic_box["width"] * (0.75 if horizontal else 0.5),
            mosaic_box["y"] + mosaic_box["height"] * (0.5 if horizontal else 0.75),
            steps=10,
        )
        self.page.wait_for_timeout(200)
        self.page.mouse.up()
        self.page.wait_for_timeout(500)

        first_pane = self.page.locator(".pluto-line-plot").first
        second_pane = self.page.locator(".pluto-line-plot").last
        first_box = first_pane.bounding_box()
        second_box = second_pane.bounding_box()
        assert first_box is not None, f"{first_name} pane should have bounding box"
        assert second_box is not None, f"{second_name} pane should have bounding box"

        axis = "x" if horizontal else "y"
        assert second_box[axis] > first_box[axis], (
            f"{second_name} pane ({second_box[axis]}) should be "
            f"{'right of' if horizontal else 'below'} "
            f"{first_name} pane ({first_box[axis]})"
        )

        console.workspace.close_page(second_name)
        console.workspace.close_page(first_name)

    def test_split_horizontal(self) -> None:
        """Should split a leaf horizontally via context menu."""
        self.log("test_split_horizontal: Creating two plots and splitting")
        self._split_and_drag("horizontal")

    def test_split_vertical(self) -> None:
        """Should split a leaf vertically via context menu."""
        self.log("test_split_vertical: Creating two plots and splitting vertically")
        self._split_and_drag("vertical")

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
        classes: str = str(self.console.layout.page.evaluate("""
            Array.from(document.documentElement.classList)
                .find(c => c.startsWith('pluto-theme-')) || ''
            """))
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

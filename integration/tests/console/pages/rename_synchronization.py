#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from console.workspace import PageType
from x import random_name


class RenameSynchronization(ConsoleCase):
    """Test that renaming pages synchronizes across UI elements in both directions:
    tree → tab (via the renderer's useRetrieveObservableName flux subscription) and
    tab → tree (via the renderer's useName.onRename invoking the resource rename API).
    """

    def run(self) -> None:
        page_types: list[PageType] = ["Schematic", "Line Plot", "Log", "Table"]
        for page_type in page_types:
            self.test_tree_to_tab(page_type)
            self.test_tab_to_tree(page_type)

    def _create_page(self, page_type: PageType) -> str:
        """Create a page and register its cleanup. Returns the page name."""
        name = f"{page_type.lower().replace(' ', '_')}_{random_name()}"
        self.console.workspace.create_page(page_type, name)
        self._cleanup_pages.append(name)
        return name

    def _swap_cleanup(self, old: str, new: str) -> None:
        self._cleanup_pages.remove(old)
        self._cleanup_pages.append(new)

    def _delete(self, name: str) -> None:
        self.console.workspace.close_page(name)
        self.console.workspace.delete_page(name)
        self._cleanup_pages.remove(name)

    def _assert_tab(self, name: str) -> None:
        self.console.layout.press_escape()
        self.console.layout.get_tab(name).wait_for(state="visible", timeout=5000)

    def _assert_visualization_toolbar(self, name: str) -> None:
        self.console.layout.show_visualization_toolbar()
        title = self.console.layout.get_visualization_toolbar_title()
        assert title == name, f"Visualization Toolbar should show '{name}', got '{title}'"
        self.console.layout.hide_visualization_toolbar()

    def test_tree_to_tab(self, page_type: PageType) -> None:
        """Tree-driven rename should propagate to the open tab + visualization toolbar."""
        self.log(f"{page_type} tree → tab")
        original = self._create_page(page_type)
        renamed = f"{original}_tree_renamed"

        self.console.workspace.rename_page(original, renamed)
        self._swap_cleanup(original, renamed)

        assert self.console.workspace.page_exists(renamed)
        self._assert_tab(renamed)
        self._assert_visualization_toolbar(renamed)
        self._delete(renamed)

    def test_tab_to_tree(self, page_type: PageType) -> None:
        """Tab-driven rename should propagate to the Resources Toolbar."""
        self.log(f"{page_type} tab → tree")
        original = self._create_page(page_type)
        renamed = f"{original}_tab_renamed"

        self.console.layout.rename_tab(old_name=original, new_name=renamed)
        self._swap_cleanup(original, renamed)

        assert self.console.workspace.page_exists(renamed), (
            f"{page_type} renamed via tab should appear in Resources Toolbar"
        )
        self._assert_visualization_toolbar(renamed)
        self._delete(renamed)

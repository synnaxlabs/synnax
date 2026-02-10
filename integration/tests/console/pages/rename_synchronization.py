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
from framework.utils import get_random_name


class RenameSynchronization(ConsoleCase):
    """Test that renaming pages synchronizes across UI elements."""

    def run(self) -> None:
        page_types: list[PageType] = ["Schematic", "Line Plot", "Log", "Table"]

        for page_type in page_types:
            self.test_rename_synchronization(page_type)

    def test_rename_synchronization(self, page_type: PageType) -> None:
        """Test rename synchronization for a specific page type."""
        self.log(f"Testing {page_type} rename synchronization")
        console = self.console

        suffix = get_random_name()
        original_name = f"test_{page_type.lower().replace(' ', '_')}_{suffix}"
        new_name = f"{page_type}_renamed_{suffix}"

        self.log(f"1. Creating {page_type}: {original_name}")
        console.workspace.create_page(page_type, original_name)

        self.log(f"2. Verifying page exists in Resources Toolbar")
        assert console.workspace.page_exists(
            original_name
        ), f"{page_type} '{original_name}' should exist in Resources Toolbar after creation"

        self.log(f"3. Renaming to: {new_name}")
        console.workspace.rename_page(original_name, new_name)

        self.log("4. Verifying Resources Toolbar after rename")
        assert console.workspace.page_exists(
            new_name
        ), f"{page_type} should be renamed in Resources Toolbar"

        self.log("5. Verifying Mosaic Tab")
        console.layout.press_escape()
        console.layout.get_tab(new_name).wait_for(state="visible", timeout=5000)

        self.log("6. Verifying Visualization Toolbar")
        console.layout.show_visualization_toolbar()
        toolbar_title = console.layout.get_visualization_toolbar_title()
        assert (
            toolbar_title == new_name
        ), f"Visualization Toolbar should show '{new_name}', got '{toolbar_title}'"
        console.layout.hide_visualization_toolbar()

        self.log(f"7. Cleanup: Closing {new_name}")
        console.workspace.close_page(new_name)

        self.log(f"{page_type} rename synchronization passed")

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
import synnax as sy


class Workspace(ConsoleCase):
    """Test workspace operations."""

    def run(self) -> None:
        self._create_workspace("WorkspaceA")
        self._create_workspace("WorkspaceB")
        self.test_switch_workspaces_in_resources()
        self.test_rename_workspace()
        self.test_clear_workspace_from_selector()
        self.test_delete_workspace()
        self._select_workspace("TestSpace")
        self.log("Cleanup complete. 'TestSpace' is active.")

    def test_switch_workspaces_in_resources(self) -> None:
        """Test switching between workspaces by double-clicking in resources toolbar."""
        self.log("Testing switch workspaces in resources view")

        self._select_workspace("WorkspaceA")
        assert self.page.get_by_role("button").filter(has_text="WorkspaceA").is_visible(), \
            "WorkspaceA should be active after selection"

        self._select_workspace("WorkspaceB")
        assert self.page.get_by_role("button").filter(has_text="WorkspaceB").is_visible(), \
            "WorkspaceB should be active after selection"

        self.log("Switch workspaces in resources view test passed")

    def test_rename_workspace(self) -> None:
        """Test renaming a workspace via context menu and verify synchronization."""
        self.log("Testing rename workspace with synchronization")
        self._select_workspace("WorkspaceA")

        self.console.show_resource_toolbar("workspace")
        self.console.get_workspace_item("WorkspaceA").click(button="right")
        self.page.get_by_text("Rename", exact=True).click(timeout=5000)
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type("RenamedWorkspace")
        self.console.ENTER

        # Verify synchronization: Resources Toolbar
        assert self.console.workspace_exists("RenamedWorkspace"), \
            "Workspace should be renamed in Resources Toolbar"

        # Verify synchronization: Workspace Selector
        workspace_selector = self.page.get_by_role("button").filter(has_text="RenamedWorkspace")
        workspace_selector.wait_for(state="visible", timeout=5000)
        assert workspace_selector.is_visible(), \
            "Workspace Selector should show renamed workspace"

        # Rename back to original
        self.console.get_workspace_item("RenamedWorkspace").click(button="right")
        self.page.get_by_text("Rename", exact=True).click(timeout=5000)
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type("WorkspaceA")
        self.console.ENTER
        self.console.close_nav_drawer()

        self.log("Rename workspace with synchronization test passed")

    def test_clear_workspace_from_selector(self) -> None:
        """Test clearing workspaces from the selector (switching to no workspace)."""
        self.log("Testing clear workspace from selector")

        self._select_workspace("WorkspaceA")

        workspace_selector = self.page.get_by_role("button").filter(has_text="WorkspaceA")
        workspace_selector.click(timeout=5000)
        self.page.get_by_role("button", name="Clear").click(timeout=5000)

        self.page.get_by_role("button", name="No workspace").wait_for(
            state="visible", timeout=5000
        )
        assert self.page.get_by_role("button", name="No workspace").is_visible(), \
            "No workspace should be active after clearing"

        self.console.close_nav_drawer()

        self.log("Clear workspace from selector test passed")

    def test_delete_workspace(self) -> None:
        """Test deleting a workspace via context menu."""
        self.log("Testing delete workspace")

        self._delete_workspace("WorkspaceB")
        self._delete_workspace("WorkspaceA")

        assert not self.console.workspace_exists("WorkspaceB"), "WorkspaceB should be deleted"
        assert not self.console.workspace_exists("WorkspaceA"), "WorkspaceA should be deleted"

        self.log("Delete workspace test passed")

    def _delete_workspace(self, name: str) -> None:
        """Delete a workspace via context menu."""
        self.console.show_resource_toolbar("workspace")

        workspace = self.console.get_workspace_item(name)
        workspace.wait_for(state="visible", timeout=5000)
        workspace.click(button="right", timeout=5000)

        self.page.get_by_text("Delete", exact=True).click(timeout=5000)

        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        sy.sleep(0.2)

        self.console.close_nav_drawer()
        self.log(f"Deleted workspace '{name}'")

    def _create_workspace(self, name: str) -> None:
        """Create a workspace via command palette."""
        if self.console.workspace_exists(name):
            self.log(f"Workspace '{name}' already exists")
            return

        self.console.command_palette("Create a Workspace")
        name_input = self.page.locator("input[placeholder='Workspace Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.page.get_by_role("button", name="Create", exact=True).click(timeout=5000)
        sy.sleep(0.1)
        self.log(f"Created workspace '{name}'")

    def _select_workspace(self, name: str) -> None:
        """Select a workspace from the resources toolbar."""
        self.console.show_resource_toolbar("workspace")
        self.console.get_workspace_item(name).dblclick(timeout=5000)
        self.page.get_by_role("button").filter(has_text=name).wait_for(
            state="visible", timeout=5000
        )

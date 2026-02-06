#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase


class Workspace(ConsoleCase):
    """Test workspace operations."""

    def run(self) -> None:
        self.console.workspace.create("WorkspaceA")
        self.console.workspace.create("WorkspaceB")
        self.test_switch_workspaces_in_resources()
        self.test_rename_workspace()
        self.test_clear_workspace_from_selector()
        self.test_delete_workspace()

    def test_switch_workspaces_in_resources(self) -> None:
        """Test switching between workspaces by double-clicking in resources toolbar."""
        self.log("Testing switch workspaces in resources view")

        self.console.workspace.select("WorkspaceA")
        assert (
            self.page.get_by_role("button").filter(has_text="WorkspaceA").is_visible()
        ), "WorkspaceA should be active after selection"

        self.console.workspace.select("WorkspaceB")
        assert (
            self.page.get_by_role("button").filter(has_text="WorkspaceB").is_visible()
        ), "WorkspaceB should be active after selection"

    def test_rename_workspace(self) -> None:
        """Test renaming a workspace via context menu and verify synchronization."""
        self.log("Testing rename workspace with synchronization")
        self.console.workspace.select("WorkspaceA")
        self.console.workspace.rename(
            old_name="WorkspaceA", new_name="RenamedWorkspace"
        )

        assert self.console.workspace.exists(
            "RenamedWorkspace"
        ), "Workspace should be renamed in Resources Toolbar"

        workspace_selector = self.page.get_by_role("button").filter(
            has_text="RenamedWorkspace"
        )
        workspace_selector.wait_for(state="visible", timeout=5000)
        assert (
            workspace_selector.is_visible()
        ), "Workspace Selector should show renamed workspace"

        self.console.workspace.rename(
            old_name="RenamedWorkspace", new_name="WorkspaceA"
        )
        self.console.layout.close_left_toolbar()

    def test_clear_workspace_from_selector(self) -> None:
        """Test clearing workspaces from the selector (switching to no workspace)."""
        self.log("Testing clear workspace from selector")

        self.console.workspace.select("WorkspaceA")

        workspace_selector = self.page.get_by_role("button").filter(
            has_text="WorkspaceA"
        )
        workspace_selector.click(timeout=5000)
        self.page.get_by_role("button", name="Clear").click(timeout=5000)

        self.page.get_by_role("button", name="No workspace").wait_for(
            state="visible", timeout=5000
        )
        assert self.page.get_by_role(
            "button", name="No workspace"
        ).is_visible(), "No workspace should be active after clearing"

        self.console.layout.close_left_toolbar()

    def test_delete_workspace(self) -> None:
        """Test deleting a workspace via context menu."""
        self.log("Testing delete workspace")

        self.console.workspace.delete("WorkspaceB")
        self.console.workspace.delete("WorkspaceA")

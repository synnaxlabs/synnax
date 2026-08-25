#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import random
import shutil
import zipfile

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.base import ResourceClient
from console.layout import LayoutClient
from framework.run_dir import resolve_results_path


class ProjectClient(ResourceClient):
    """Project entity management: creation, selection, rename, delete, and
    import/export of whole projects."""

    ITEM_PREFIX = "project:"

    def __init__(self, layout: LayoutClient):
        super().__init__(layout)
        self._active_project: str | None = None

    @property
    def active(self) -> str | None:
        """Name of the project activated through this client, if any."""
        return self._active_project

    def get_item(self, name: str) -> Locator:
        """Get a project item locator from the resources toolbar.

        Note: Returns a Locator that can be waited on, even if the item isn't
        visible yet. Use exists() to check if an item is currently visible.

        :param name: Name of the project.
        :returns: Locator for the project item.
        """
        return (
            self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']")
            .filter(has_text=name)
            .first
        )

    def exists(self, name: str) -> bool:
        """Check if a project exists in the resources toolbar.

        :param name: Name of the project to check.
        :returns: True if the project exists, False otherwise.
        """
        self.layout.show_resource_toolbar("Projects")
        try:
            self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']").first.wait_for(
                state="visible", timeout=5000
            )
        except PlaywrightTimeoutError:
            return False
        return self.tree.find_by_name(self.ITEM_PREFIX, name) is not None

    def wait_for_removed(self, name: str) -> None:
        """Wait for a project to be removed from the resources toolbar.

        :param name: Name of the project to wait for removal.
        """
        if self.on_splash():
            return
        try:
            self.layout.show_resource_toolbar("Projects")
        except PlaywrightTimeoutError:
            # Deleting the active project drops the console to the Splash screen, and
            # that transition can land after the check above, taking the nav with it.
            if self.on_splash():
                return
            raise
        project_item = self.layout.page.locator(
            f"div[id^='{self.ITEM_PREFIX}']"
        ).filter(has_text=name)
        project_item.first.wait_for(state="hidden", timeout=5000)

    def import_from_directory(self, directory_path: str) -> None:
        """Import a project via the real "Import project" command flow.

        The command opens the import modal; its "Select folder" button opens a
        directory chooser fulfilled with ``directory_path`` (Playwright walks it
        and uploads each file with its webkitRelativePath set). The Core owns the
        bundle: a ``manifest.json`` ``name`` names the project; a legacy bundle
        names it after the folder. Waits for the project selector to display the
        imported project's name.
        """
        expected_name = os.path.basename(directory_path.rstrip(os.sep))
        manifest_path = os.path.join(directory_path, "manifest.json")
        if os.path.isfile(manifest_path):
            with open(manifest_path, "r", encoding="utf-8") as f:
                expected_name = json.load(f).get("name") or expected_name
        self.layout.command_palette("Import project")
        with self.layout.page.expect_file_chooser() as fc_info:
            self.layout.page.get_by_role("button", name="Select folder").click(
                timeout=5000
            )
        fc_info.value.set_files(directory_path)
        self.wait_for_active(expected_name)
        self._active_project = expected_name

    def export(self, name: str) -> str:
        """Export a project via the real Export context menu action.

        The browser runs without the File System Access pickers (see the case launch
        args), so the export falls back to a plain download, which Playwright captures.
        Saves the bundle zip under the results dir and extracts it into a real directory
        the import test can consume.
        """
        self.layout.show_resource_toolbar("Projects")
        project_item = self.get_item(name)
        project_item.wait_for(state="visible", timeout=5000)
        with self.layout.page.expect_download(timeout=10000) as download_info:
            self.ctx_menu.action(project_item, "Export")
        zip_path = resolve_results_path(f"{name}_export.zip")
        download_info.value.save_as(zip_path)

        if not self.notifications.wait_for(f"Downloaded {name}"):
            raise AssertionError(
                f"Export of project {name!r} did not emit a success notification"
            )
        export_dir = resolve_results_path(f"{name}_export")
        if os.path.isdir(export_dir):
            shutil.rmtree(export_dir)
        os.makedirs(export_dir)
        with zipfile.ZipFile(zip_path) as zf:
            zf.extractall(export_dir)

        self.notifications.close_all()
        self.layout.close_left_toolbar()
        return export_dir

    def create(self, name: str) -> bool:
        """Create a project via command palette.

        :param name: Name of the project to create.
        :returns: True if the project was created, False if it already exists.
        """
        if self.on_splash():
            self._create_from_splash(name)
            self._wait_for_app()
            return True

        if self.exists(name):
            return False

        if random.choice([True, False]):
            self.layout.command_palette("Create project")
        else:
            self.layout.close_left_toolbar()
            self._selector_trigger().click(timeout=5000)
            self.layout.page.locator("button.console-create-list-item").filter(
                has_text="New project"
            ).click(timeout=5000)

        name_input = self.layout.page.locator(
            ".console-modal input[placeholder='Name']"
        )
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.layout.page.get_by_role("button", name="Create", exact=True).click(
            timeout=5000
        )
        name_input.wait_for(state="hidden", timeout=5000)
        self.layout.show_resource_toolbar("Projects")
        self.get_item(name).wait_for(state="visible", timeout=5000)
        self.layout.close_left_toolbar()
        return True

    def select(self, name: str) -> None:
        """Select a project from the resources toolbar.

        :param name: Name of the project to select.
        """
        if self.on_splash():
            if self._select_from_splash(name):
                self._wait_for_app()
            return

        if self._active_project == name:
            return
        self.layout.show_resource_toolbar("Projects")
        self.get_item(name).dblclick(timeout=5000)
        self.wait_for_active(name)
        self._active_project = name
        self.layout.close_left_toolbar()

    def _selector_trigger(self) -> Locator:
        return self.layout.page.locator("button.console-project-selector__trigger")

    def wait_for_active(self, name: str) -> None:
        """Wait for the project selector to mark ``name`` as the active project.

        The selector trigger shows only the project's avatar, so the dialog is
        opened and the entry for ``name`` is checked for selection styling. An
        in-flight project switch (e.g. a just-finished import) remounts the
        layout and closes the dialog mid-check, so the sequence retries.
        """
        dialog = self.layout.page.locator(".console-project-selector-dialog")
        last_err: PlaywrightTimeoutError | None = None
        for _ in range(3):
            try:
                self._selector_trigger().click(timeout=5000)
                dialog.wait_for(state="visible", timeout=5000)
                dialog.get_by_placeholder("Search projects...").fill(name)
                item = (
                    dialog.get_by_role("option", selected=True)
                    .filter(has_text=name)
                    .first
                )
                item.wait_for(state="visible", timeout=5000)
                self._selector_trigger().click(timeout=5000)
                dialog.wait_for(state="hidden", timeout=5000)
                return
            except PlaywrightTimeoutError as e:
                last_err = e
                if dialog.is_visible():
                    self.layout.press_escape()
                    dialog.wait_for(state="hidden", timeout=5000)
        assert last_err is not None
        raise last_err

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename a project via context menu.

        :param old_name: Current name of the project.
        :param new_name: New name for the project.
        """
        self.layout.show_resource_toolbar("Projects")
        project = self.get_item(old_name)
        project.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(project, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        if self._active_project == old_name:
            self._active_project = new_name
        self.layout.close_left_toolbar()

    def delete(self, name: str) -> None:
        """Delete a project via context menu.

        :param name: Name of the project to delete.
        """
        if self.on_splash():
            # The resources toolbar only exists inside the app. Select the
            # project to delete so it becomes active and the app loads; the
            # delete then drops back to the Splash screen.
            if not self._select_from_splash(name):
                return
            self._wait_for_app()

        self.layout.show_resource_toolbar("Projects")

        project = self.get_item(name)
        project.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(project, "Delete")

        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        # Notifications stack over the confirmation dialog and swallow the click.
        self.notifications.close_all()
        delete_btn.click(timeout=5000)
        self.wait_for_removed(name)
        if self._active_project == name:
            self._active_project = None
        self.layout.close_left_toolbar()

    def select_bootstrap(self, name: str) -> None:
        """Select the per-test project ``name`` from the Splash screen after login.

        The project is provisioned server-side before the browser reaches the
        Splash screen, so this waits for it to appear in the Splash list,
        selects it, and returns once the main app palette is visible.

        :param name: Name of the project to select.
        """
        self.on_splash()
        self._search_splash(name)
        item = (
            self.layout.page.locator(".console-project-splash__list")
            .get_by_text(name, exact=True)
            .first
        )
        item.wait_for(state="visible", timeout=10000)
        item.click(timeout=5000)
        self._wait_for_app()
        self._active_project = name

    def _search_splash(self, name: str) -> None:
        """Filter the Splash list to ``name``: a shared core can hold enough
        projects to push it below the list's fold, where it never reports
        visible."""
        search = self.layout.page.get_by_placeholder("Search projects...")
        try:
            search.wait_for(state="visible", timeout=2000)
        except PlaywrightTimeoutError:
            return
        search.fill(name)

    def on_splash(self) -> bool:
        """Report whether the project Splash screen is showing.

        Deleting or clearing the active project drops the console to the Splash
        screen (an active project is required to use the app), so any helper
        that mutates project state may be entered from either screen. Waits for
        whichever of the Splash screen or the main app palette mounts first,
        then reports which one is visible.
        """
        self.layout.page.wait_for_selector(
            ".console-project-splash, .console-palette button",
            state="visible",
            timeout=15000,
        )
        return self.layout.page.locator(".console-project-splash").is_visible()

    def _wait_for_app(self) -> None:
        """Wait for the main app palette, the signal that a project is active."""
        self.layout.page.wait_for_selector(
            ".console-palette button", state="visible", timeout=15000
        )

    def _select_from_splash(self, name: str) -> bool:
        """Select ``name`` from the Splash project list.

        Returns False if the list is absent (no projects exist) or ``name`` is
        not in it, leaving the caller to create the project instead.
        """
        list_container = self.layout.page.locator(".console-project-splash__list")
        try:
            list_container.wait_for(state="visible", timeout=2000)
        except PlaywrightTimeoutError:
            return False
        self._search_splash(name)
        item = list_container.get_by_text(name, exact=True)
        try:
            item.first.wait_for(state="visible", timeout=3000)
        except PlaywrightTimeoutError:
            return False
        item.first.click(timeout=5000)
        self._active_project = name
        return True

    def _create_from_splash(self, name: str) -> None:
        """Create ``name`` via the Splash New Project action."""
        self.layout.page.locator(
            ".console-project-splash button.console-create-list-item"
        ).click(timeout=5000)
        name_input = self.layout.page.locator(
            ".console-modal input[placeholder='Name']"
        )
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.layout.page.get_by_role("button", name="Create", exact=True).click(
            timeout=5000
        )
        name_input.wait_for(state="hidden", timeout=5000)
        self._active_project = name

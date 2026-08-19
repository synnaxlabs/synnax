#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator

from console.base import ResourceClient
from console.layout import LayoutClient


class StatusesClient(ResourceClient):
    """Console statuses client for managing statuses via the UI."""

    TOOLBAR_ITEM_SELECTOR = ".console-status-list-item"
    EXPLORER_ITEM_SELECTOR = ".console-status__list-item"
    SEARCH_INPUT_PLACEHOLDER = "Search statuses..."

    # ── Private Helpers ──────────────────────────────────────────────────

    def _toolbar_ctx_menu_action(self, name: str, action_text: str) -> None:
        """Show the toolbar, find an item by name, and perform a context menu action."""
        self.notifications.close_all()
        self.show_toolbar()
        item = self.get_toolbar_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, action_text)

    def _select_explorer_statuses(self, names: list[str]) -> Locator:
        """Select multiple explorer statuses via their checkbox labels."""
        return self.layout.select_items(names, self.get_explorer_item)

    # ── Create ────────────────────────────────────────────────────────────

    def create(
        self,
        name: str,
        *,
        labels: list[str] | None = None,
    ) -> None:
        """Create a new status via the command palette.

        Args:
            name: The name for the new status.
            labels: Optional list of label names to attach.
        """
        self.layout.command_palette("Create status")
        modal = self.layout.page.locator(LayoutClient.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)

        name_input = modal.locator("input[placeholder='Name']")
        name_input.fill(name)

        if labels is not None:
            self.layout.select_labels(labels, modal)

        modal.get_by_role("button", name="Create").click(timeout=2000)
        modal.wait_for(state="hidden", timeout=5000)

    # ── Explorer ──────────────────────────────────────────────────────────

    def open_explorer(self) -> None:
        """Open the Status Explorer via the command palette."""
        self.layout.hide_visualization_toolbar()
        self.layout.command_palette("Open status explorer")
        self.layout.page.get_by_text("All statuses").wait_for(
            state="visible", timeout=5000
        )
        self.layout.page.locator(self.EXPLORER_ITEM_SELECTOR).first.wait_for(
            state="visible", timeout=5000
        )

    def get_explorer_item(self, name: str) -> Locator:
        """Get a status item locator from the explorer by name."""
        return self.layout.get_list_item(self.EXPLORER_ITEM_SELECTOR, name)

    def exists_in_explorer(self, name: str) -> bool:
        """Check if a status exists in the explorer."""
        return self.layout.locator_exists(self.get_explorer_item(name))

    def wait_for_removed_from_explorer(self, name: str) -> None:
        """Wait for a status to be removed from the explorer."""
        self.layout.wait_for_hidden(self.get_explorer_item(name))

    def get_labels_in_explorer(self, name: str) -> list[str]:
        """Get the label names displayed on a status item in the explorer.

        Args:
            name: The status name to look up.

        Returns:
            List of label name strings displayed as tags on the status item.
        """
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        tags = item.locator(".pluto-tag").all()
        return [tag.inner_text().strip() for tag in tags if tag.is_visible()]

    def delete_from_explorer(self, name: str) -> None:
        """Delete a single status via context menu in the explorer."""
        self.notifications.close_all()
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.layout.delete_with_confirmation(item)
        item.wait_for(state="hidden", timeout=5000)

    def delete_explorer_statuses(self, names: list[str]) -> None:
        """Delete multiple statuses via multi-select context menu in the explorer."""
        if not names:
            return
        last_item = self._select_explorer_statuses(names)
        self.notifications.close_all()
        self.ctx_menu.action(last_item, "Delete")
        self.layout.confirm_delete()
        for name in names:
            self.wait_for_removed_from_explorer(name)

    def favorite_from_explorer(self, name: str) -> None:
        """Favorite a status via context menu in the explorer."""
        self.layout.hide_visualization_toolbar()
        self.notifications.close_all()
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.open_on(item)
        menu = self.layout.page.get_by_role("menu")
        fav_btn = menu.get_by_text("Favorite", exact=True)
        unfav_btn = menu.get_by_text("Unfavorite", exact=True)
        fav_btn.or_(unfav_btn).wait_for(state="visible", timeout=2000)
        if unfav_btn.is_visible():
            self.ctx_menu.close()
            return
        self.ctx_menu.click_option("Favorite")

    def unfavorite_from_explorer(self, name: str) -> None:
        """Unfavorite a status via context menu in the explorer."""
        self.notifications.close_all()
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Unfavorite")

    # ── Explorer Search & Filter ──────────────────────────────────────────

    def enable_explorer_editing(self) -> None:
        """Enable editing mode in the explorer to show search/filter controls."""
        search_input = self.layout.page.locator(
            f"input[placeholder='{self.SEARCH_INPUT_PLACEHOLDER}']"
        )
        if search_input.is_visible():
            return
        edit_button = self.layout.page.get_by_role(
            "button", name="Enable editing", exact=True
        ).first
        edit_button.click()
        search_input.wait_for(state="visible", timeout=5000)

    def open_explorer_filter(self) -> Locator:
        """Open the label filter dropdown in the explorer.

        Returns:
            Locator for the visible filter dialog.
        """
        self.enable_explorer_editing()
        filter_button = self.layout.page.get_by_role(
            "button", name="Filter", exact=True
        ).first
        filter_button.click()
        dialog = self.layout.dialog
        dialog.wait_for(state="visible", timeout=5000)
        return dialog

    def select_explorer_variant_filter(self, variant_name: str) -> None:
        """Select a variant in the explorer's variant filter dropdown.

        Args:
            variant_name: The display name of the variant (e.g. "Error", "Success").
        """
        filter_dialog = self.open_explorer_filter()
        select_variants_trigger = filter_dialog.get_by_text("Select variants")
        select_variants_trigger.click()
        item = (
            self.layout.dialog.get_by_role("option").filter(has_text=variant_name).first
        )
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.layout.press_escape()
        self.layout.press_escape()

    def clear_explorer_variant_filter(self, variant_name: str) -> None:
        """Remove a variant from the active filter by clicking its chip close button.

        Args:
            variant_name: The display name of the variant to remove.
        """
        remove_btn = self.layout.page.get_by_role(
            "button", name=f"Remove {variant_name}", exact=True
        ).first
        remove_btn.wait_for(state="visible", timeout=5000)
        remove_btn.dispatch_event("click")
        remove_btn.wait_for(state="hidden", timeout=5000)

    def select_explorer_label_filter(self, label_name: str) -> None:
        """Select a label in the explorer's label filter dropdown.

        Args:
            label_name: The name of the label to filter by.
        """
        filter_dialog = self.open_explorer_filter()
        select_labels_trigger = filter_dialog.get_by_text("Select labels")
        select_labels_trigger.click()
        item = (
            self.layout.dialog.get_by_role("option").filter(has_text=label_name).first
        )
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.layout.press_escape()
        self.layout.press_escape()

    def clear_explorer_label_filter(self, label_name: str) -> None:
        """Remove a label from the active filter by clicking its chip close button."""
        remove_btn = self.layout.page.get_by_role(
            "button", name=f"Remove {label_name}", exact=True
        ).first
        remove_btn.wait_for(state="visible", timeout=5000)
        remove_btn.dispatch_event("click")
        remove_btn.wait_for(state="hidden", timeout=5000)

    # ── Toolbar ───────────────────────────────────────────────────────────

    def show_toolbar(self) -> None:
        """Show the statuses toolbar in the left sidebar."""
        self.layout.show_resource_toolbar("Statuses")

    def hide_toolbar(self) -> None:
        """Hide the statuses toolbar."""
        self.layout.close_left_toolbar()

    def get_toolbar_item(self, name: str) -> Locator:
        """Get a status item locator from the toolbar by name."""
        return self.layout.get_list_item(self.TOOLBAR_ITEM_SELECTOR, name)

    def exists_in_toolbar(self, name: str) -> bool:
        """Check if a status exists in the toolbar (is favorited)."""
        self.show_toolbar()
        return self.layout.locator_exists(self.get_toolbar_item(name))

    def wait_for_removed_from_toolbar(self, name: str) -> None:
        """Wait for a status to be removed from the toolbar."""
        self.show_toolbar()
        self.layout.wait_for_hidden(self.get_toolbar_item(name))

    def unfavorite_from_toolbar(self, name: str) -> None:
        """Unfavorite a status via context menu in the toolbar."""
        self._toolbar_ctx_menu_action(name, "Unfavorite")

    def rename_from_toolbar(self, old_name: str, new_name: str) -> None:
        """Rename a status via context menu and modal in the toolbar."""
        self.show_toolbar()
        item = self.get_toolbar_item(old_name)
        self.layout.rename_with_modal(item, new_name)

    def delete_from_toolbar(self, name: str) -> None:
        """Delete a status via context menu in the toolbar."""
        self.show_toolbar()
        item = self.get_toolbar_item(name)
        self.layout.delete_with_confirmation(item)

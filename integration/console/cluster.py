#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Client for the connection badge, connect-core modal, cold connection
takeover, and the pre-login Core selection surface."""

import re

from playwright.sync_api import Locator

from console.layout import LayoutClient


class ClusterClient:
    """Client for the Console's cluster connection surfaces."""

    MODAL_SELECTOR = ".console-connect-core"
    BADGE_DIALOG_SELECTOR = ".console-connection-badge__dialog"
    TROUBLE_SELECTOR = ".console-connection"

    def __init__(self, layout: LayoutClient):
        self.page = layout.page
        self.layout = layout
        self.ctx_menu = layout.ctx_menu

    # ---------------------------------------------------------------- badge

    @property
    def badge(self) -> Locator:
        """The connection badge trigger in the top nav bar."""
        return self.page.get_by_label("Connection status")

    @property
    def badge_dialog(self) -> Locator:
        return self.page.locator(self.BADGE_DIALOG_SELECTOR)

    def open_badge(self) -> None:
        """Open the connection badge dialog if it is not already open."""
        if self.badge_dialog.is_visible():
            return
        self.badge.click(timeout=5000)
        self.layout.wait_for_visible(self.badge_dialog)

    def close_badge(self) -> None:
        if not self.badge_dialog.is_visible():
            return
        self.page.keyboard.press("Escape")
        self.layout.wait_for_hidden(self.badge_dialog)

    def status_state(self) -> Locator:
        """The state label inside the open badge dialog."""
        return self.badge_dialog.locator(".console-connection-badge__state")

    def wait_for_status(self, label: str, timeout: float = 15000) -> None:
        """Wait for the open badge dialog to report ``label``.

        Labels are exact ("Connected" must not match "Disconnected").
        """
        self.status_state().filter(has_text=re.compile(f"^{label}$")).wait_for(
            state="visible", timeout=timeout
        )

    def address(self) -> str:
        """The host:port shown in the open badge dialog."""
        return self.badge_dialog.locator(
            ".console-connection-badge__address"
        ).inner_text()

    def retry_now(self) -> None:
        """Click "Retry now" in the open badge dialog."""
        self.badge_dialog.get_by_role("button", name="Retry now").click(timeout=5000)

    # ----------------------------------------------------- connect modal

    @property
    def modal(self) -> Locator:
        return self.page.locator(self.MODAL_SELECTOR)

    def fill_connect_modal(
        self,
        *,
        name: str | None = None,
        host: str | None = None,
        port: int | None = None,
    ) -> None:
        """Fill the open connect-core modal and submit it.

        The submit button reads "Save" in edit mode and "Connect" in create
        mode; both are matched. The modal dispatches and closes even when the
        connection check fails, so this returns once the modal is hidden.
        """
        self.layout.wait_for_visible(self.modal)
        if name is not None:
            self.modal.get_by_label("Name").fill(name)
        if host is not None:
            self.modal.get_by_label("Host").fill(host)
        if port is not None:
            self.modal.get_by_label("Port").fill(str(port))
        self.modal.get_by_role("button", name=re.compile("^(Save|Connect)$")).click(
            timeout=5000
        )
        self.layout.wait_for_hidden(self.modal)

    def edit_connection(self, *, host: str, port: int) -> None:
        """Repoint the active cluster through the badge's "Edit connection"."""
        self.open_badge()
        self.badge_dialog.get_by_role("button", name="Edit connection").click(
            timeout=5000
        )
        self.fill_connect_modal(host=host, port=port)

    # ------------------------------------------------------ cold takeover

    @property
    def trouble(self) -> Locator:
        """The cold-connect trouble card that replaces the workspace."""
        return self.page.locator(self.TROUBLE_SELECTOR)

    def wait_for_trouble(self, timeout: float = 20000) -> None:
        self.layout.wait_for_visible(self.trouble)
        self.trouble.get_by_role("button", name="Retry now").wait_for(
            state="visible", timeout=timeout
        )

    def edit_connection_from_trouble(self, *, host: str, port: int) -> None:
        """Repoint the cluster from the trouble card's "Edit connection"."""
        self.trouble.get_by_role("button", name="Edit connection").click(timeout=5000)
        self.fill_connect_modal(host=host, port=port)

    # ------------------------------------------- pre-login cluster list

    @property
    def core_list(self) -> Locator:
        return self.page.locator(".console-core-list__items")

    def core_item(self, name: str) -> Locator:
        return (
            self.core_list.locator(".console-core-list-item")
            .filter(has_text=name)
            .first
        )

    def add_core(self, *, name: str, host: str, port: int) -> None:
        """Add a Core through the cluster list's create action."""
        self.page.get_by_text("Add a Core").click(timeout=5000)
        self.fill_connect_modal(name=name, host=host, port=port)
        self.layout.wait_for_visible(self.core_item(name))

    def rename_core(self, *, old_name: str, new_name: str) -> None:
        self.ctx_menu.action(self.core_item(old_name), "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self.layout.wait_for_visible(self.core_item(new_name))

    def remove_core(self, name: str) -> None:
        item = self.core_item(name)
        self.ctx_menu.action(item, "Remove")
        self.layout.wait_for_hidden(item)

    def refresh_core(self, name: str) -> None:
        self.ctx_menu.action(self.core_item(name), "Refresh connection")

    def copy_core_link(self, name: str) -> str:
        self.ctx_menu.action(self.core_item(name), "Copy link")
        return str(self.page.evaluate("navigator.clipboard.readText()"))

    def select_core(self, name: str) -> None:
        """Select a Core from the list, advancing Login to credentials."""
        self.core_item(name).click(timeout=5000)

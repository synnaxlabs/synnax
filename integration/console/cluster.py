#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Client for the Core badge and the cold connection takeover."""

import re

from playwright.sync_api import Locator

from console.layout import LayoutClient


class ClusterClient:
    """Client for the Console's cluster connection surfaces."""

    BADGE_DIALOG_SELECTOR = ".console-core-badge__dialog"
    TROUBLE_SELECTOR = ".console-connection"

    def __init__(self, layout: LayoutClient):
        self.page = layout.page
        self.layout = layout

    # ---------------------------------------------------------------- badge

    @property
    def badge(self) -> Locator:
        """The Core badge trigger in the top nav bar."""
        return self.page.get_by_label("Core menu")

    @property
    def badge_dialog(self) -> Locator:
        return self.page.locator(self.BADGE_DIALOG_SELECTOR)

    def open_badge(self) -> None:
        """Open the Core badge dialog if it is not already open."""
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
        return self.badge_dialog.locator(".console-core-badge__state")

    def wait_for_status(self, label: str, *, timeout: float = 15000) -> None:
        """Wait for the open badge dialog to report ``label``.

        Labels are exact ("Connected" must not match "Disconnected").
        """
        self.status_state().filter(has_text=re.compile(f"^{label}$")).wait_for(
            state="visible", timeout=timeout
        )

    def address(self) -> str:
        """The host:port shown in the open badge dialog."""
        return self.badge_dialog.locator(".console-core-badge__address").inner_text()

    # ------------------------------------------------------ cold takeover

    @property
    def trouble(self) -> Locator:
        """The cold-connect trouble card that replaces the workspace."""
        return self.page.locator(self.TROUBLE_SELECTOR)

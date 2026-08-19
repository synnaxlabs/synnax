#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Client for the panel selector strip and panel tab movement."""

from playwright.sync_api import Locator

from console.layout import LayoutClient


class PanelsClient:
    """Client for the panel selector strip in the top nav bar."""

    STRIP_SELECTOR = ".console-panel-selector"

    def __init__(self, layout: LayoutClient):
        self.page = layout.page
        self.layout = layout
        self.ctx_menu = layout.ctx_menu

    @property
    def strip(self) -> Locator:
        """The panel selector strip."""
        return self.page.locator(self.STRIP_SELECTOR)

    @property
    def pills(self) -> Locator:
        """Every panel pill in the strip, in render order."""
        return self.strip.get_by_role("tab")

    def pill(self, name: str) -> Locator:
        """The pill for the panel named ``name``."""
        return self.pills.filter(has_text=name).first

    def names(self) -> list[str]:
        """The strip's panel names in render order."""
        return self.pills.all_inner_texts()

    def selected_name(self) -> str:
        """The name of the selected panel."""
        return self.strip.locator("[role='tab'][aria-selected='true']").inner_text()

    @property
    def create_button(self) -> Locator:
        """The strip's create button, also a drop target for mosaic tabs."""
        return self.strip.locator("button:has(.pluto-icon--add)").first

    def create(self) -> None:
        """Create a panel and wait for its pill to appear selected."""
        count = self.pills.count()
        self.create_button.click()
        self.layout.wait_for_visible(self.pills.nth(count))

    def select(self, name: str) -> None:
        """Select the panel named ``name`` and wait for the selection."""
        self.layout.click(self.pill(name))
        self.layout.wait_for_visible(
            self.pill(name).and_(self.page.locator("[aria-selected='true']"))
        )

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename a panel through the strip's context menu."""
        self.ctx_menu.action(self.pill(old_name), "Rename")
        editable = self.page.locator(".pluto-text--editable[contenteditable='true']")
        editable.wait_for(state="visible", timeout=5000)
        editable.fill(new_name)
        self.page.keyboard.press("Enter")
        self.layout.wait_for_hidden(editable)
        self.layout.wait_for_visible(self.pill(new_name))

    def delete(self, name: str) -> None:
        """Delete a panel through the strip's context menu, confirming."""
        pill = self.pill(name)
        self.ctx_menu.action(pill, "Delete")
        self.layout.confirm_delete()
        self.layout.wait_for_hidden(pill)

    def reorder(self, name: str, target: str) -> None:
        """Drag the pill for ``name`` onto the pill for ``target``."""
        self.pill(name).drag_to(self.pill(target))

    def _wait_followed(self, tab_name: str, panel_name: str) -> None:
        """Every move follows the tab: its destination panel becomes selected
        and the tab is selected inside it."""
        self.layout.wait_for_visible(
            self.pill(panel_name).and_(self.page.locator("[aria-selected='true']"))
        )
        self.layout.wait_for_tab(tab_name)

    def move_tab_to_panel(self, tab_name: str, panel_name: str | None) -> None:
        """Move a mosaic tab to a panel through the tab menu's picker.

        :param tab_name: Name of the mosaic tab to move.
        :param panel_name: Target panel name, or None for the picker's
            "New panel" entry, which mints a panel named after the tab.
        """
        tab = self.layout.get_tab(tab_name)
        self.ctx_menu.action(tab, "Move to panel")
        target = panel_name if panel_name is not None else "New panel"
        option = self.layout.dialog.get_by_role("option").filter(has_text=target)
        option.first.click()
        self._wait_followed(tab_name, panel_name if panel_name else tab_name)

    def drag_tab_to_pill(self, tab_name: str, panel_name: str) -> None:
        """Drag a mosaic tab onto a panel pill, moving the tab there."""
        self.layout.get_tab(tab_name).drag_to(self.pill(panel_name))
        self._wait_followed(tab_name, panel_name)

    def drag_tab_to_create_button(self, tab_name: str) -> None:
        """Drag a mosaic tab onto the create button, minting a panel for it."""
        self.layout.get_tab(tab_name).drag_to(self.create_button)
        self._wait_followed(tab_name, tab_name)

    def reload_console(self) -> None:
        """Reload the Console from the strip's context menu and wait for it."""
        self.ctx_menu.open_on(self.pills.first)
        with self.page.expect_navigation(wait_until="load", timeout=30000):
            self.ctx_menu.click_option("Reload Console")
        self.layout.wait_for_visible(self.pills.first)
        # Selection reconciles after hydration and overrides any select
        # dispatched before it lands, so wait for a selected pill.
        self.layout.wait_for_visible(
            self.strip.locator("[role='tab'][aria-selected='true']")
        )

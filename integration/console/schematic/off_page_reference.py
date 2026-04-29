#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from console.schematic.symbol import Symbol


class OffPageReference(Symbol):
    """Schematic off-page reference symbol for cross-schematic navigation."""

    _symbol_group = "General"

    def __init__(
        self,
        *,
        label: str,
        page_name: str,
    ):
        """Initialize an off-page reference symbol.

        Args:
            label: Display label for the symbol
            page_name: Name of the target schematic page to navigate to
        """
        super().__init__(label, symbol_type="Off Page", rotatable=False)
        self._page_name = page_name

    def set_label(self, label: str) -> None:
        """Set the label via the Properties panel.

        The off-page reference form has a flat layout with the Label field
        directly in Properties (no "Style" tab like other symbols).
        """
        self.click()
        selected_node = self.page.locator(".react-flow__node.selected")
        selected_node.wait_for(state="visible", timeout=5000)
        self.layout.show_visualization_toolbar()
        self.page.get_by_text("Properties").click()
        self.layout.fill_input_field("Label", label)
        self.label = label

    def _apply_properties(self) -> None:
        self.set_properties(page_name=self._page_name)

    def set_properties(
        self,
        channel_name: str | None = None,
        page_name: str | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Set off-page reference properties.

        Args:
            channel_name: Unused, kept for interface compatibility.
            page_name: Name of the target schematic page to link to.

        Returns:
            Dictionary of applied properties.
        """
        self.click()
        applied_properties: dict[str, Any] = {}

        self.page.get_by_text("Properties").click()

        if page_name is not None:
            # Use specific CSS class selector — click_btn("Page") matches
            # "Off Page" text elsewhere on the page.
            page_trigger = self.page.locator(
                ".pluto-symbol-form__page-field button"
            ).first
            page_trigger.wait_for(state="visible", timeout=5000)
            page_trigger.click()
            self.layout.select_from_dropdown(page_name)
            applied_properties["page_name"] = page_name

        return applied_properties

    def double_click(self) -> None:
        """Double-click the off-page reference to navigate to the linked page.

        Navigation only works when edit mode is disabled.
        """
        self._disable_edit_mode()
        self.locator.dblclick()

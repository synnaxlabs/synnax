#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import ABC, abstractmethod
from typing import Any

import synnax as sy
from playwright.sync_api import FloatRect, Locator, Page

from ..layout import LayoutClient
from ..notifications import NotificationsClient

""" Symbol Box helpers """


def box_left(box: FloatRect) -> float:
    """Left edge x-coordinate."""
    return box["x"]


def box_right(box: FloatRect) -> float:
    """Right edge x-coordinate."""
    return box["x"] + box["width"]


def box_top(box: FloatRect) -> float:
    """Top edge y-coordinate."""
    return box["y"]


def box_bottom(box: FloatRect) -> float:
    """Bottom edge y-coordinate."""
    return box["y"] + box["height"]


def box_center_x(box: FloatRect) -> float:
    """Center x-coordinate."""
    return box["x"] + box["width"] / 2


def box_center_y(box: FloatRect) -> float:
    """Center y-coordinate."""
    return box["y"] + box["height"] / 2


class Symbol(ABC):
    """Base class for all schematic symbols"""

    page: Page
    layout: LayoutClient
    notifications: NotificationsClient
    locator: Locator
    symbol_id: str | None
    label: str
    rotatable: bool
    _symbol_type: str
    _symbol_group: str | None = None

    def __init__(self, label: str, symbol_type: str, rotatable: bool = False):
        """Initialize a symbol with configuration parameters.

        The symbol is not yet attached to a schematic. Call schematic.create_symbol(symbol)
        to add it to the schematic.

        Args:
            label: Display label for the symbol
            symbol_type: The type of symbol (e.g., "Valve", "Button", "Three Way")
            rotatable: Whether the symbol can be rotated (default: False)
        """
        if label.strip() == "":
            raise ValueError("Label cannot be empty")

        self.label = label
        self._symbol_type = symbol_type
        self.rotatable = rotatable
        self.symbol_id = None

    def create(self, layout: LayoutClient, notifications: NotificationsClient) -> None:
        """Attach this symbol to a schematic (called by Schematic.create_symbol).

        This method adds the symbol to the schematic UI using the SymbolToolbar factory.

        Args:
            layout: LayoutClient for UI interactions
            notifications: NotificationsClient for closing notifications
        """
        from .symbol_toolbar import SymbolToolbar

        self.page = layout.page
        self.layout = layout
        self.notifications = notifications

        toolbar = SymbolToolbar(self.layout, self.notifications)
        self.symbol_id = toolbar.add_symbol(self._symbol_type, self._symbol_group)
        self.locator = self.page.get_by_test_id(self.symbol_id)
        self.set_label(self.label)
        self._apply_properties()

    @abstractmethod
    def _apply_properties(self) -> None:
        """Apply symbol-specific configuration after being added to schematic.

        This method should call set_properties() with the symbol's configuration.
        """
        pass

    def _disable_edit_mode(self) -> None:
        self.notifications.close_all()
        edit_off_icon = self.page.get_by_label("pluto-icon--edit-off")
        if edit_off_icon.count() > 0:
            edit_off_icon.click()

    def _enable_edit_mode(self) -> None:
        self.notifications.close_all()
        enable_editing_link = self.page.get_by_text("enable editing", exact=False)
        if enable_editing_link.count() > 0:
            enable_editing_link.click()
            self.page.get_by_label("pluto-icon--edit-off").wait_for(
                state="visible", timeout=2000
            )
            return
        edit_icon = self.page.get_by_label("pluto-icon--edit")
        if edit_icon.count() > 0:
            edit_icon.click()

    def click(self, sleep: int = 100) -> None:
        """Click the symbol to select it.

        Args:
            sleep: Time in milliseconds to wait after clicking. Buffer for network delays and slow animations.
        """

        self.layout.click(self.locator, sleep=sleep)

    def meta_click(self, sleep: int = 0) -> None:
        """
        Click the symbol with the platform-appropriate modifier key (Cmd/Ctrl) held down.

        Args:
            sleep: Time in milliseconds to wait after clicking. Buffer for network delays and slow animations.
        """

        self.layout.meta_click(self.locator, sleep=sleep)

    def set_label(self, label: str) -> None:
        self.click()
        self.page.get_by_text("Style").click(force=True)
        self.layout.fill_input_field("Label", label)
        self.label = label

    @abstractmethod
    def set_properties(
        self,
        channel_name: str | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """
        Set symbol properties. Must be implemented by all child classes.

        Args:
            channel_name: Optional channel name to set
            **kwargs: Additional properties specific to each symbol type

        Returns:
            Dictionary of applied properties
        """
        pass

    def set_channel(self, *, input_field: str, channel_name: str) -> None:
        if channel_name is not None:
            self.layout.click_btn(input_field)
            self.layout.select_from_dropdown(channel_name, "Search")

    def set_value(self, value: float) -> None:
        """Set the symbol's value if applicable. Default implementation does nothing."""
        pass

    def press(self, sleep: int = 100) -> None:
        """Press/activate the symbol if applicable. Default implementation does nothing.

        Args:
            sleep: Time in milliseconds to wait after pressing. Buffer for network delays and slow animations.
        """
        pass

    def move(self, *, delta_x: int, delta_y: int) -> None:
        """Move the symbol by the specified number of pixels using drag"""
        pos = self.position
        start_x = box_center_x(pos)
        start_y = box_center_y(pos)
        target_x = start_x + delta_x
        target_y = start_y + delta_y

        # Move
        self.page.mouse.move(start_x, start_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()
        sy.sleep(0.1)  # CI flakiness

    @property
    def position(self) -> FloatRect:
        """Get the symbol's bounding box for alignment checks.

        Raises:
            RuntimeError: If the symbol's bounding box cannot be retrieved
        """
        box = self.locator.bounding_box()
        if not box:
            raise RuntimeError(
                f"Could not get bounding box for symbol {self.symbol_id}"
            )
        return box

    def delete(self) -> None:
        self.click()
        self.layout.press_delete()

    def toggle_absolute_control(self) -> None:
        """Toggle absolute control authority for this symbol by clicking its control chip button."""
        # Locate the control chip button within this specific symbol's container
        control_chip = self.locator.locator(".pluto-control-chip").first
        self.layout.click(control_chip)

    def get_properties(self, tab: str = "Symbols") -> dict[str, Any]:
        """
        Get the current properties of the symbol.

        Base implementation opens the Properties panel and optionally selects a tab.
        Subclasses should call super().get_properties(tab="TabName") to open the correct tab.

        Args:
            tab: Optional tab name to select (e.g., "Control", "Telemetry", "Style")

        Returns:
            Empty dict - subclasses should populate with actual properties
        """
        self.click()
        self.page.get_by_text("Properties").click()
        if tab:
            self.page.get_by_text(tab).last.click()
        return {}

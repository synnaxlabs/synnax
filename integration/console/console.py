#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import platform
import random
import re
import time
from collections.abc import Generator
from contextlib import contextmanager
from typing import Any, Literal

import synnax as sy
from playwright.sync_api import Locator, Page

from .channels import ChannelClient
from .labels import LabelClient
from .layout import LayoutClient

# Define literal types for page creation
PageType = Literal[
    "Control Sequence",
    "Line Plot",
    "Schematic",
    "Log",
    "Table",
    "NI Analog Read Task",
    "NI Analog Write Task",
    "NI Counter Read Task",
    "NI Digital Read Task",
    "NI Digital Write Task",
    "LabJack Read Task",
    "LabJack Write Task",
    "OPC UA Read Task",
    "OPC UA Write Task",
]


class Console:
    """
    Console UI automation interface.

    Provides utility methods for interacting with the Synnax Console application
    via Playwright, including page management, keyboard shortcuts, form interactions,
    and element clicking helpers.
    """

    channels: ChannelClient
    labels: LabelClient
    layout: LayoutClient
    page: Page

    def __init__(self, page: Page):
        # Playwright
        self.page = page
        self.channels = ChannelClient(page, self)
        self.labels = LabelClient(page, self)
        self.layout = LayoutClient(page, self)

    def command_palette(self, command: str) -> None:
        """Execute a command via the command palette"""
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(f">{command}", timeout=2000)
        self.click(command, timeout=2000)

    @property
    def ESCAPE(self) -> None:
        self.page.keyboard.press("Escape")

    @property
    def ENTER(self) -> None:
        self.page.keyboard.press("Enter")

    @property
    def META_ENTER(self) -> None:
        self.page.keyboard.press("ControlOrMeta+Enter")

    @property
    def DELETE(self) -> None:
        self.page.keyboard.press("Delete")

    @property
    def MODAL_OPEN(self) -> bool:
        return (
            self.page.locator(
                "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
            ).count()
            > 0
        )

    def select_from_dropdown(self, text: str, placeholder: str | None = None) -> None:
        """Select an item from an open dropdown."""
        sy.sleep(0.3)
        target_item = f".pluto-list__item:not(.pluto-tree__item):has-text('{text}')"

        if placeholder is not None:
            search_input = self.page.locator(f"input[placeholder*='{placeholder}']")
            if search_input.count() > 0:
                search_input.wait_for(state="attached", timeout=5000)
                search_input.fill(text)
                sy.sleep(0.1)

        for attempt in range(10):
            try:
                self.page.wait_for_selector(target_item, timeout=100)
                item = self.page.locator(target_item).first
                item.wait_for(state="attached", timeout=5000)
                item.click()
                return
            except Exception:
                continue

        items = self.page.locator(
            ".pluto-list__item:not(.pluto-tree__item)"
        ).all_text_contents()
        raise RuntimeError(
            f"Could not find item '{text}' in dropdown. Available items: {items}"
        )

    def create_page(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """
        Public method for creating a new page in one of two ways:
        - By the New Page (+) button
        - By the command palette
        """

        if random.random() < 0.5:
            page_tab, page_id = self._create_page_by_new_page_button(
                page_type, page_name
            )
        else:
            page_tab, page_id = self._create_page_by_command_palette(
                page_type, page_name
            )

        return page_tab, page_id

    def _create_page_by_new_page_button(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Create a new page via the New Page (+) button."""

        self.page.locator(".pluto-icon--add").first.click()  # (+)
        self.page.get_by_role("button", name=page_type).first.click()
        page_tab, page_id = self._handle_new_page(page_type, page_name)

        return page_tab, page_id

    def _create_page_by_command_palette(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Create a new page via command palette"""

        # Handle "a" vs "an" article for proper command matching
        vowels = ["A", "E", "I", "O", "U"]
        # Special case for "NI" (en-eye)
        article = (
            "an"
            if page_type[0].upper() in vowels or page_type.startswith("NI")
            else "a"
        )
        page_command = f"Create {article} {page_type}"
        self.command_palette(page_command)
        page_tab, page_id = self._handle_new_page(page_type, page_name)

        return page_tab, page_id

    def _handle_new_page(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Handle the new page creation"""
        if self.MODAL_OPEN:
            page_name = page_name or page_type
            self.page.get_by_role("textbox", name="Name").fill(page_name)
            self.page.get_by_role("textbox", name="Name").press("ControlOrMeta+Enter")

        page_tab = (
            self.page.locator("div")
            .filter(has_text=re.compile(f"^{re.escape(page_type)}$"))
            .first
        )
        page_id = page_tab.inner_text().strip()

        # If page name provided, rename the page
        if page_name is not None:
            page_tab.dblclick()
            self.page.get_by_text(page_type).first.fill(page_name)
            self.page.keyboard.press("Enter")  # Confirm the change
            page_id = page_name  # Update page_id to the custom name

        return page_tab, page_id

    def close_page(self, page_name: str) -> None:
        """
        Close a page by name.
        Ignore unsaved changes.
        """
        tab = (
            self.page.locator("div")
            .filter(has_text=re.compile(f"^{re.escape(page_name)}$"))
            .first
        )
        tab.get_by_label("pluto-tabs__close").click()

        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()

    def check_for_error_screen(self) -> None:
        """Checks for 'Something went wrong' text and clicks 'Try again' if found"""
        sy.sleep(0.3)
        if self.page.get_by_text("Something went wrong").is_visible():
            sy.sleep(0.2)
            self.page.get_by_text("Try again").click()
            sy.sleep(0.2)

    def check_for_notifications(
        self, timeout: sy.CrudeTimeSpan = 0.2
    ) -> list[dict[str, Any]]:
        """
        Check for notifications in the bottom right corner.
        Polls every 100ms until notifications are found or timeout is reached.

        :param timeout: Maximum time to wait for notifications in seconds (default: 0.2)
        :returns: List of notification dictionaries with details
        """
        start_time = time.time()
        poll_interval = 50  # ms

        while time.time() - start_time < timeout:
            notifications = []
            notification_elements = self.page.locator(".pluto-notification").all()

            # If we found notifications, parse and return them
            if len(notification_elements) > 0:
                for notification in notification_elements:
                    try:
                        # Extract notification details
                        notification_data = {}

                        # Get the count (e.g., "x1")
                        count_element = notification.locator(".pluto-text--small").first
                        if count_element.count() > 0:
                            count_text = count_element.inner_text().strip()
                            notification_data["count"] = count_text

                        # Get the timestamp
                        time_element = notification.locator(".pluto-notification__time")
                        if time_element.count() > 0:
                            timestamp = time_element.inner_text().strip()
                            notification_data["timestamp"] = timestamp

                        # Get the main message
                        message_element = notification.locator(
                            ".pluto-notification__message"
                        )
                        if message_element.count() > 0:
                            message = message_element.inner_text().strip()
                            notification_data["message"] = message

                        # Get the description
                        description_element = notification.locator(
                            ".pluto-notification__description"
                        )
                        if description_element.count() > 0:
                            description = description_element.inner_text().strip()
                            notification_data["description"] = description

                        # Determine notification type based on icon or styling
                        error_icon = notification.locator("svg[color*='error']")
                        if error_icon.count() > 0:
                            notification_data["type"] = "error"
                        else:
                            notification_data["type"] = "info"

                        notifications.append(notification_data)

                    except Exception as e:
                        raise RuntimeError(f"Error parsing notification: {e}")

                return notifications

            sy.sleep(poll_interval / 1000)

        # Timeout reached, return empty list
        return []

    def close_notification(self, notification_index: int = 0) -> bool:
        """
        Close a notification by clicking its close button.

        :param notification_index: Index of the notification to close (0 for first)
        :returns: True if notification was closed, False if not found
        """
        try:
            notification_elements = self.page.locator(".pluto-notification").all()
            if notification_index >= len(notification_elements):
                return False

            notification = notification_elements[notification_index]
            close_button = notification.locator(".pluto-notification__silence")

            if close_button.count() > 0:
                close_button.wait_for(state="attached", timeout=500)
                close_button.click()
                notification.wait_for(state="hidden", timeout=2000)
                return True
            return False

        except Exception:
            return False

    def close_all_notifications(self) -> int:
        """
        Close all visible notifications.

        :returns: Number of notifications closed
        """
        closed_count = 0
        max_attempts = 10

        for _ in range(max_attempts):
            notification_elements = self.page.locator(".pluto-notification").all()
            if len(notification_elements) == 0:
                break

            if self.close_notification(0):
                closed_count += 1
            else:
                sy.sleep(0.1)

        # Small sleep to ensure any closing animations complete
        if closed_count > 0:
            sy.sleep(0.1)

        return closed_count

    def screenshot(self, name: str | None = None) -> None:
        """Take a screenshot of the entire console page."""
        results_dir = os.path.join(os.path.dirname(__file__), "..", "tests", "results")
        os.makedirs(results_dir, exist_ok=True)
        if name is None:
            name = "console.png"
        else:
            if not name.endswith(".png"):
                name = name + ".png"

        path = os.path.join(results_dir, name)
        self.page.screenshot(
            path=path, full_page=True, animations="disabled", type="png"
        )

    def click_btn(self, button_label: str) -> None:
        """Click a button by label."""
        button = (
            self.page.locator(f"text={button_label}")
            .locator("..")
            .locator("button")
            .first
        )
        button.wait_for(state="attached", timeout=300)
        button.click()

    def get_toggle(self, toggle_label: str) -> bool:
        """Get the value of a toggle by label."""
        toggle = (
            self.page.locator(f"text={toggle_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        return toggle.is_checked()

    def click_checkbox(self, checkbox_label: str) -> None:
        """Click a checkbox by label."""
        checkbox = (
            self.page.locator(f"text={checkbox_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        checkbox.wait_for(state="attached", timeout=300)
        checkbox.click()

    def fill_input_field(self, input_label: str, value: str) -> None:
        """Fill an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        input_field.wait_for(state="attached", timeout=300)
        input_field.fill(value)

    def get_input_field(self, input_label: str) -> str:
        """Get the value of an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        input_field.wait_for(state="attached", timeout=400)
        return input_field.input_value(timeout=200)

    def get_dropdown_value(self, dropdown_label: str) -> str:
        """Get the current value of a dropdown by label."""
        dropdown_button = (
            self.page.locator(f"text={dropdown_label}")
            .locator("..")
            .locator("button")
            .first
        )
        dropdown_button.wait_for(state="attached", timeout=300)
        return dropdown_button.inner_text().strip()

    def get_selected_button(self, button_options: list[str]) -> str:
        """Get the currently selected button from a button group (no label)."""
        for option in button_options:
            button = self.page.get_by_text(option).first
            if button.count() > 0:
                button.wait_for(state="attached", timeout=300)
                class_name = button.get_attribute("class") or ""
                if "pluto-btn--filled" in class_name:
                    return option

        raise RuntimeError(f"No selected button found from options: {button_options}")

    def click(
        self, selector: str | Locator, timeout: int = 500, sleep: int = 100
    ) -> None:
        """
        Click an element by text selector or Locator.

        When clicking a Locator, uses bring_to_front wrapper for robustness.

        Args:
            selector: Either a text string to search for, or a Playwright Locator
            timeout: Maximum time in milliseconds to wait for actionability.
            sleep: Time in milliseconds to wait after clicking. Buffer for network delays and slow animations.
        """
        if isinstance(selector, str):
            element = self.page.get_by_text(selector, exact=True).first
            element.click(timeout=timeout)
        else:
            with self.bring_to_front(selector) as el:
                el.click(timeout=timeout)

        sy.sleep(sleep / 1000)

    def meta_click(
        self, selector: str | Locator, timeout: int = 500, sleep: int = 100
    ) -> None:
        """
        Click an element with platform-appropriate modifier key (Cmd on Mac, Ctrl elsewhere) held.

        When clicking a Locator, uses bring_to_front wrapper for robustness.

        Args:
            selector: Either a text string to search for, or a Playwright Locator
            timeout: Maximum time in milliseconds to wait for actionability.
            sleep: Time in milliseconds to wait after clicking. Buffer for network delays and slow animations.
        """

        modifier = "Meta" if platform.system() == "Darwin" else "Control"

        if isinstance(selector, str):
            element = self.page.get_by_text(selector, exact=True).first
            self.page.keyboard.down(modifier)
            element.click(timeout=timeout)
            self.page.keyboard.up(modifier)
        else:
            with self.bring_to_front(selector) as el:
                self.page.keyboard.down(modifier)
                el.click(timeout=timeout)
                self.page.keyboard.up(modifier)

        sy.sleep(sleep / 1000)

    def check_for_modal(self) -> bool:
        """Check for a modal"""
        return (
            self.page.locator(
                "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
            ).count()
            > 0
        )

    @contextmanager
    def bring_to_front(self, element: Locator) -> Generator[Locator, None, None]:
        """
        Context manager that temporarily brings an element to the front by setting z-index.

        This ensures the element is clickable even if other elements are overlapping it.
        The original z-index is restored when exiting the context.

        Args:
            element: The Playwright Locator to bring to front

        Yields:
            The same element, now with z-index set to 9999

        Example:
            with console.bring_to_front(element) as el:
                el.click(timeout=500)
        """
        original_z_index = element.evaluate("element => element.style.zIndex || 'auto'")
        element.evaluate("element => element.style.zIndex = '9999'")
        try:
            yield element
        finally:
            element.evaluate(f"element => element.style.zIndex = '{original_z_index}'")

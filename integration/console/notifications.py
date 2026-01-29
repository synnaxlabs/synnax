#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from typing import TYPE_CHECKING, Any

import synnax as sy
from playwright.sync_api import Page, TimeoutError as PlaywrightTimeoutError

if TYPE_CHECKING:
    from .console import Console


class NotificationsClient:
    """Notifications management for Console UI automation."""

    def __init__(self, page: Page, console: "Console"):
        """Initialize the notifications client.

        Args:
            page: Playwright Page instance
            console: Console instance for UI interactions
        """
        self.page = page
        self.console = console

    def check(self, timeout: sy.CrudeTimeSpan = 0.2) -> list[dict[str, Any]]:
        """Check for notifications in the bottom right corner.

        Polls every 100ms until notifications are found or timeout is reached.

        Args:
            timeout: Maximum time to wait for notifications in seconds (default: 0.2)

        Returns:
            List of notification dictionaries with details
        """
        start_time = time.time()
        poll_interval = 50  # ms

        while time.time() - start_time < timeout:
            notifications = []
            notification_elements = self.page.locator(".pluto-notification").all()

            if len(notification_elements) > 0:
                for notification in notification_elements:
                    try:
                        notification_data = {}

                        count_element = notification.locator(".pluto-text--small").first
                        if count_element.count() > 0:
                            count_text = count_element.inner_text().strip()
                            notification_data["count"] = count_text

                        time_element = notification.locator(".pluto-notification__time")
                        if time_element.count() > 0:
                            timestamp = time_element.inner_text().strip()
                            notification_data["timestamp"] = timestamp

                        message_element = notification.locator(
                            ".pluto-notification__message"
                        )
                        if message_element.count() > 0:
                            message = message_element.inner_text().strip()
                            notification_data["message"] = message

                        description_element = notification.locator(
                            ".pluto-notification__description"
                        )
                        if description_element.count() > 0:
                            description = description_element.inner_text().strip()
                            notification_data["description"] = description

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

        return []

    def close(self, notification_index: int = 0) -> bool:
        """Close a notification by clicking its close button.

        Args:
            notification_index: Index of the notification to close (0 for first)

        Returns:
            True if notification was closed, False if not found
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

    def close_all(self) -> int:
        """Close all visible notifications.

        Returns:
            Number of notifications closed
        """
        closed_count = 0
        max_attempts = 10

        for _ in range(max_attempts):
            notification_elements = self.page.locator(".pluto-notification").all()
            if len(notification_elements) == 0:
                break

            if self.close(0):
                closed_count += 1
            else:
                sy.sleep(0.1)

        if closed_count > 0:
            sy.sleep(0.1)

        return closed_count

    def close_connection(self) -> bool:
        """Close the 'Connected to...' notification if present.

        Returns:
            True if notification was found and close was triggered, False otherwise.
        """
        notification = self.page.locator(".pluto-notification:has-text('Connected to')")
        if notification.count() == 0:
            return False

        close_btn = notification.locator(".pluto-notification__silence")
        if close_btn.count() > 0:
            close_btn.click(force=True)
            return True
        return False

    def wait_for(self, text: str) -> bool:
        """Wait for a notification containing specific text to appear.

        Args:
            text: Text to search for in the notification.
            timeout: Maximum time to wait in milliseconds.

        Returns:
            True if notification was found, False if timeout.
        """
        notification = self.page.locator(f".pluto-notification:has-text('{text}')")
        try:
            notification.wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from playwright.sync_api import Locator, Page, expect
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

import synnax as sy


class NotificationsClient:
    """Notifications management for Console UI automation."""

    def __init__(self, page: Page):
        self.page = page

    def _all(self) -> Locator:
        """All visible notifications. Each notification carries role=status."""
        return self.page.get_by_role("status")

    def check(
        self, timeout: sy.CrudeTimeSpan = 200 * sy.TimeSpan.MILLISECOND
    ) -> list[dict[str, Any]]:
        """Check for notifications in the bottom right corner.

        Polls every 50ms until notifications are found or timeout is reached.

        :param timeout: Maximum time to wait for notifications.
        :returns: List of notification dictionaries with details.
        """
        timer = sy.Timer()
        timeout_span = sy.TimeSpan.from_seconds(timeout)
        poll_interval = 50 * sy.TimeSpan.MILLISECOND

        while timer.elapsed() < timeout_span:
            notifications = []
            notification_elements = self._all().all()

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

            sy.sleep(poll_interval)

        return []

    def close(self, notification_index: int = 0) -> bool:
        """Close a notification by clicking its Silence button.

        :param notification_index: Index of the notification to close (0 for
            first).
        :returns: True if the notification was closed, False if not found.
        """
        try:
            notifications = self._all()
            if notifications.count() <= notification_index:
                return False

            notification = notifications.nth(notification_index)
            close_button = notification.get_by_role(
                "button", name="Silence", exact=True
            )

            if close_button.count() > 0:
                close_button.first.dispatch_event("click")
                notification.wait_for(state="hidden", timeout=2000)
                return True
            return False

        except PlaywrightTimeoutError:
            return False

    def close_all(self) -> int:
        """Silence every visible notification at once and wait for them to clear.

        :returns: Number of notifications silenced.
        """
        buttons = self._all().get_by_role("button", name="Silence", exact=True)
        count = buttons.count()
        # Last to first: a silenced toast unmounts and shifts the indexes after it.
        for i in reversed(range(count)):
            buttons.nth(i).dispatch_event("click")
        if count > 0:
            try:
                expect(buttons).to_have_count(0, timeout=2000)
            except AssertionError:
                pass
        return count

    def close_connection(self) -> bool:
        """Close the 'Connected to...' notification if present.

        :returns: True if the notification was found and close was triggered,
            False otherwise.
        """
        notification = self._all().filter(has_text="Connected to")
        if notification.count() == 0:
            return False

        close_btn = notification.first.get_by_role("button", name="Silence", exact=True)
        if close_btn.count() == 0:
            return False
        try:
            close_btn.first.dispatch_event("click", timeout=2000)
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_for(self, text: str) -> bool:
        """Wait for a notification containing specific text to appear.

        :param text: Text to search for in the notification.
        :returns: True if the notification was found, False if timeout.
        """
        notification = self._all().filter(has_text=text)
        try:
            notification.first.wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

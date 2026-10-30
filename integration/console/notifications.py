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

# A notification expires on its own after a few seconds, so a button that is still
# attached resolves at once.
_SILENCE_TIMEOUT = 1000

_READ_JS = """(elements) => elements.map((el) => {
  const text = (selector) => el.querySelector(selector)?.innerText.trim();
  const fields = {
    count: text(".lyra-text--tiny"),
    timestamp: text(".lyra-notification__time"),
    message: text(".lyra-notification__message"),
    description: text(".lyra-notification__description"),
  };
  const read = Object.fromEntries(
    Object.entries(fields).filter(([, value]) => value != null),
  );
  read.type = el.querySelector("svg[color*='error']") == null ? "info" : "error";
  return read;
})"""


class NotificationsClient:
    """Notifications management for Console UI automation."""

    def __init__(self, page: Page):
        self.page = page

    def _all(self) -> Locator:
        """All visible notifications. Each notification carries role=status."""
        return self.page.get_by_role("status")

    def check(
        self, *, timeout: sy.CrudeTimeSpan = 200 * sy.TimeSpan.MILLISECOND
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
            # One call reads every notification, so a toast that arrives mid-read cannot
            # shift the others.
            notifications: list[dict[str, Any]] = self._all().evaluate_all(_READ_JS)
            if len(notifications) > 0:
                return notifications
            sy.sleep(poll_interval)

        return []

    def close(self, *, notification_index: int = 0) -> bool:
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

    def close_all(self) -> None:
        """Silence every notification, including the ones folded behind the "+N more"
        control, and wait for them to clear.

        A notification that expires on its own before its click lands is already in
        the wanted state, so it does not fail the call.
        """
        buttons = self._all().get_by_role("button", name="Silence", exact=True)
        clear_all = self.page.locator(".console-notifications__controls").get_by_role(
            "button", name="Clear all", exact=True
        )
        if clear_all.count() > 0:
            try:
                clear_all.dispatch_event("click", timeout=_SILENCE_TIMEOUT)
            except PlaywrightTimeoutError:
                # An expired toast can drop the count to four, which hides the control.
                pass
        # Last to first: a silenced toast unmounts and shifts the indexes after it.
        for i in reversed(range(buttons.count())):
            try:
                buttons.nth(i).dispatch_event("click", timeout=_SILENCE_TIMEOUT)
            except PlaywrightTimeoutError:
                pass
        try:
            expect(buttons).to_have_count(0, timeout=2000)
        except AssertionError:
            pass

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

    def wait_for(
        self, text: str, *, name: str | None = None, description: str | None = None
    ) -> bool:
        """Wait for a notification containing specific text to appear.

        :param text: Text to search for in the notification.
        :param name: Text the notification's name must contain, if given.
        :param description: Text the notification's description must contain, if
            given.
        :returns: True if the notification was found, False if timeout.
        """
        notification = self._all().filter(has_text=text)
        if name is not None:
            notification = notification.filter(
                has=self.page.locator(".lyra-notification__name", has_text=name)
            )
        if description is not None:
            notification = notification.filter(
                has=self.page.locator(
                    ".lyra-notification__description", has_text=description
                )
            )
        try:
            notification.first.wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

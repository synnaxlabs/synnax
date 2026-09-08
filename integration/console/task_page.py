#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator, Page

import synnax as sy
from console.layout import LayoutClient
from console.page import ConsolePage

CHANNEL_LIST_SELECTOR = ".console-channel-list"
ADD_CHANNEL_BUTTON = "header:has-text('Channels') .pluto-icon--add"


class TaskPage(ConsolePage):
    """Base class for task pages with common task operations.

    Provides common functionality for all task types (NI, LabJack, OPC UA, Modbus):
    - deploy() - Configure, save, and start the task
    - stop() - Stop the task
    - status() - Get task status
    - set_parameters() - Set common task parameters

    Subclasses should implement device-specific functionality like channel management.
    """

    pluto_label: str = ".console-task-configure"
    STARTED_MESSAGE = "Task started successfully"
    STOPPED_MESSAGE = "Task stopped successfully"

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        *,
        pane_locator: Locator,
    ) -> None:
        """Initialize a TaskPage wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(layout, client, page_name, pane_locator=pane_locator)

    def _pane(self) -> Locator | Page:
        return self.pane_locator if self.pane_locator is not None else self.page

    def _controls(self) -> Locator:
        controls = self._pane().locator(".console-task-controls").first
        controls.wait_for(state="visible", timeout=5000)
        return controls

    def _channel_list(self) -> Locator:
        return self._pane().locator(CHANNEL_LIST_SELECTOR).first

    def select_device(self, label: str, name: str) -> None:
        """Pick ``name`` in the device dropdown of the form field ``label``."""
        self.layout.click_btn(label)
        self.layout.select_from_dropdown(name)

    def add_channel_row(self, index: int) -> None:
        """Append a channel row. The first one comes from the empty-state action, the
        rest from the list header."""
        if index == 0:
            self.layout.click("Add channel")
        else:
            self._pane().locator(ADD_CHANNEL_BUTTON).click()

    def deploy(self, expect: str | None = STARTED_MESSAGE) -> None:
        """Deploy the task from the controls bar.

        Play configures, saves, and starts the task.

        :param expect: Status message to wait for after the click. None skips the
            wait.
        """
        controls = self._controls()
        self.notifications.close_all()
        play = controls.locator("button:has(.pluto-icon--play)")
        play.wait_for(state="visible", timeout=5000)
        play.click()
        if expect is not None:
            self.wait_for_status(expect)

    def stop(self) -> None:
        """Stop the task from the controls bar and wait for the stopped message."""
        controls = self._controls()
        stop_btn = controls.locator("button:has(.pluto-icon--stop)")
        stop_btn.wait_for(state="visible", timeout=5000)
        self.notifications.close_all()
        stop_btn.click()
        self.wait_for_status(self.STOPPED_MESSAGE)

    def wait_for_status(self, substr: str, timeout: float = 30000) -> str:
        """Wait for the status bar to contain a substring.

        :param substr: The substring to wait for in the status message.
        :param timeout: Maximum wait in milliseconds.
        :returns: The full status message text.
        """
        msg = self._controls().locator(
            f".console-task-status__message-text:has-text('{substr}')"
        )
        msg.wait_for(state="visible", timeout=timeout)
        return msg.inner_text().strip()

    def wait_for_status_level(
        self,
        levels: tuple[str, ...],
        timeout: sy.TimeSpan = 30 * sy.TimeSpan.SECOND,
    ) -> dict[str, str]:
        """Poll the status box until its level is one of ``levels``.

        :param levels: Accepted status levels, e.g. ``("error", "warning")``.
        :param timeout: Maximum time to wait.
        :returns: The matching status, as returned by ``status()``.
        """
        timer = sy.Timer()
        while True:
            status = self.status()
            if status["level"] in levels:
                return status
            if timer.elapsed() > timeout:
                raise AssertionError(
                    f"Task status level stayed '{status['level']}' "
                    f"('{status['msg']}'), expected one of {levels}"
                )
            sy.sleep(0.25)

    def field_help_text(self, label: str) -> str:
        """Return the help text shown under a form field, such as a validation
        error after a blocked deploy.

        :param label: Label of the field.
        """
        item = self.page.locator(f"text={label}").locator(
            "xpath=ancestor::*[contains(@class, 'pluto-input__item')][1]"
        )
        help_text = item.locator(".pluto-input-help-text").first
        help_text.wait_for(state="visible", timeout=5000)
        return help_text.inner_text().strip()

    def status(self) -> dict[str, str]:
        """Get the current status information from the task status box.

        Returns:
            Dictionary containing:
                - msg: The status message (e.g., "Task has not been configured")
                - level: The alert level (e.g., "disabled", "info", "success", "error")
        """
        status_element = (
            self._controls().locator(".console-task-status__message-text").first
        )

        # Parse status level from CSS class
        class_attr = status_element.get_attribute("class") or ""
        level = "unknown"
        for cls in class_attr.split():
            if cls.startswith("pluto--status-"):
                level = cls.replace("pluto--status-", "")
                break

        msg = status_element.inner_text()

        return {
            "msg": msg,
            "level": level,
        }

    def copy_link(self) -> str:
        """Copy link to the task via the utility button in the form header."""
        link_button = self.page.locator(".pluto-icon--link").locator("..")
        link_button.click(timeout=5000)
        return self.layout.read_clipboard()

    def verify_config(self, expected_channels: list[str]) -> None:
        """Verify the task config page is visible and contains expected channels.

        Args:
            expected_channels: Channel identifiers expected in the channel list.
        """
        if self.pane_locator is None:
            raise RuntimeError("No pane locator available for config verification")
        self.pane_locator.wait_for(state="visible", timeout=10000)
        for channel in expected_channels:
            self.pane_locator.get_by_text(channel).first.wait_for(
                state="visible", timeout=5000
            )

    def set_parameters(
        self,
        *,
        task_name: str | None = None,
        data_saving: bool | None = None,
        auto_start: bool | None = None,
        sample_rate: float | None = None,
        stream_rate: float | None = None,
    ) -> None:
        """Set common task parameters.

        :param task_name: The name for the task.
        :param data_saving: Whether to save data to the server.
        :param auto_start: Whether to start the task automatically.
        :param sample_rate: Sample rate in Hz, for tasks with a "Sample rate" field.
        :param stream_rate: Stream rate in Hz, for tasks with a "Stream rate" field.
        """
        layout = self.layout

        if task_name is not None:
            layout.fill_input_field("Name", task_name)
            layout.press_enter()

        if data_saving is not None:
            if data_saving != layout.get_toggle("Data saving"):
                layout.click_checkbox("Data saving")

        if auto_start is not None:
            if auto_start != layout.get_toggle("Auto start"):
                layout.click_checkbox("Auto start")

        if sample_rate is not None:
            layout.fill_input_field("Sample rate", str(sample_rate))

        if stream_rate is not None:
            layout.fill_input_field("Stream rate", str(stream_rate))

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""MQTT invalid configuration tests through the Console.

The Console form blocks invalid settings before they reach the Driver and renders a
field error. A broker that is down and a channel another task writes are only found
at runtime, so the Driver reports those.
"""

from console.task.mqtt import BROKER_FIELD_LABEL, MQTTRead
from tests.console.task.mqtt_case import MQTTCase

TOPIC = "plant/sensors"
POINTER = "/temperature"
WILDCARD_TOPIC = "plant/+"


class MQTTInvalidConfig(MQTTCase):
    """Verify invalid MQTT read task configurations are rejected.

    Tests (run sequentially on one task page):
        1. No device selected: the form blocks the deploy.
        2. Wildcard topic: the form blocks the deploy at the topic field.
        3. No enabled entries: the form blocks the deploy.
        4. Broker down: the Driver warns until the broker returns, then runs.
        5. Duplicate channel: a second task on the same field is rejected while
           the first one runs.
    """

    def run(self) -> None:
        page = self.test_no_device()
        self.test_wildcard_topic(page)
        self.test_no_enabled_entries(page)
        self.test_broker_down(page)
        self.test_duplicate_channel(page)

    def test_no_device(self) -> MQTTRead:
        self.log("Testing: No device selected")
        page = self.create_page(MQTTRead, "MQTT Console Invalid")
        self.assert_device_required(page, BROKER_FIELD_LABEL)
        return page

    def test_wildcard_topic(self, page: MQTTRead) -> None:
        self.log("Testing: Wildcard topic")
        self.select_broker(page)
        page.add_entry()
        page.set_topic(WILDCARD_TOPIC)
        page.add_field(POINTER)
        page.deploy(expect=None)
        message = page.field_help_text("Topic")
        assert message == "Topic must not hold the wildcards + or #", (
            f"Topic field should report the wildcard, got '{message}'"
        )
        self.assert_not_deployed(page, "wildcard topic")

    def test_no_enabled_entries(self, page: MQTTRead) -> None:
        self.log("Testing: No enabled entries")
        page.set_topic(TOPIC)
        page.disable_entry(TOPIC)
        page.deploy(expect=None)
        self.assert_not_deployed(page, "no enabled entries")

    def test_broker_down(self, page: MQTTRead) -> None:
        """Deploy against a stopped broker; the task warns, then reads once the
        broker is back."""
        self.log("Testing: Broker down (runtime)")
        page.enable_entry(TOPIC)
        self.cleanup_simulator()
        self.assert_driver_rejects(page, "broker down")
        self.start_simulator()
        page.wait_for_status(page.STARTED_MESSAGE, timeout=60000)
        self.console.tasks.wait_for_state(page.page_name, "running")

    def test_duplicate_channel(self, page: MQTTRead) -> None:
        """Deploy a second task on the same field of the same topic while the first
        one runs; the Console reuses the channel and the Driver rejects the writer."""
        self.log("Testing: Duplicate channel (two tasks on same channel)")
        # Page objects locate their pane by task type, so the first tab closes before a
        # second read task opens.
        self.console.close_all_tabs()
        second = self.create_page(MQTTRead, "MQTT Console Duplicate")
        self.select_broker(second)
        second.add_entry()
        second.set_topic(TOPIC)
        second.add_field(POINTER)
        self.assert_driver_rejects(second, "duplicate channel")
        self.console.tasks.stop(page.page_name)

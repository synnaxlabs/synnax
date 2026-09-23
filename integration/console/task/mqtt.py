#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""MQTT task pages for Console UI automation."""

from playwright.sync_api import Locator, expect

from console.task_page import TaskPage

BROKER_FIELD_LABEL = "MQTT broker"
BROWSER_SELECTOR = ".console-mqtt-browser"
BROWSER_LOADING_SELECTOR = ".console-mqtt-browser__loading-icon"
ENTRY_LIST_SELECTOR = ".console-topic-list"
ENTRY_TITLE_SELECTOR = ".console-mqtt-topic-list-item__topic"
DETAILS_SELECTOR = ".console-topic-details"
ADD_BUTTON = "header button:has(.pluto-icon--add)"
# The list wraps a title in bidi isolates so a topic reads left to right.
LRI = "\N{LEFT-TO-RIGHT ISOLATE}"
PDI = "\N{POP DIRECTIONAL ISOLATE}"


class MQTTTask(TaskPage):
    """Shared MQTT read and write page. Entries come from the Browser by drag, or
    from the list header and the details pane."""

    noun: str

    def _browser(self) -> Locator:
        return self._pane().locator(BROWSER_SELECTOR).first

    def _entry_list(self) -> Locator:
        return self._pane().locator(ENTRY_LIST_SELECTOR).first

    def _rows(self) -> Locator:
        return self._entry_list().get_by_role("option")

    def _row(self, title: str) -> Locator:
        title_text = self.page.get_by_text(f"{LRI}{title}{PDI}", exact=True)
        return self._rows().filter(has=title_text)

    def _details(self) -> Locator:
        return self._pane().locator(DETAILS_SELECTOR).first

    def _wait_for_browse(self) -> None:
        loading = self._browser().locator(BROWSER_LOADING_SELECTOR)
        loading.wait_for(state="hidden", timeout=30000)

    def _fill(self, label: str, value: str) -> None:
        """Fill the details field ``label`` and blur it so the form takes the value."""
        field = self._details().get_by_text(label, exact=True).locator("..")
        field.locator("input").first.fill(value)
        self.page.keyboard.press("Tab")

    def _select(self, label: str, option: str) -> None:
        """Pick ``option`` in the details dropdown ``label``."""
        field = self._details().get_by_text(label, exact=True).locator("..")
        field.locator("button").first.click()
        self.layout.select_from_dropdown(option, exact=True)

    def select_broker(self, name: str) -> None:
        """Pick the broker the task talks to and wait for its Browser.

        :param name: Name of the broker device.
        """
        self.select_device(BROKER_FIELD_LABEL, name)
        self._browser().wait_for(state="visible", timeout=10000)

    def browse_topics(self, filter: str = "#") -> None:
        """Browse the topics of the broker that match ``filter`` for a few seconds.

        :param filter: MQTT topic filter, with the wildcards + and #.
        """
        browser = self._browser()
        browser.get_by_placeholder("#").fill(filter)
        browser.get_by_role("button", name="Browse").click()
        self._wait_for_browse()

    def add_topics(self, topics: list[str]) -> None:
        """Drag browsed topics onto the entry list.

        :param topics: Topics the Browser lists.
        """
        target = self._entry_list()
        for topic in topics:
            item = (
                self._browser()
                .get_by_role("option")
                .filter(has=self.page.get_by_text(topic, exact=True))
            )
            item.wait_for(state="visible", timeout=15000)
            item.drag_to(target)
            self._row(topic).first.wait_for(state="visible", timeout=5000)

    def browse_sparkplug(self, group: str = "") -> None:
        """Switch the Browser to Sparkplug B and browse the edge nodes of ``group``.

        :param group: Sparkplug B group. Empty browses every group.
        """
        browser = self._browser()
        browser.get_by_role("button", name="Sparkplug B").click()
        browser.get_by_placeholder("All groups").fill(group)
        browser.get_by_role("button", name="Browse").click()
        self._wait_for_browse()

    def _tree_item(self, name: str) -> Locator:
        return (
            self._browser()
            .get_by_role("treeitem")
            .filter(has=self.page.get_by_text(name, exact=True))
        )

    def expand_node(self, name: str) -> None:
        """Expand an edge node or a device in the Sparkplug B Browser. The tags of an
        edge node load through the scan task of the rack.

        :param name: ``group/edge node`` of an edge node, or the name of a device.
        """
        node = self._tree_item(name)
        node.wait_for(state="visible", timeout=15000)
        if node.get_attribute("aria-expanded") == "true":
            return
        node.click()
        expect(node).to_have_attribute("aria-expanded", "true", timeout=30000)

    def add_tags(self, tags: list[str]) -> None:
        """Drag browsed Sparkplug B tags onto the entry list.

        :param tags: Tag names the Browser lists under an expanded node.
        """
        target = self._entry_list()
        for tag in tags:
            item = self._tree_item(tag)
            item.wait_for(state="visible", timeout=15000)
            item.drag_to(target)
            self._row(tag).first.wait_for(state="visible", timeout=5000)

    def add_entry(self) -> None:
        """Append an empty entry and select it. The first one comes from the
        empty-state action, the rest from the list header."""
        index = self._rows().count()
        if index == 0:
            self.layout.click(f"Add {self.noun}")
        else:
            self._entry_list().locator(ADD_BUTTON).first.click()
        self._rows().nth(index).wait_for(state="visible", timeout=5000)

    def select_entry(self, title: str) -> None:
        """Select the listed entry ``title`` so its details show."""
        self._row(title).first.click()
        self._details().wait_for(state="visible", timeout=5000)

    def set_topic(self, topic: str) -> None:
        """Set the topic of the selected entry."""
        self._fill("Topic", topic)

    def entries(self) -> list[str]:
        """Return the title of each listed entry, in order."""
        titles = self._entry_list().locator(ENTRY_TITLE_SELECTOR).all_inner_texts()
        return [t.strip(LRI + PDI) for t in titles]

    def disable_entry(self, title: str) -> None:
        """Disable a listed entry via its context menu."""
        self.ctx_menu.action(self._row(title).first, "Disable")

    def enable_entry(self, title: str) -> None:
        """Enable a listed entry via its context menu."""
        self.ctx_menu.action(self._row(title).first, "Enable")


class MQTTRead(MQTTTask):
    """MQTT read task page. A plain entry holds the fields of a JSON payload."""

    page_type = "MQTT read task"
    pluto_label: str = ".console-task-configure--mqtt_read"
    noun = "entry"

    def _fields(self) -> Locator:
        return self._channel_list().get_by_role("option")

    def add_field(self, pointer: str, data_type: str | None = None) -> None:
        """Append a field to the selected entry.

        :param pointer: JSON pointer of the field in the payload.
        :param data_type: Synnax data type, e.g. "float64". Keeps the default when
            None.
        """
        index = self._fields().count()
        if index == 0:
            self.layout.click("Add field")
        else:
            self._channel_list().locator(ADD_BUTTON).first.click()
        row = self._fields().nth(index)
        row.wait_for(state="visible", timeout=5000)
        row.locator("input").first.fill(pointer)
        self.page.keyboard.press("Tab")
        if data_type is not None:
            row.get_by_role("button").first.click()
            self.layout.select_from_dropdown(data_type, exact=True)

    def fields(self) -> list[str]:
        """Return the pointer of each field of the selected entry, in order."""
        rows = self._fields()
        return [
            rows.nth(i).locator("input").first.input_value()
            for i in range(rows.count())
        ]


class MQTTWrite(MQTTTask):
    """MQTT write task page. A plain target publishes one command channel."""

    page_type = "MQTT write task"
    pluto_label: str = ".console-task-configure--mqtt_write"
    noun = "target"

    def configure_channel(self, pointer: str, json_type: str, data_type: str) -> None:
        """Set the command channel of the selected plain target.

        :param pointer: JSON pointer the value takes in the payload.
        :param json_type: JSON type as listed, e.g. "Number".
        :param data_type: Synnax data type of the channel, e.g. "float64".
        """
        self._fill("JSON pointer", pointer)
        self._select("JSON type", json_type)
        self._select("Synnax data type", data_type)


class MQTTEdge(TaskPage):
    """Sparkplug edge node page. Each tag is a row of channel, name, and
    Sparkplug B type."""

    page_type = "Sparkplug edge node"
    pluto_label: str = ".console-task-configure--mqtt_sparkplug_edge"

    def _rows(self) -> Locator:
        return self._channel_list().get_by_role("option")

    def select_broker(self, name: str) -> None:
        """Pick the broker the edge node publishes to."""
        self.select_device(BROKER_FIELD_LABEL, name)

    def set_node(self, group: str, edge_node: str) -> None:
        """Set the Sparkplug B identity of the edge node."""
        self.layout.fill_input_field("Group", group)
        self.layout.fill_input_field("Edge node", edge_node)

    def add_tag(self, channel_name: str, name: str | None = None) -> None:
        """Append a tag that publishes ``channel_name``. The tag takes the name of
        the channel unless ``name`` is given."""
        index = self._rows().count()
        self.add_channel_row(index)
        row = self._rows().nth(index)
        row.wait_for(state="visible", timeout=5000)
        row.get_by_role("button").first.click()
        self.layout.select_from_dropdown(channel_name)
        name_input = row.locator("input").first
        expect(name_input).to_have_value(channel_name, timeout=5000)
        if name is not None:
            name_input.fill(name)
            self.page.keyboard.press("Tab")

    def tags(self) -> list[str]:
        """Return the name of each tag, in order."""
        rows = self._rows()
        return [
            rows.nth(i).locator("input").first.input_value()
            for i in range(rows.count())
        ]

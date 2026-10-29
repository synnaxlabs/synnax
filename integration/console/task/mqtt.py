#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""MQTT task pages for Console UI automation."""

import re

from playwright.sync_api import Locator, expect

from console.task_page import TaskPage

BROKER_FIELD_LABEL = "MQTT broker"
DETAILS_SELECTOR = ".console-details__form"
ENTRY_TITLE_SELECTOR = ".console-mqtt-entry-label > .lyra-text--overflow-fade"
FIELD_POINTER_SELECTOR = ".console-field-pointer"
CHANNEL_NAME_SELECTOR = ".console-task__channel-name"
TREE_ROWS_JS = f"""els => els.map((e) => [
    Number(e.getAttribute("aria-level")),
    e.querySelector("{ENTRY_TITLE_SELECTOR}")?.textContent ?? "",
    e.querySelector("{FIELD_POINTER_SELECTOR}")?.textContent ?? "",
])"""
# The list wraps a title in bidi isolates so a topic reads left to right.
LRI = "\N{LEFT-TO-RIGHT ISOLATE}"
PDI = "\N{POP DIRECTIONAL ISOLATE}"
WHOLE_PAYLOAD = "Whole payload"
# The submit button of the add dialog counts the picks: "Add 4 channels".
SUBMIT_PICKS = re.compile(r"^Add \d+ ")
# A listen of the dialog takes a few seconds before it lists anything.
LISTEN_TIMEOUT = 30000
TAG_NAME_PLACEHOLDER = "oven/temperature"
NO_COMMAND_CHANNEL = "No command channel"


def _plain(title: str) -> str:
    """Return a listed title without its bidi isolates."""
    return title.strip().strip(LRI + PDI)


class MQTTTask(TaskPage):
    """Shared MQTT read and write page. Entries come from the add dialog, which lists
    what the broker publishes."""

    add_label: str

    def _rows(self) -> Locator:
        raise NotImplementedError

    def _title(self, title: str) -> Locator:
        return self._rows().get_by_text(f"{LRI}{title}{PDI}", exact=True).first

    def _details(self) -> Locator:
        return self._pane().locator(DETAILS_SELECTOR).first

    def _add_button(self) -> Locator:
        return self._pane().get_by_role("button", name=self.add_label, exact=True)

    def _dialog(self) -> Locator:
        tabs = self.page.get_by_role("button", name="JSON topics", exact=True)
        return self.page.get_by_role("dialog").filter(has=tabs)

    def _open_dialog(self) -> Locator:
        self._add_button().click()
        dialog = self._dialog()
        dialog.wait_for(state="visible", timeout=5000)
        return dialog

    def _pick(self, checkbox: Locator) -> None:
        checkbox.wait_for(state="visible", timeout=LISTEN_TIMEOUT)
        # The indicator paints over the input and takes a real click.
        checkbox.dispatch_event("click")
        expect(checkbox).to_be_checked(timeout=5000)

    def _submit(self, dialog: Locator) -> None:
        dialog.get_by_role("button", name=SUBMIT_PICKS).click()
        dialog.wait_for(state="hidden", timeout=5000)

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
        """Pick the broker the task talks to and wait for the form body.

        :param name: Name of the broker device.
        """
        self.select_device(BROKER_FIELD_LABEL, name=name)
        self._add_button().wait_for(state="visible", timeout=10000)

    def add_topics(self, topics: list[str]) -> None:
        """Pick topics in the add dialog and add them. A pick takes the numbers and
        booleans of a payload, or the whole of a bare value.

        :param topics: Topics the broker publishes.
        """
        dialog = self._open_dialog()
        search = dialog.get_by_placeholder("Search topics", exact=True)
        for topic in topics:
            search.fill(topic)
            leaf = topic.rsplit("/", 1)[-1]
            self._pick(dialog.get_by_role("checkbox", name=f"Pick {leaf}", exact=True))
        self._submit(dialog)
        for topic in topics:
            self._title(topic).wait_for(state="visible", timeout=5000)

    def add_tags(self, edge_node: str, tags: list[str]) -> None:
        """Pick Sparkplug B tags of an edge node in the add dialog and add them.

        :param edge_node: Name of the edge node, without its group.
        :param tags: Names of the tags, as the edge node or one of its devices publishes
            them.
        """
        dialog = self._open_dialog()
        dialog.get_by_role("button", name="Sparkplug B", exact=True).click()
        node = dialog.get_by_role("treeitem").get_by_text(edge_node, exact=True)
        node.wait_for(state="visible", timeout=LISTEN_TIMEOUT)
        node.click()
        for tag in tags:
            self._pick(dialog.get_by_role("checkbox", name=f"Record {tag}", exact=True))
        self._submit(dialog)

    def add_entry(self) -> None:
        """Add an entry with no topic through the add dialog, and select it."""
        dialog = self._open_dialog()
        dialog.get_by_role("button", name="Add a topic by name", exact=True).click()
        dialog.wait_for(state="hidden", timeout=5000)
        topic = self._details().get_by_text("Topic", exact=True)
        topic.wait_for(state="visible", timeout=5000)

    def select_entry(self, title: str) -> None:
        """Select the listed entry ``title`` so its details show."""
        self._title(title).click()
        self._details().wait_for(state="visible", timeout=5000)

    def set_topic(self, topic: str) -> None:
        """Set the topic of the selected entry."""
        self._fill("Topic", topic)

    def entries(self) -> list[str]:
        """Return the title of each listed entry, in order. A Sparkplug B entry is
        titled by its full tag path, such as ``Plant/Line1/Pump1/setpoint``."""
        titles = self._rows().locator(ENTRY_TITLE_SELECTOR).all_inner_texts()
        return [_plain(t) for t in titles]

    def disable_entry(self, title: str) -> None:
        """Disable a listed entry via its context menu."""
        self.ctx_menu.action(self._title(title), "Disable")

    def enable_entry(self, title: str) -> None:
        """Enable a listed entry via its context menu."""
        self.ctx_menu.action(self._title(title), "Enable")


class MQTTRead(MQTTTask):
    """MQTT read task page. The list is a tree: a plain entry holds the fields of a
    JSON payload beneath it."""

    page_type = "MQTT read task"
    pluto_label: str = ".console-task-configure--mqtt_read"
    add_label = "Add data"

    def _rows(self) -> Locator:
        return self._pane().get_by_role("tree").first.get_by_role("treeitem")

    def add_field(
        self, title: str, pointer: str, *, data_type: str | None = None
    ) -> None:
        """Add a field to a listed entry via its context menu, and select it.

        :param title: Title of the entry.
        :param pointer: JSON pointer of the field in the payload.
        :param data_type: Synnax data type, e.g. "float64". Keeps the default when
            None.
        """
        self.ctx_menu.action(self._title(title), "Add field")
        self._details().get_by_text("Pointer", exact=True).wait_for(
            state="visible", timeout=5000
        )
        self._fill("Pointer", pointer)
        if data_type is not None:
            self._select("Data type", data_type)

    def fields(self, title: str) -> list[str]:
        """Return the pointer of each field of a listed entry, in order. An empty
        pointer takes the whole payload.

        :param title: Title of the entry.
        :raises AssertionError: If no entry has the title.
        """
        rows = self._rows().evaluate_all(TREE_ROWS_JS)
        pointers: list[str] = []
        found = False
        for level, entry, pointer in rows:
            if int(level) == 1:
                if found:
                    break
                found = _plain(str(entry)) == title
            elif found:
                pointers.append("" if pointer == WHOLE_PAYLOAD else str(pointer))
        if not found:
            raise AssertionError(f"No entry is titled {title!r}")
        return pointers


class MQTTWrite(MQTTTask):
    """MQTT write task page. A plain target publishes one command channel."""

    page_type = "MQTT write task"
    pluto_label: str = ".console-task-configure--mqtt_write"
    add_label = "Add targets"

    def _rows(self) -> Locator:
        return self._channel_list().get_by_role("option")

    def configure_channel(
        self, *, pointer: str, json_type: str, data_type: str
    ) -> None:
        """Set the command channel of the selected plain target.

        :param pointer: JSON pointer the value takes in the payload.
        :param json_type: JSON type as listed, e.g. "Number".
        :param data_type: Synnax data type of the channel, e.g. "float64".
        """
        self._fill("Pointer", pointer)
        self._select("JSON type", json_type)
        self._select("Data type", data_type)


class MQTTEdge(TaskPage):
    """Sparkplug edge node page. Each tag is a row of channel, name, Sparkplug B type,
    and command channel."""

    page_type = "Sparkplug edge node"
    pluto_label: str = ".console-task-configure--mqtt_sparkplug_edge"

    def _rows(self) -> Locator:
        return self._channel_list().get_by_role("option")

    def _add_button(self) -> Locator:
        return self._pane().get_by_role("button", name="Add tag", exact=True)

    @staticmethod
    def _name_input(row: Locator) -> Locator:
        return row.get_by_placeholder(TAG_NAME_PLACEHOLDER, exact=True)

    def _fill(self, placeholder: str, value: str) -> None:
        self._pane().get_by_placeholder(placeholder, exact=True).fill(value)
        self.page.keyboard.press("Tab")

    def _row(self, name: str) -> Locator:
        rows = self._rows()
        for i in range(rows.count()):
            row = rows.nth(i)
            if self._name_input(row).input_value() == name:
                return row
        raise AssertionError(f"No tag is named {name!r}")

    def select_broker(self, name: str) -> None:
        """Pick the broker the edge node publishes to and wait for the form body.

        :param name: Name of the broker device.
        """
        self.select_device(BROKER_FIELD_LABEL, name=name)
        self._add_button().wait_for(state="visible", timeout=10000)

    def set_node(self, group: str, edge_node: str) -> None:
        """Set the Sparkplug B identity of the edge node."""
        self._fill("plant", group)
        self._fill("line1", edge_node)

    def add_tag(self, channel_name: str, *, name: str | None = None) -> None:
        """Add a tag that publishes ``channel_name``. The tag takes the name of the
        channel unless ``name`` is given."""
        index = self._rows().count()
        self._add_button().click()
        row = self._rows().nth(index)
        row.wait_for(state="visible", timeout=5000)
        row.get_by_role("button").first.click()
        self.layout.select_from_dropdown(channel_name)
        name_input = self._name_input(row)
        expect(name_input).to_have_value(channel_name, timeout=5000)
        if name is not None:
            name_input.fill(name)
            self.page.keyboard.press("Tab")

    def add_command_channel(self, name: str) -> None:
        """Name a command channel for a tag via its context menu. Configure creates the
        channel.

        :param name: Name of the tag.
        """
        row = self._row(name)
        self.ctx_menu.action(self._name_input(row), "Add command channel")
        expect(row.get_by_text(NO_COMMAND_CHANNEL, exact=True)).to_have_count(
            0, timeout=5000
        )

    def command_channel(self, name: str) -> str:
        """Return the name of the command channel of a tag, or "" for none.

        :param name: Name of the tag.
        """
        names = self._row(name).locator(CHANNEL_NAME_SELECTOR)
        if names.count() == 0:
            return ""
        return names.first.inner_text().strip()

    def tags(self) -> list[str]:
        """Return the name of each tag, in order."""
        rows = self._rows()
        return [
            self._name_input(rows.nth(i)).input_value() for i in range(rows.count())
        ]

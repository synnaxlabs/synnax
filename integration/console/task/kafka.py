#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Kafka task pages for Console UI automation."""

import re

from playwright.sync_api import Locator, expect

from console.page import PageType
from console.task_page import TaskPage

_CLUSTER_FIELD_LABEL = "Kafka cluster"
_DETAILS_SELECTOR = ".console-details__form"
_LIST_SELECTOR = ".console-panes__list"
_TOPIC_SELECTOR = '[role="treeitem"][aria-level="1"]'
# The add dialog is the element with this class.
_DISCOVER_SELECTOR = ".console-kafka-discover"
# The name of a topic or record in the tree, after the tag of its kind.
_LABEL_SELECTOR = ".console-group-item__label .lyra-text--overflow-fade"
_EXPAND_SELECTOR = ".lyra-tree__expansion-indicator"
# The tree labels the box that reads a field "Read <field name>".
_READ_BOX_PREFIX = "Read "
# The submit button of the add dialog counts the picks: "Add 4 fields".
_SUBMIT_PICKS = re.compile(r"^Add \d+ ")
# The dialog lists topics, then reads a sample of a topic before it can pick it.
_LISTEN_TIMEOUT = 30000


class KafkaTask(TaskPage):
    """Shared Kafka read and write page. The tree lists each topic, with its records or
    targets beneath it, named by record key."""

    add_label: str

    def _list(self) -> Locator:
        return self._pane().locator(_LIST_SELECTOR).first

    def _items(self) -> Locator:
        return self._list().get_by_role("treeitem")

    def _label(self, item: Locator) -> str:
        return item.locator(_LABEL_SELECTOR).first.inner_text().strip()

    def _topics(self) -> Locator:
        return self._list().locator(_TOPIC_SELECTOR)

    def _topic(self, topic: str) -> Locator:
        label = self.page.locator(_LABEL_SELECTOR).get_by_text(topic, exact=True)
        return self._topics().filter(has=label)

    def _expand(self, item: Locator) -> None:
        if item.get_attribute("aria-expanded") == "false":
            item.locator(_EXPAND_SELECTOR).first.click()
        expect(item).to_have_attribute("aria-expanded", "true", timeout=5000)

    def _under(self, topic: str) -> list[Locator]:
        """Return the items beneath the listed topic, in order, with the topic and each
        of its records or targets expanded."""
        self._expand(self._topic(topic))
        items: list[Locator] = []
        inside = False
        i = 0
        # Expanding an item adds items after it, so the count is read on each pass.
        while i < self._items().count():
            item = self._items().nth(i)
            i += 1
            if item.get_attribute("aria-level") == "1":
                if inside:
                    break
                inside = self._label(item) == topic
                continue
            if not inside:
                continue
            if item.get_attribute("aria-expanded") == "false":
                self._expand(item)
            items.append(item)
        return items

    def _records(self, topic: str) -> list[Locator]:
        """Return the records or targets beneath a listed topic, in order."""
        return [i for i in self._under(topic) if i.get_attribute("aria-level") == "2"]

    def _add_button(self) -> Locator:
        return self._pane().get_by_role("button", name=self.add_label, exact=True)

    def select_cluster(self, name: str) -> None:
        """Pick the cluster the task talks to and wait for the form body.

        :param name: Name of the cluster device.
        """
        self.select_device(_CLUSTER_FIELD_LABEL, name=name)
        self._add_button().wait_for(state="visible", timeout=10000)

    def add_topics(self, topics: list[str]) -> None:
        """Pick topics that the cluster lists in the add dialog and add them.

        :param topics: Names of the topics.
        """
        self._add_button().click()
        dialog = self.page.locator(_DISCOVER_SELECTOR)
        dialog.wait_for(state="visible", timeout=5000)
        search = dialog.get_by_placeholder("Search topics", exact=True)
        for topic in topics:
            search.fill(topic)
            box = dialog.get_by_role("checkbox", name=f"Pick {topic}", exact=True)
            box.wait_for(state="visible", timeout=_LISTEN_TIMEOUT)
            # The indicator paints over the input and takes a real click.
            box.dispatch_event("click")
            expect(box).to_be_checked(timeout=_LISTEN_TIMEOUT)
        dialog.get_by_role("button", name=_SUBMIT_PICKS).click()
        dialog.wait_for(state="hidden", timeout=5000)
        for topic in topics:
            self._topic(topic).wait_for(state="visible", timeout=5000)

    def topics(self) -> list[str]:
        """Return the name of each listed topic, in order."""
        items = self._topics()
        return [self._label(items.nth(i)) for i in range(items.count())]

    def record_keys(self, topic: str) -> list[str]:
        """Return the label of each record or target beneath a listed topic, in order.
        A record with no key shows a stand-in, such as "Any key".

        :param topic: Name of the topic.
        """
        return [self._label(item) for item in self._records(topic)]


class KafkaRead(KafkaTask):
    """Kafka read task page. Each record of a topic lists the fields of its messages,
    with a box that sets whether the task reads each one."""

    page_type: PageType = "kafka_read"
    pluto_label: str = ".console-task-configure--kafka_read"
    add_label = "Add data"

    def fields(self, topic: str) -> list[str]:
        """Return the name of each field that the task reads from the records of a
        listed topic, in order.

        :param topic: Name of the topic.
        """
        read = re.compile(f"^{_READ_BOX_PREFIX}")
        names: list[str] = []
        for item in self._under(topic):
            box = item.get_by_role("checkbox", name=read)
            if box.count() > 0 and box.is_checked():
                label = box.get_attribute("aria-label") or ""
                names.append(label.removeprefix(_READ_BOX_PREFIX))
        return names


class KafkaWrite(KafkaTask):
    """Kafka write task page. Each target of a topic sends the message of its ICD
    under its record key."""

    page_type: PageType = "kafka_write"
    pluto_label: str = ".console-task-configure--kafka_write"
    add_label = "Add targets"

    def set_record_key(self, topic: str, record_key: str) -> None:
        """Set the record key of the only target of a listed topic.

        :param topic: Name of the topic.
        :param record_key: The key the target sends its records under.
        :raises AssertionError: If the topic has no target or more than one.
        """
        targets = self._records(topic)
        if len(targets) != 1:
            raise AssertionError(f"{topic} has {len(targets)} targets, expected one")
        targets[0].locator(_LABEL_SELECTOR).first.click()
        details = self._pane().locator(_DETAILS_SELECTOR).first
        field = details.get_by_text("Record key", exact=True).locator("..")
        field.locator("input").first.fill(record_key)
        self.page.keyboard.press("Tab")
        expect(targets[0].locator(_LABEL_SELECTOR).first).to_have_text(
            record_key, timeout=5000
        )

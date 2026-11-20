#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

import numpy as np

import synnax as sy
from console.case import ConsoleCase
from console.log import Log
from framework.utils import assert_envelope, named_envelope_copy
from x import random_name

ENTRY_COUNT = 1000
NEWEST = f"entry-{ENTRY_COUNT - 1:05d}"
ENTRY_PATTERN = re.compile(r"entry-(\d{5})")
# Far wider than any log pane, so the value wraps over several rows.
LONG_VALUE = "wrap-" + "0123456789" * 60


def _entry_indices(lines: list[str]) -> list[int]:
    return [int(m.group(1)) for ln in lines if (m := ENTRY_PATTERN.search(ln))]


def _has_newest(lines: list[str]) -> bool:
    return any(NEWEST in ln for ln in lines)


def _shows_older(lines: list[str]) -> bool:
    indices = _entry_indices(lines)
    return bool(indices) and max(indices) < ENTRY_COUNT - 1


def _has_long_value(lines: list[str]) -> bool:
    return any(LONG_VALUE in ln for ln in lines)


class LogHistory(ConsoleCase):
    """Test that a log shows stored entries: on reopen, while scrolled back, when an
    entry wraps, and after an export and import.
    """

    suffix: str
    idx_name: str
    data_name: str

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()

    def setup_channels(self) -> None:
        """Create a string channel and write ENTRY_COUNT stored entries to it."""
        self.idx_name = f"log_history_idx_{self.suffix}"
        self.data_name = f"log_history_data_{self.suffix}"
        index_ch = self.client.channels.create(
            name=self.idx_name, is_index=True, retrieve_if_name_exists=True
        )
        self.client.channels.create(
            name=self.data_name,
            data_type=sy.DataType.STRING,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )
        start = sy.TimeStamp.now() - sy.TimeSpan.SECOND * 5
        timestamps = int(start) + np.arange(ENTRY_COUNT, dtype=np.int64) * int(
            sy.TimeSpan.MILLISECOND
        )
        values = [f"entry-{i:05d}" for i in range(ENTRY_COUNT)]
        self.client.write(start, {self.idx_name: timestamps, self.data_name: values})

    def run(self) -> None:
        """Run all log history tests."""
        self.setup_channels()

        log = self.console.pages.create(Log, f"Log History {self.suffix}")
        self._cleanup_pages.append(log.page_name)
        log.set_channel(self.data_name)

        log = self.test_history_on_reopen(log)
        self.test_scroll_back(log)
        self.test_wrap(log)
        log.close()
        self.test_export_import(log.page_name)

    def test_history_on_reopen(self, log: Log) -> Log:
        """Test that a log shows stored entries when it opens and when it reopens."""
        self.log("Testing log history on open and reopen")

        entries = log.wait_for_copied_entries(until=_has_newest)
        assert _has_newest(entries), f"Log should show stored entries, got {entries!r}"

        log.close()
        reopened = self.console.pages.open(Log, log.page_name)
        entries = reopened.wait_for_copied_entries(until=_has_newest)
        assert _has_newest(entries), (
            f"Reopened log should show stored entries, got {entries!r}"
        )
        assert not reopened.is_waiting_for_data(), (
            "Reopened log should not wait for data"
        )
        return reopened

    def test_scroll_back(self, log: Log) -> None:
        """Test that scrolling up pauses the log and shows older entries."""
        self.log("Testing log scroll-back")

        log.scroll_back(2000)
        assert log.wait_until_scrolling_paused(), "Scrolling back should pause the log"

        entries = log.wait_for_copied_entries(until=_shows_older)
        assert _shows_older(entries), (
            f"Scrolled-back log should show older entries, got {entries!r}"
        )

        log.resume_scrolling()
        entries = log.wait_for_copied_entries(until=_has_newest)
        assert _has_newest(entries), (
            f"Resumed log should show the newest entry, got {entries!r}"
        )

    def test_wrap(self, log: Log) -> None:
        """Test that a value wider than the log copies as one whole entry."""
        self.log("Testing log entry wrap")

        now = sy.TimeStamp.now()
        with self.client.open_writer(
            start=now, channels=[self.idx_name, self.data_name]
        ) as writer:
            writer.write({self.idx_name: [now], self.data_name: [LONG_VALUE]})

        entries = log.wait_for_copied_entries(until=_has_long_value)
        assert _has_long_value(entries), (
            f"Wrapped entry should copy as one whole line, got {entries!r}"
        )

    def test_export_import(self, log_name: str) -> None:
        """Test that an exported log imports with its channel and stored entries."""
        self.log("Testing log export and import")

        exported = self.console.pages.export(log_name)
        assert_envelope(exported, envelope_type="log", min_version=2, name=log_name)

        import_name = f"Imported Log {self.suffix}"
        export_path = str(self.results_dir / f"{log_name}_export.json")
        with named_envelope_copy(export_path, name=import_name) as tmp_path:
            self.console.pages.import_file(tmp_path, name=import_name)
        self._cleanup_pages.append(import_name)

        imported = Log.from_open_page(
            layout=self.console.layout, client=self.client, name=import_name
        )
        assert imported.has_channel(self.data_name), (
            f"Imported log should keep channel {self.data_name}"
        )
        entries = imported.wait_for_copied_entries(until=_has_long_value)
        assert _has_long_value(entries), (
            f"Imported log should show stored entries, got {entries!r}"
        )
        imported.close()

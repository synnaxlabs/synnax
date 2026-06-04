#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import NamedTuple

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc import ArcCase

TRIG_SET = "trig_status_set"
TRIG_DUP_FUNC = "trig_dup_func"
TRIG_DUP_FLOW = "trig_dup_flow"
DUP_FUNC_KEY = "dup_func_key"

DUP_NAME = "duplicate_status_name"
DUP_KEYS = ("44440001", "44440002")
DUP_VARIANT = "info"
DUP_MESSAGE = "some message"

ARC_STL_STATUS_SOURCE = """
import status

trig_status_set -> status.set{
    key_or_name="status_flow_byname",
    message="set by name (flow)",
    variant="info"
} -> flow_byname_key
trig_status_set -> status.set{
    key_or_name="11110002",
    message="set by key (flow)",
    variant="warning"
} -> flow_bykey_key

func set_func_byname() {
    func_byname_key = status.set("status_func_byname", "set by name (func)", "success")
}
trig_status_set -> set_func_byname{}
func set_func_bykey() {
    func_bykey_key = status.set("22220002", "set by key (func)", "error")
}
trig_status_set -> set_func_bykey{}

func set_duplicate() {
    dup_func_key = status.set("duplicate_status_name", "some message", "info")
}
trig_dup_func -> set_duplicate{}
trig_dup_flow -> status.set{"duplicate_status_name", "some message", "info"}
"""


class Case(NamedTuple):
    """One form x resolution-mode combination over a predefined status row.

    variant and message are the values status.set writes; asserting the row
    carries them after set proves the predefined row was upserted in place.
    """

    name: str
    key: str
    variant: str
    message: str
    key_channel: str


CASES: list[Case] = [
    Case(
        name="status_flow_byname",
        key="11110001",
        variant="info",
        message="set by name (flow)",
        key_channel="flow_byname_key",
    ),
    Case(
        name="status_flow_bykey",
        key="11110002",
        variant="warning",
        message="set by key (flow)",
        key_channel="flow_bykey_key",
    ),
    Case(
        name="status_func_byname",
        key="22220001",
        variant="success",
        message="set by name (func)",
        key_channel="func_byname_key",
    ),
    Case(
        name="status_func_bykey",
        key="22220002",
        variant="error",
        message="set by key (func)",
        key_channel="func_bykey_key",
    ),
]

KEY_CHANNELS = [c.key_channel for c in CASES]


class StlStatus(ArcCase):
    """Test status.set over predefined rows, by name and by key.

    A single set trigger upserts every row at once. set must return the
    predefined key and upsert in place. Covers flow and func form.
    """

    arc_source = ARC_STL_STATUS_SOURCE
    arc_name_prefix = "ArcStlStatus"
    start_cmd_channel = "start_stl_status_cmd"
    subscribe_channels = KEY_CHANNELS + [DUP_FUNC_KEY]
    collect_notifications = True

    def setup(self) -> None:
        create_virtual_channel(self.client, TRIG_SET, sy.DataType.UINT8)
        create_virtual_channel(self.client, TRIG_DUP_FUNC, sy.DataType.UINT8)
        create_virtual_channel(self.client, TRIG_DUP_FLOW, sy.DataType.UINT8)
        create_virtual_channel(self.client, DUP_FUNC_KEY, sy.DataType.STRING)
        for ch in KEY_CHANNELS:
            create_virtual_channel(self.client, ch, sy.DataType.STRING)
        for c in CASES:
            self.client.statuses.set(
                sy.Status(
                    key=c.key,
                    name=c.name,
                    variant="disabled",
                    message="initialized",
                )
            )
        for key in ("44440001", "44440002"):
            self.client.statuses.set(
                sy.Status(
                    key=key,
                    name="duplicate_status_name",
                    variant="disabled",
                    message="initialized",
                )
            )
        super().setup()

    def teardown(self) -> None:
        for c in CASES:
            keys = [s.key for s in self._rows(c.name)]
            if keys:
                self.client.statuses.delete(keys)
        super().teardown()

    def _rows(self, name: str) -> list[sy.Status]:
        return [
            s for s in self.client.statuses.retrieve(search_term=name) if s.name == name
        ]

    def verify_sequence_execution(self) -> None:
        self._verify_set()
        self._verify_warn_on_duplicate()

    def _verify_set(self) -> None:
        self.log("Firing set trigger: every case upserts its predefined row")
        self.writer.write(TRIG_SET, 1)
        for c in CASES:
            self.wait_for_eq(c.key_channel, c.key, is_virtual=True)
            rows = self._rows(c.name)
            if len(rows) != 1:
                self.fail(f"{c.name}: expected 1 row after set, got {len(rows)}")
            row = rows[0]
            if (row.variant, row.message) != (c.variant, c.message):
                self.fail(
                    f"{c.name}: expected upsert to ({c.variant!r}, {c.message!r}), "
                    f"got ({row.variant!r}, {row.message!r})"
                )

    def _verify_warn_on_duplicate(self) -> None:
        self.log("Firing set duplicate: warn and set only the first status")
        self.writer.write(TRIG_DUP_FUNC, 1)
        self.wait_for_eq(DUP_FUNC_KEY, DUP_KEYS[0], is_virtual=True)
        if not self.wait_for_notification(f'multiple statuses named "{DUP_NAME}"'):
            self.fail("duplicate set did not surface a multi-match warning")
        rows = {s.key: s for s in self._rows(DUP_NAME)}
        first = rows[DUP_KEYS[0]]
        if (first.variant, first.message) != (DUP_VARIANT, DUP_MESSAGE):
            self.fail(
                f"first match {DUP_KEYS[0]}: expected ({DUP_VARIANT!r}, {DUP_MESSAGE!r}), "
                f"got ({first.variant!r}, {first.message!r})"
            )
        second = rows[DUP_KEYS[1]]
        if (second.variant, second.message) != ("disabled", "initialized"):
            self.fail(
                f"second match {DUP_KEYS[1]} should be unchanged, "
                f"got ({second.variant!r}, {second.message!r})"
            )

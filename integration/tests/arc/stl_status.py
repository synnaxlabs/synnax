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
from tests.arc.arc_case import ArcConsoleCase

TRIG_SET = "trig_status_set"
TRIG_DEL = "trig_status_del"

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
trig_status_del -> status.delete{
    key_or_name="status_flow_byname"
} -> flow_byname_del_ok
trig_status_del -> status.delete{key_or_name="11110002"} -> flow_bykey_del_ok

func set_func_byname() {
    func_byname_key = status.set("status_func_byname", "set by name (func)", "success")
}
trig_status_set -> set_func_byname{}
func set_func_bykey() {
    func_bykey_key = status.set("22220002", "set by key (func)", "error")
}
trig_status_set -> set_func_bykey{}
func del_func_byname() {
    func_byname_del_ok = status.delete("status_func_byname")
}
trig_status_del -> del_func_byname{}
func del_func_bykey() {
    func_bykey_del_ok = status.delete("22220002")
}
trig_status_del -> del_func_bykey{}
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
    del_ok_channel: str


CASES: list[Case] = [
    Case(
        name="status_flow_byname",
        key="11110001",
        variant="info",
        message="set by name (flow)",
        key_channel="flow_byname_key",
        del_ok_channel="flow_byname_del_ok",
    ),
    Case(
        name="status_flow_bykey",
        key="11110002",
        variant="warning",
        message="set by key (flow)",
        key_channel="flow_bykey_key",
        del_ok_channel="flow_bykey_del_ok",
    ),
    Case(
        name="status_func_byname",
        key="22220001",
        variant="success",
        message="set by name (func)",
        key_channel="func_byname_key",
        del_ok_channel="func_byname_del_ok",
    ),
    Case(
        name="status_func_bykey",
        key="22220002",
        variant="error",
        message="set by key (func)",
        key_channel="func_bykey_key",
        del_ok_channel="func_bykey_del_ok",
    ),
]

KEY_CHANNELS = [c.key_channel for c in CASES]
DEL_OK_CHANNELS = [c.del_ok_channel for c in CASES]


class StlStatus(ArcConsoleCase):
    """Test status.set / status.delete over predefined rows, by name and by key.

    A single set trigger upserts every row at once, a single delete trigger
    removes them. set must return the predefined key and upsert in place;
    delete must report success and remove the row. Covers flow and func form.
    """

    arc_source = ARC_STL_STATUS_SOURCE
    arc_name_prefix = "ArcStlStatus"
    start_cmd_channel = "start_stl_status_cmd"
    subscribe_channels = KEY_CHANNELS + DEL_OK_CHANNELS

    def setup(self) -> None:
        create_virtual_channel(self.client, TRIG_SET, sy.DataType.UINT8)
        create_virtual_channel(self.client, TRIG_DEL, sy.DataType.UINT8)
        for ch in KEY_CHANNELS:
            create_virtual_channel(self.client, ch, sy.DataType.STRING)
        for ch in DEL_OK_CHANNELS:
            create_virtual_channel(self.client, ch, sy.DataType.UINT8)
        for c in CASES:
            self.client.statuses.set(
                sy.Status(
                    key=c.key,
                    name=c.name,
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

        self.log("Firing delete trigger: every case removes its row")
        self.writer.write(TRIG_DEL, 1)
        for c in CASES:
            self.wait_for_eq(c.del_ok_channel, 1, is_virtual=True)
            if self._rows(c.name):
                self.fail(f"{c.name}: row still present after delete")

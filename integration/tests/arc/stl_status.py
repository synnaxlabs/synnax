#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Callable, NamedTuple, TypeVar

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_STL_STATUS_SOURCE = """
// Flow lifecycle. status.set emits the resolved key on its output, which we
// wire into flow_set_key so the Python side can verify that create and
// upsert resolve to the same UUID.
trigger_flow_set_create -> status.set{
    key_or_name="integ_status_flow",
    message="created from flow",
    variant="info"
} -> flow_set_key
trigger_flow_set_upsert -> status.set{
    key_or_name="integ_status_flow",
    message="updated from flow",
    variant="warning"
} -> flow_set_key
trigger_flow_del -> status.delete{key_or_name="integ_status_flow"}

// Func lifecycle. status.set is a regular expression in func form; we assign
// its return value (the resolved key) to func_set_key.
func set_create_via_func() {
    func_set_key = status.set("integ_status_func", "created from func", "success")
}
trigger_func_set_create -> set_create_via_func{}

func set_upsert_via_func() {
    func_set_key = status.set("integ_status_func", "updated from func", "error")
}
trigger_func_set_upsert -> set_upsert_via_func{}

func del_via_func() {
    status.delete("integ_status_func")
}
trigger_func_del -> del_via_func{}

// Chain pattern: chain_key is both a sink for status.set's output and the
// source that drives status.delete. Each trigger creates the row, writes the
// key, then immediately deletes it, so the next trigger creates a fresh row
// with a new UUID and the channel value changes across cycles.
trigger_chain -> status.set{
    key_or_name="integ_status_chain",
    message="cycle",
    variant="info"
} -> chain_key -> status.delete{key_or_name="integ_status_chain"}
"""


class Op(NamedTuple):
    variant: str
    message: str


class Lifecycle(NamedTuple):
    """Drives one create -> upsert -> delete walk on a single status row."""

    label: str
    name: str
    trigger_create: str
    trigger_upsert: str
    trigger_del: str
    key_channel: str
    create: Op
    upsert: Op


LIFECYCLES: list[Lifecycle] = [
    Lifecycle(
        label="flow",
        name="integ_status_flow",
        trigger_create="trigger_flow_set_create",
        trigger_upsert="trigger_flow_set_upsert",
        trigger_del="trigger_flow_del",
        key_channel="flow_set_key",
        create=Op("info", "created from flow"),
        upsert=Op("warning", "updated from flow"),
    ),
    Lifecycle(
        label="func",
        name="integ_status_func",
        trigger_create="trigger_func_set_create",
        trigger_upsert="trigger_func_set_upsert",
        trigger_del="trigger_func_del",
        key_channel="func_set_key",
        create=Op("success", "created from func"),
        upsert=Op("error", "updated from func"),
    ),
]

CHAIN_NAME = "integ_status_chain"
CHAIN_TRIGGER = "trigger_chain"
CHAIN_KEY = "chain_key"

ALL_NAMES = [lc.name for lc in LIFECYCLES] + [CHAIN_NAME]
TRIGGER_CHANNELS = [
    t
    for lc in LIFECYCLES
    for t in (lc.trigger_create, lc.trigger_upsert, lc.trigger_del)
] + [CHAIN_TRIGGER]
KEY_CHANNELS = [lc.key_channel for lc in LIFECYCLES] + [CHAIN_KEY]

T = TypeVar("T")


class StlStatus(ArcConsoleCase):
    """Test status.set / status.delete in flow, func, and chain contexts.

    For each Lifecycle we walk create -> upsert -> delete on a single row and
    verify the resolved key (captured via the matching key channel) is stable
    across create and upsert. The chain test wires status.set -> channel ->
    status.delete and verifies the captured key changes across cycles since
    the row is recreated each time.
    """

    arc_source = ARC_STL_STATUS_SOURCE
    arc_name_prefix = "ArcStlStatus"
    start_cmd_channel = "start_stl_status_cmd"
    subscribe_channels = KEY_CHANNELS

    def setup(self) -> None:
        for ch in TRIGGER_CHANNELS:
            create_virtual_channel(self.client, ch, sy.DataType.UINT8)
        for ch in KEY_CHANNELS:
            create_virtual_channel(self.client, ch, sy.DataType.STRING)
        super().setup()

    def teardown(self) -> None:
        for name in ALL_NAMES:
            keys = [s.key for s in self._rows(name)]
            if keys:
                self.client.statuses.delete(keys)
        super().teardown()

    def _rows(self, name: str) -> list[sy.Status]:
        return [
            s for s in self.client.statuses.retrieve(search_term=name) if s.name == name
        ]

    def _poll(
        self,
        check: Callable[[], T | None],
        what: str,
        timeout_s: float = 5.0,
    ) -> T:
        """Poll check() at 100ms intervals until it returns a truthy value."""
        deadline = sy.TimeStamp.now() + int(timeout_s * sy.TimeSpan.SECOND)
        while sy.TimeStamp.now() < deadline:
            result = check()
            if result:
                return result
            sy.sleep(0.1)
        self.fail(f"{what} did not happen within {timeout_s}s")
        raise AssertionError("unreachable")

    def _run_lifecycle(self, lc: Lifecycle) -> None:
        self.log(f"=== {lc.label}: create -> upsert -> delete ===")

        self.writer.write(lc.trigger_create, 1)
        created = self._poll(
            lambda: next(iter(self._rows(lc.name)), None),
            f"{lc.name} create",
        )
        if (created.variant, created.message) != lc.create:
            self.fail(f"{lc.name} create: got {(created.variant, created.message)}")
        self._poll(
            lambda: self.read_tlm(lc.key_channel, "") == created.key,
            f"{lc.key_channel}={created.key} after create",
        )

        self.writer.write(lc.trigger_upsert, 1)
        updated = self._poll(
            lambda: next(
                (r for r in self._rows(lc.name) if r.message == lc.upsert.message),
                None,
            ),
            f"{lc.name} upsert to {lc.upsert.message!r}",
        )
        if updated.key != created.key:
            self.fail(f"{lc.name} upsert changed key: {created.key} -> {updated.key}")
        if updated.variant != lc.upsert.variant:
            self.fail(
                f"{lc.name} upsert variant: expected {lc.upsert.variant}, "
                f"got {updated.variant}"
            )
        rows = self._rows(lc.name)
        if len(rows) != 1:
            self.fail(f"{lc.name}: expected 1 row after upsert, got {len(rows)}")

        self.writer.write(lc.trigger_del, 1)
        self._poll(lambda: not self._rows(lc.name), f"{lc.name} delete")

    def _test_chain(self) -> None:
        self.log("=== chain: status.set -> chain_key -> status.delete ===")

        self.writer.write(CHAIN_TRIGGER, 1)
        k1 = self._poll(lambda: self.read_tlm(CHAIN_KEY, ""), "chain first key")
        self._poll(lambda: not self._rows(CHAIN_NAME), "chain first delete")

        self.writer.write(CHAIN_TRIGGER, 1)
        # Walrus binds the streamed value so we can return it only when it's
        # both non-empty AND distinct from k1 (proves the row was recreated).
        k2 = self._poll(
            lambda: (k := self.read_tlm(CHAIN_KEY, "")) and k != k1 and k,
            f"chain second key != {k1}",
        )
        self._poll(lambda: not self._rows(CHAIN_NAME), "chain second delete")
        self.log(f"chain keys differ across cycles: {k1} -> {k2}")

    def verify_sequence_execution(self) -> None:
        for lc in LIFECYCLES:
            self._run_lifecycle(lc)
        self._test_chain()

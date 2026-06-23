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

START_CMD = "start_stl_ranges_cmd"
DONE = "ranges_done"
PARENT_NAME = "STL_Range_Parent"
PARENT_KEY_TOKEN = "__PARENT_KEY__"

# ranges.end sets the end to now, so end ~= start instead of TimeStampMax.
END_NOW_TOLERANCE = 200 * sy.TimeSpan.MILLISECOND
# Start-time window slack for scheduling and driver/test clock skew.
START_SLACK = 500 * sy.TimeSpan.MILLISECOND

ARC_STL_RANGES_SOURCE = """
import ranges
import time

// ───────────────────────── Func Context ──────────────────────────
func range_test{}() {
    // Create with End
    key_val := ranges.create("RangeFunc_Create_End")
    ranges.end(key_val)

    // Create no End
    ranges.create("RangeFunc_Create")

    // RGB
    ranges.create("RangeFunc_RGB_1", "rgb(255, 0, 0)")
    ranges.create("RangeFunc_RGB_2", "rgb(0, 255, 0)")
    ranges.create("RangeFunc_RGB_3", "rgb(0, 0, 255)")

    // RGBA
    ranges.create("RangeFunc_RGBA_1", "rgba(255, 0, 0, 0.5)")
    ranges.create("RangeFunc_RGBA_2", "rgba(0, 255, 0, 0.25)")
    ranges.create("RangeFunc_RGBA_3", "rgba(0, 0, 255, 0.75)")

    // Hex
    ranges.create("RangeFunc_Hex_1", "#112233")
    ranges.create("RangeFunc_Hex_2", "#44aa66")
    ranges.create("RangeFunc_Hex_3", "#ddeeff")

    // Parent (key injected by the test)
    ranges.create("RangeFunc_Child", "rgb(10, 20, 30)", "__PARENT_KEY__")
}
start_stl_ranges_cmd -> range_test{}


// ───────────────────────── Flow Context ──────────────────────────
// Create no End
start_stl_ranges_cmd -> ranges.create("RangeFlow_Create")

// RGB
start_stl_ranges_cmd -> ranges.create("RangeFlow_RGB_1", "rgb(255, 0, 0)")
start_stl_ranges_cmd -> ranges.create("RangeFlow_RGB_2", "rgb(0, 255, 0)")
start_stl_ranges_cmd -> ranges.create("RangeFlow_RGB_3", "rgb(0, 0, 255)")

// RGBA
start_stl_ranges_cmd -> ranges.create("RangeFlow_RGBA_1", "rgba(255, 0, 0, 0.5)")
start_stl_ranges_cmd -> ranges.create("RangeFlow_RGBA_2", "rgba(0, 255, 0, 0.25)")
start_stl_ranges_cmd -> ranges.create("RangeFlow_RGBA_3", "rgba(0, 0, 255, 0.75)")

// Hex
start_stl_ranges_cmd -> ranges.create("RangeFlow_Hex_1", "#112233")
start_stl_ranges_cmd -> ranges.create("RangeFlow_Hex_2", "#44aa66")
start_stl_ranges_cmd -> ranges.create("RangeFlow_Hex_3", "#ddeeff")

// Parent (key injected by the test)
start_stl_ranges_cmd -> ranges.create("RangeFlow_Child", "rgb(10, 20, 30)", "__PARENT_KEY__")

// ────────────────────────── End Signal ───────────────────────────
time.wait{100ms} -> 1 -> ranges_done
"""


class Case(NamedTuple):
    """One created range and what it should look like afterward.

    color is the expected (r, g, b, a) or None when the range has no color.
    ends_now is True when ranges.end closed the range (end ~= start), False when
    the range is left open (end == TimeStampMax).
    """

    name: str
    color: tuple[int, int, int, float] | None
    ends_now: bool


CASES: list[Case] = [
    Case("RangeFunc_Create_End", None, True),
    Case("RangeFunc_Create", None, False),
    Case("RangeFunc_RGB_1", (255, 0, 0, 1.0), False),
    Case("RangeFunc_RGB_2", (0, 255, 0, 1.0), False),
    Case("RangeFunc_RGB_3", (0, 0, 255, 1.0), False),
    Case("RangeFunc_RGBA_1", (255, 0, 0, 0.5), False),
    Case("RangeFunc_RGBA_2", (0, 255, 0, 0.25), False),
    Case("RangeFunc_RGBA_3", (0, 0, 255, 0.75), False),
    Case("RangeFunc_Hex_1", (0x11, 0x22, 0x33, 1.0), False),
    Case("RangeFunc_Hex_2", (0x44, 0xAA, 0x66, 1.0), False),
    Case("RangeFunc_Hex_3", (0xDD, 0xEE, 0xFF, 1.0), False),
    Case("RangeFunc_Child", (10, 20, 30, 1.0), False),
    Case("RangeFlow_Create", None, False),
    Case("RangeFlow_RGB_1", (255, 0, 0, 1.0), False),
    Case("RangeFlow_RGB_2", (0, 255, 0, 1.0), False),
    Case("RangeFlow_RGB_3", (0, 0, 255, 1.0), False),
    Case("RangeFlow_RGBA_1", (255, 0, 0, 0.5), False),
    Case("RangeFlow_RGBA_2", (0, 255, 0, 0.25), False),
    Case("RangeFlow_RGBA_3", (0, 0, 255, 0.75), False),
    Case("RangeFlow_Hex_1", (0x11, 0x22, 0x33, 1.0), False),
    Case("RangeFlow_Hex_2", (0x44, 0xAA, 0x66, 1.0), False),
    Case("RangeFlow_Hex_3", (0xDD, 0xEE, 0xFF, 1.0), False),
    Case("RangeFlow_Child", (10, 20, 30, 1.0), False),
]

CHILD_NAMES = ["RangeFunc_Child", "RangeFlow_Child"]

NAMES = [c.name for c in CASES]


class StlRanges(ArcCase):
    """Test ranges.create and ranges.end in both func and flow form.

    A single trigger runs the func body and fires every flow create, producing
    ranges with and without an explicit end, with rgb/rgba/hex colors, and under
    a parent. Each created range is verified for start, end, color, and parent.
    """

    arc_source = ARC_STL_RANGES_SOURCE
    arc_name_prefix = "ArcStlRanges"
    start_cmd_channel = START_CMD
    subscribe_channels = [DONE]

    def setup(self) -> None:
        self._parent = self.client.ranges.create(
            name=PARENT_NAME,
            time_range=sy.TimeRange(sy.TimeStamp.now(), sy.TimeStamp.MAX),
        )
        self.arc_source = ARC_STL_RANGES_SOURCE.replace(
            PARENT_KEY_TOKEN, str(self._parent.key)
        )
        create_virtual_channel(self.client, DONE, sy.DataType.UINT8)
        self._before = sy.TimeStamp.now()
        super().setup()

    def teardown(self) -> None:
        with self._try_to("delete created ranges"):
            keys = [
                r.key for r in self.client.ranges.retrieve(names=NAMES + [PARENT_NAME])
            ]
            if keys:
                self.client.ranges.delete(keys)
        super().teardown()

    def _retrieve_ranges(self) -> dict[str, sy.Range]:
        found = {r.name: r for r in self.client.ranges.retrieve(names=NAMES)}
        missing = [n for n in NAMES if n not in found]
        if missing:
            self.fail(f"ranges not created: {missing}")
        return found

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for range_test to create every range")
        self.wait_for_eq(DONE, 1)
        after = sy.TimeStamp.now()
        found = self._retrieve_ranges()

        for case in CASES:
            self._verify_range(case, found[case.name], self._before, after)

        self._verify_parents(found)

    def _verify_parents(self, found: dict[str, sy.Range]) -> None:
        children = self.client.ontology.retrieve_children(self._parent.ontology_id)
        keys = {res.id.key for res in children}
        for name in CHILD_NAMES:
            if str(found[name].key) not in keys:
                self.fail(
                    f"{name} not parented under {PARENT_NAME}; "
                    f"parent's children = {keys}"
                )

    def _verify_range(
        self, case: Case, rng: sy.Range, before: sy.TimeStamp, after: sy.TimeStamp
    ) -> None:
        start = int(rng.time_range.start)
        end = int(rng.time_range.end)

        lo = int(before) - int(START_SLACK)
        hi = int(after) + int(START_SLACK)
        if not lo <= start <= hi:
            self.fail(
                f"{case.name}: start {start} not within the trigger window [{lo}, {hi}]"
            )

        if case.ends_now:
            if not 0 <= end - start <= int(END_NOW_TOLERANCE):
                self.fail(
                    f"{case.name}: expected end ~= start (within "
                    f"{END_NOW_TOLERANCE}), got end-start={end - start}ns"
                )
        elif end != int(sy.TimeStamp.MAX):
            self.fail(
                f"{case.name}: expected open range (end == TimeStampMax), got {end}"
            )

        self._verify_color(case, rng)

    def _verify_color(self, case: Case, rng: sy.Range) -> None:
        if case.color is None:
            if rng.color is not None:
                self.fail(f"{case.name}: expected no color, got {rng.color}")
            return
        if rng.color is None:
            self.fail(f"{case.name}: expected color {case.color}, got none")
        r, g, b, a = case.color
        got = (rng.color.r, rng.color.g, rng.color.b)
        if got != (r, g, b) or abs(rng.color.a - a) > 1e-6:
            self.fail(
                f"{case.name}: expected color {case.color}, got "
                f"({rng.color.r}, {rng.color.g}, {rng.color.b}, {rng.color.a})"
            )

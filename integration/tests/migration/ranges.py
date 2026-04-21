#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration verify: confirm ranges survived migration via API and console UI."""

from typing import Any

import numpy as np

import synnax as sy
from console.case import ConsoleCase

NpArray = np.ndarray[Any, Any]

EPOCH = sy.TimeStamp(1_000_000_000 * sy.TimeSpan.SECOND)

PARENT_NAME = "mig_range_parent"
PARENT_COLOR = "#E63946"
PARENT_TR = sy.TimeRange(EPOCH, EPOCH + 100 * sy.TimeSpan.SECOND)

CHILD_1_NAME = "mig_range_child_1"
CHILD_2_NAME = "mig_range_child_2"

CHILDREN = [
    (CHILD_1_NAME, "#457B9D", sy.TimeRange(EPOCH, EPOCH + 40 * sy.TimeSpan.SECOND)),
    (
        CHILD_2_NAME,
        "#2A9D8F",
        sy.TimeRange(
            EPOCH + 50 * sy.TimeSpan.SECOND,
            EPOCH + 90 * sy.TimeSpan.SECOND,
        ),
    ),
]

DATA_NAME = "mig_range_data"
ALIAS_NAME = "mig_range_sensor"

METADATA: dict[str, str] = {
    "operator": "migration_test",
    "location": "pad_39a",
    "status": "nominal",
}

DATA_VALUES: NpArray = np.array(
    [1.1, 2.2, 3.3, 4.4, 5.5, 6.6, 7.7, 8.8, 9.9, 10.10],
    dtype=np.float64,
)


class RangesVerify(ConsoleCase):
    """Verify range state survived migration via API and console UI."""

    parent: sy.Range

    def run(self) -> None:
        self.test_range_properties()
        self.test_metadata()
        self.test_child_ranges()
        self.test_data_access()
        self.test_range_overview()
        self.test_range_in_explorer()

    def test_range_properties(self) -> None:
        self.log("Testing: Range properties survived migration")
        self.parent = self.client.ranges.retrieve(name=PARENT_NAME)
        assert self.parent.name == PARENT_NAME
        assert self.parent.time_range.start == PARENT_TR.start, (
            f"Start mismatch: {self.parent.time_range.start} != {PARENT_TR.start}"
        )
        assert self.parent.time_range.end == PARENT_TR.end, (
            f"End mismatch: {self.parent.time_range.end} != {PARENT_TR.end}"
        )
        assert self.parent.color == PARENT_COLOR, (
            f"Color mismatch: {self.parent.color.hex()} != {PARENT_COLOR}"
        )

    def test_metadata(self) -> None:
        self.log("Testing: Metadata survived migration")
        result = self.parent.meta_data.get(list(METADATA.keys()))
        assert result == METADATA, f"Metadata mismatch: {result}"

    def test_child_ranges(self) -> None:
        self.log("Testing: Child ranges survived migration")
        children = self.parent.children
        assert len(children) == len(CHILDREN), (
            f"Expected {len(CHILDREN)} children, got {len(children)}"
        )
        by_name = {c.name: c for c in children}
        for name, color, tr in CHILDREN:
            child = by_name.get(name)
            assert child is not None, f"Child '{name}' not found"
            assert child.time_range.start == tr.start, f"{name} start mismatch"
            assert child.time_range.end == tr.end, f"{name} end mismatch"
            assert child.color == color, (
                f"{name} color mismatch: {child.color.hex()} != {color}"
            )

    def test_data_access(self) -> None:
        self.log("Testing: Data accessible through range after migration")
        alias_data = self.parent[ALIAS_NAME].to_numpy()
        assert np.array_equal(alias_data, DATA_VALUES), (
            f"Alias data mismatch: {alias_data}"
        )
        direct_data = self.parent[DATA_NAME].to_numpy()
        assert np.array_equal(direct_data, DATA_VALUES), (
            f"Direct data mismatch: {direct_data}"
        )

    def test_range_in_explorer(self) -> None:
        self.log("Testing: Ranges visible in console explorer")
        self.console.ranges.open_explorer()
        assert self.console.ranges.exists_in_explorer(PARENT_NAME), (
            f"Parent '{PARENT_NAME}' not found in explorer"
        )
        for name, _, _ in CHILDREN:
            assert self.console.ranges.exists_in_explorer(name), (
                f"Child '{name}' not found in explorer"
            )

    def test_range_overview(self) -> None:
        self.log("Testing: Range overview shows metadata and children")
        self.console.ranges.open_explorer()
        self.console.ranges.open_overview_from_explorer(CHILD_1_NAME)
        self.console.ranges.wait_for_overview(CHILD_1_NAME)
        self.console.ranges.navigate_to_parent(PARENT_NAME)
        self.console.ranges.wait_for_overview(PARENT_NAME)

        for key, value in METADATA.items():
            assert self.console.ranges.metadata_exists(key), (
                f"Metadata key '{key}' not visible in overview"
            )
            actual = self.console.ranges.get_metadata_value(key)
            assert actual == value, (
                f"Metadata '{key}': expected '{value}', got '{actual}'"
            )

        for name, _, _ in CHILDREN:
            assert self.console.ranges.child_range_exists(name), (
                f"Child '{name}' not visible in overview"
            )

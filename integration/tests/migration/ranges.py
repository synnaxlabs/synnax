#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: create ranges on old version, verify after migration."""

from abc import abstractmethod
from typing import Any

import numpy as np

import synnax as sy
from console.case import ConsoleCase
from framework.test_case import TestCase

NpArray = np.ndarray[Any, Any]

EPOCH = sy.TimeStamp(1_000_000_000 * sy.TimeSpan.SECOND)

PARENT_NAME = "mig_range_parent"
PARENT_COLOR = "#E63946"
PARENT_TR = sy.TimeRange(EPOCH, EPOCH + 100 * sy.TimeSpan.SECOND)

CHILD_1_NAME = "mig_range_child_1"
CHILD_1_COLOR = "#457B9D"
CHILD_1_TR = sy.TimeRange(EPOCH, EPOCH + 40 * sy.TimeSpan.SECOND)

CHILD_2_NAME = "mig_range_child_2"
CHILD_2_COLOR = "#2A9D8F"
CHILD_2_TR = sy.TimeRange(
    EPOCH + 50 * sy.TimeSpan.SECOND,
    EPOCH + 90 * sy.TimeSpan.SECOND,
)

CHILDREN = [
    (CHILD_1_NAME, CHILD_1_COLOR, CHILD_1_TR),
    (CHILD_2_NAME, CHILD_2_COLOR, CHILD_2_TR),
]

IDX_NAME = "mig_range_idx"
DATA_NAME = "mig_range_data"

METADATA: dict[str, str] = {
    "operator": "migration_test",
    "location": "pad_39a",
    "status": "nominal",
}

ALIAS_NAME = "mig_range_sensor"

SAMPLE_COUNT = 10
DATA_VALUES: NpArray = np.array(
    [1.1, 2.2, 3.3, 4.4, 5.5, 6.6, 7.7, 8.8, 9.9, 10.10],
    dtype=np.float64,
)


class RangesMigration(TestCase):
    """Base class defining the migration test contract for ranges.

    Subclasses must implement each test method — setup creates the state,
    verify checks it after migration.
    """

    parent: sy.Range

    def run(self) -> None:
        self.test_range_properties()
        self.test_metadata()
        self.test_child_ranges()
        self.test_data_access()

    @abstractmethod
    def test_range_properties(self) -> None: ...

    @abstractmethod
    def test_metadata(self) -> None: ...

    @abstractmethod
    def test_child_ranges(self) -> None: ...

    @abstractmethod
    def test_data_access(self) -> None: ...


class RangesSetup(RangesMigration):
    """Create ranges, metadata, children, channels, aliases, and sample data."""

    def test_range_properties(self) -> None:
        self.log("Testing: Create parent range")
        self.parent = self.client.ranges.create(
            name=PARENT_NAME,
            time_range=PARENT_TR,
            color=PARENT_COLOR,
            retrieve_if_name_exists=True,
        )
        assert self.parent.name == PARENT_NAME

    def test_metadata(self) -> None:
        self.log("Testing: Set metadata")
        self.parent.meta_data.set(METADATA)
        result = self.parent.meta_data.get(list(METADATA.keys()))
        assert result == METADATA, f"Metadata readback mismatch: {result}"

    def test_child_ranges(self) -> None:
        self.log("Testing: Create child ranges")
        for name, color, tr in CHILDREN:
            self.parent.create_child_range(name=name, time_range=tr, color=color)
        children = self.parent.children
        assert len(children) == len(CHILDREN), (
            f"Expected {len(CHILDREN)} children, got {len(children)}"
        )

    def test_data_access(self) -> None:
        self.log("Testing: Create channels, write data, and set alias")
        idx = self.client.channels.create(
            name=IDX_NAME,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        data_ch = self.client.channels.create(
            name=DATA_NAME,
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
            retrieve_if_name_exists=True,
        )

        timestamps = np.array(
            [EPOCH + i * sy.TimeSpan.SECOND for i in range(SAMPLE_COUNT)],
            dtype=np.int64,
        )
        with self.client.open_writer(
            start=PARENT_TR.start,
            channels=[idx.key, data_ch.key],
            name="mig_ranges_writer",
        ) as writer:
            writer.write({idx.key: timestamps, data_ch.key: DATA_VALUES})

        self.parent.set_alias(DATA_NAME, ALIAS_NAME)
        data = self.parent[ALIAS_NAME].to_numpy()
        assert len(data) == SAMPLE_COUNT, (
            f"Alias readback: expected {SAMPLE_COUNT} samples, got {len(data)}"
        )


class RangesVerify(ConsoleCase, RangesMigration):
    """Verify range state survived migration via API and console UI."""

    def run(self) -> None:
        # API verification
        self.test_range_properties()
        self.test_metadata()
        self.test_child_ranges()
        self.test_data_access()
        # Console UI verification — overview first
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
        # Via alias
        alias_data = self.parent[ALIAS_NAME].to_numpy()
        assert np.array_equal(alias_data, DATA_VALUES), (
            f"Alias data mismatch: {alias_data}"
        )
        # Via channel name
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

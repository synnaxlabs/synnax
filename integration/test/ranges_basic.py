#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from datetime import datetime
from test.framework.test_case import TestCase

import synnax as sy

time_format = "%Y-%m-%d %H:%M:%S.%f"

RANGE_SPAN_S = 3.456


class Ranges_Basic(TestCase):
    """
    Create a parent range and child ranges.
    """

    def setup(self) -> None:
        self.range_start = datetime.now()

    def run(self) -> None:

        time.sleep(RANGE_SPAN_S)
        self.range_end = datetime.now()

        range_name = f"{self.name}_parent"
        self._log_message(f"Creating parent range: {range_name}")

        parent_range = self.client.ranges.create(
            name=range_name,
            time_range=sy.TimeRange(
                start=self.range_start,
                end=self.range_end,
            ),
        )

        # Assume the latest one is the one we just created.
        my_ranges = self.client.ranges.retrieve(names=[range_name])
        my_range = my_ranges[0]  # Initialize target range to the first one.

        for r in my_ranges:
            if r.time_range.start >= my_range.time_range.start:
                my_range = r

        start_time = my_range.time_range.start
        end_time = my_range.time_range.end
        time_span_s = sy.TimeSpan(end_time - start_time).seconds

        # Child Range 1: By defining a parent
        child_range_1_name = f"{self.name}_child_1"
        self._log_message(f"Creating child range 2: {child_range_1_name}")
        self.client.ranges.create(
            parent=sy.ontology.ID(type="range", key=str(parent_range.key)),
            name=child_range_1_name,
            time_range=sy.TimeRange(
                start=self.range_start,
                end=self.range_end,
            ),
        )

        # Child Range 2: By creating a child-range of the parent
        child_range_2_name = f"{self.name}_child_2"
        self._log_message(f"Creating child range 2: {child_range_2_name}")
        parent_range.create_child_range(
            name=child_range_2_name,
            time_range=sy.TimeRange(
                start=self.range_start,
                end=self.range_end,
            ),
        )

        # Child Range 3: By creating a sub-range of the parent
        # Method is deprecated and will be removed in a future release.
        child_range_3_name = f"{self.name}_child_3"
        self._log_message(f"Creating child range 3: {child_range_3_name}")
        parent_range.create_sub_range(
            name=child_range_3_name,
            time_range=sy.TimeRange(
                start=self.range_start,
                end=self.range_end,
            ),
        )

        # Check we have the correct span
        if time_span_s < RANGE_SPAN_S and abs(time_span_s - RANGE_SPAN_S) < 0.01:
            self._log_message(
                f"Parent range span is {time_span_s} seconds, expected {RANGE_SPAN_S} seconds"
            )
            self.fail()
            return

        # Check we have the correct start and end times (convert to timestamps for comparison)
        expected_start = sy.TimeStamp(self.range_start)
        expected_end = sy.TimeStamp(self.range_end)
        if start_time != expected_start or end_time != expected_end:
            self._log_message(
                f"Parent range start time is {start_time}, expected {expected_start}"
            )
            self._log_message(
                f"Parent range end time is {end_time}, expected {expected_end}"
            )
            self.fail()
            return

        # Get child ranges using children property
        children = parent_range.children
        if len(children) != 3:
            self._log_message(f"Expected 3 children, got {len(children)}")
            self.fail()
            return

        child_names = {child.name for child in children}
        expected_child_names = {
            child_range_1_name,
            child_range_2_name,
            child_range_3_name,
        }
        if child_names != expected_child_names:
            self._log_message(
                f"Expected child names {expected_child_names}, got {child_names}"
            )
            self.fail()
            return

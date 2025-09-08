#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import synnax as sy
from datetime import datetime

from framework.test_case import TestCase


time_format = "%Y-%m-%d %H:%M:%S.%f"

RANGE_SPAN_S = 3.234


class Ranges_Basic(TestCase):
    """
    Check if the test case is connected to the synnax server.
    """

    def setup(self) -> None:

        self.range_start = datetime.now()
        super().setup()

    def run(self) -> None:
           
        time.sleep(RANGE_SPAN_S)
        self.range_end = datetime.now()
        
        
        range_name = f"{self.name}_parent"    
        self._log_message(f"Creating parent range: {range_name}")
        

        parent_range = self.client.ranges.create(
            name=range_name,
            time_range=sy.TimeRange(
                start = self.range_start,
                end = self.range_end,
            ),
        )

        # Assume the latest one is the one we just created.
        my_ranges = self.client.ranges.retrieve(names=[range_name])
        my_range = my_ranges[0] # Initialize target range to the first one.

        for r in my_ranges:
            if r.time_range.start >= my_range.time_range.start:
                my_range = r

        start_time = my_range.time_range.start
        end_time = my_range.time_range.end
        time_span_s = sy.TimeSpan(end_time - start_time).seconds

        # Child Range 1: By creating a sub-range of the parent
        child_range_1_name = f"{self.name}_child_1"
        self._log_message(f"Creating child range 1: {child_range_1_name}")
        parent_range.create_sub_range(
            name=child_range_1_name,
            time_range=sy.TimeRange(
                start = self.range_start,
                end = self.range_end,
            ),
        )

        # Child Range 2: By defining a parent
        child_range_2_name = f"{self.name}_child_2"
        self._log_message(f"Creating child range 2: {child_range_2_name}")
        self.client.ranges.create(
            parent=sy.ontology.ID(type="range", key=str(parent_range.key)),
            name=child_range_2_name,
            time_range=sy.TimeRange(
                start = self.range_start,
                end = self.range_end,
            ),
        )

        """
        Check Range Properties
        """
        # Check we have the correct span
        if time_span_s < RANGE_SPAN_S and abs(time_span_s - RANGE_SPAN_S) < 0.01:
            self._log_message(f"Parent range span is {time_span_s} seconds, expected {RANGE_SPAN_S} seconds")
            self.fail()
            return False
        
        # Check we have the correct start and end times (convert to timestamps for comparison)
        expected_start = sy.TimeStamp(self.range_start)
        expected_end = sy.TimeStamp(self.range_end)
        if start_time != expected_start or end_time != expected_end:
            self._log_message(f"Parent range start time is {start_time}, expected {expected_start}")
            self._log_message(f"Parent range end time is {end_time}, expected {expected_end}")
            self.fail()
            return False
        
        # Get child ranges using ontology
        children = self.client.ontology.retrieve_children(my_range.ontology_id)
        if len(children) != 2:
            self._log_message(f"Expected 2 children, got {len(children)}")
            self.fail()
            return False

        child_names = []
        for child in children:
            child_names.append(child.name)
        
        if child_names != [child_range_1_name, child_range_2_name]:
            self._log_message(f"Expected child names {child_range_1_name} and {child_range_2_name}, got {child_names}")
            self.fail()
            return False


    def teardown(self) -> None:
        """
        Teardown the test case.
        """

        # Always call super() last
        super().teardown()

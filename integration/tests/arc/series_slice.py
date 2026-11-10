#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_virtual_channels
from tests.arc.arc import ArcCase

ARC_SERIES_SLICE_SOURCE = """
func slice_values() {
    data := [1.0, 2.5, 3.7, 4.2]
    sub := data[1:3]
    typed series f64 := data[0:2]
    slice_first_out = sub[0]
    slice_typed_out = typed[1]
    sum f64 := 0.0
    for v := sub {
        sum = sum + v
    }
    slice_sum_out = sum
}

start_series_slice_cmd -> slice_values()
"""

OUTPUTS = ["slice_first_out", "slice_typed_out", "slice_sum_out"]


class SeriesSlice(ArcCase):
    """Test that a slice of a series is a series, with and without a declared type.
    The Arc function slices a series literal, then indexes and iterates the slices."""

    arc_source = ARC_SERIES_SLICE_SOURCE
    arc_name_prefix = "ArcSeriesSlice"
    start_cmd_channel = "start_series_slice_cmd"
    subscribe_channels = OUTPUTS

    def setup(self) -> None:
        create_virtual_channels(
            self.client, [(name, sy.DataType.FLOAT64) for name in OUTPUTS]
        )
        super().setup()

    def verify_sequence_execution(self) -> None:
        self.log("[untyped] Waiting for sub[0] == 2.5")
        self.wait_for_near("slice_first_out", 2.5, tolerance=0.01)

        self.log("[typed] Waiting for typed[1] == 2.5")
        self.wait_for_near("slice_typed_out", 2.5, tolerance=0.01)

        self.log("[iterate] Waiting for the sum of sub == 6.2")
        self.wait_for_near("slice_sum_out", 6.2, tolerance=0.01)

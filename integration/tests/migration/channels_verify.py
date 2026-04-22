#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration verify: confirm channels and data survived migration."""

import numpy as np

import synnax as sy
from framework.test_case import TestCase
from tests.migration.channels_setup import DATA_CHANNELS, IDX_NAME


class ChannelsVerify(TestCase):
    """Verify channels exist with correct types and data after migration."""

    def run(self) -> None:
        self.log("Testing: Channel types")
        idx = self.client.channels.retrieve(IDX_NAME)
        assert idx.data_type == sy.DataType.TIMESTAMP, (
            f"Expected TIMESTAMP, got {idx.data_type}"
        )
        assert idx.is_index, "Expected index channel"

        for name, expected_type, _ in DATA_CHANNELS:
            ch = self.client.channels.retrieve(name)
            assert ch.data_type == expected_type, (
                f"{name}: expected {expected_type}, got {ch.data_type}"
            )
            assert ch.index == idx.key, (
                f"{name}: expected index={idx.key}, got {ch.index}"
            )

        self.log("Testing: Data integrity")
        time_range = sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now())
        keys = [self.client.channels.retrieve(name).key for name, _, _ in DATA_CHANNELS]
        frame = self.client.read(time_range, keys)

        for key, (name, _, expected) in zip(keys, DATA_CHANNELS):
            data = frame[key].to_numpy()
            assert np.array_equal(data, expected), f"{name}: data mismatch: {data}"

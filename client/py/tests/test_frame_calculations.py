#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import numpy as np
import pytest

import synnax as sy

@pytest.mark.framer
@pytest.mark.calculations
class TestCalculatedChannels:
    def test_basic_calc_channel(self, client: sy.Synnax):
        """Should correctly create and read from a basic calculated channel using streaming"""
        timestamp_channel = client.channels.create(
            name="test_timestamp",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )

        src_channels = client.channels.create(
            [
                sy.Channel(
                    name="test_a",
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
                sy.Channel(
                    name="test_b",
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
            ]
        )

        calc_channel = client.channels.create(
            name="test_calc",
            data_type=sy.DataType.FLOAT32,
            expression="return test_a + test_b",
            requires=[src_channels[0].key, src_channels[1].key],
        )

        # Add a small delay to ensure channels are properly registered
        time.sleep(0.1)

        start = sy.TimeStamp.now()
        timestamps = [start]
        value = np.array(
            [2.0],
            dtype=np.float32,
        )

        with client.open_streamer(calc_channel.key) as streamer:
            time.sleep(0.1)

            with client.open_writer(
                start,
                [timestamp_channel.key, src_channels[0].key, src_channels[1].key],
            ) as writer:
                writer.write(
                    {
                        timestamp_channel.key: timestamps,
                        src_channels[0].key: value / 2,
                        src_channels[1].key: value / 2,
                    }
                )
                writer.commit()  # Explicitly commit the write

                # Increase timeout and add retry logic
                max_retries = 3
                for _ in range(max_retries):
                    frame = streamer.read(timeout=2)  # Increased timeout
                    if frame is not None:
                        break
                    time.sleep(0.1)  # Small delay between retries

                assert frame is not None
                assert np.array_equal(frame[calc_channel.key], value)

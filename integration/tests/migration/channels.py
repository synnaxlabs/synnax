#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: create channels and write known data on the old version."""

import numpy as np
import synnax as sy

from framework.test_case import TestCase


class ChannelsSetup(TestCase):
    """Create channels and write known sample data for migration verification."""

    def run(self) -> None:
        self.test_create_channels()
        self.test_write_data()

    def test_create_channels(self) -> None:
        self.log("Testing: Create channels")
        client = self.client

        self.idx = client.channels.create(
            name="mig_channels_idx",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.float_ch = client.channels.create(
            name="mig_channels_float32",
            data_type=sy.DataType.FLOAT32,
            index=self.idx.key,
            retrieve_if_name_exists=True,
        )
        self.int_ch = client.channels.create(
            name="mig_channels_int64",
            data_type=sy.DataType.INT64,
            index=self.idx.key,
            retrieve_if_name_exists=True,
        )

    def test_write_data(self) -> None:
        self.log("Testing: Write sample data")
        start = sy.TimeStamp.now()
        timestamps = np.array(
            [
                start,
                start + 1 * sy.TimeSpan.SECOND,
                start + 2 * sy.TimeSpan.SECOND,
                start + 3 * sy.TimeSpan.SECOND,
                start + 4 * sy.TimeSpan.SECOND,
            ],
            dtype=np.int64,
        )
        float_values = np.array([1.5, 2.7, 3.14, 4.0, 5.55], dtype=np.float32)
        int_values = np.array([10, 20, 30, 40, 50], dtype=np.int64)

        with self.client.open_writer(
            start=start,
            channels=[self.idx.key, self.float_ch.key, self.int_ch.key],
            name="mig_channels_writer",
        ) as writer:
            writer.write(
                {
                    self.idx.key: timestamps,
                    self.float_ch.key: float_values,
                    self.int_ch.key: int_values,
                }
            )
            writer.commit()


class ChannelsVerify(TestCase):
    """Verify channels exist with correct types and data after migration."""

    def run(self) -> None:
        self.test_channel_types()
        self.test_data_integrity()

    def test_channel_types(self) -> None:
        self.log("Testing: Channel types")
        idx = self.client.channels.retrieve("mig_channels_idx")
        assert (
            idx.data_type == sy.DataType.TIMESTAMP
        ), f"Expected TIMESTAMP, got {idx.data_type}"
        assert idx.is_index, "Expected index channel"
        float_ch = self.client.channels.retrieve("mig_channels_float32")
        assert (
            float_ch.data_type == sy.DataType.FLOAT32
        ), f"Expected FLOAT32, got {float_ch.data_type}"
        int_ch = self.client.channels.retrieve("mig_channels_int64")
        assert (
            int_ch.data_type == sy.DataType.INT64
        ), f"Expected INT64, got {int_ch.data_type}"

    def test_data_integrity(self) -> None:
        self.log("Testing: Data integrity")
        float_ch = self.client.channels.retrieve("mig_channels_float32")
        int_ch = self.client.channels.retrieve("mig_channels_int64")
        time_range = sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now())
        frame = self.client.read(time_range, [float_ch.key, int_ch.key])

        float_data = frame[float_ch.key].to_numpy()
        expected_floats = np.array([1.5, 2.7, 3.14, 4.0, 5.55], dtype=np.float32)
        assert (
            len(float_data) >= 5
        ), f"Expected at least 5 float samples, got {len(float_data)}"
        assert np.allclose(
            float_data[-5:], expected_floats, atol=1e-5
        ), f"Float data mismatch: {float_data[-5:]} != {expected_floats}"
        int_data = frame[int_ch.key].to_numpy()
        expected_ints = np.array([10, 20, 30, 40, 50], dtype=np.int64)
        assert (
            len(int_data) >= 5
        ), f"Expected at least 5 int samples, got {len(int_data)}"
        assert np.array_equal(
            int_data[-5:], expected_ints
        ), f"Int data mismatch: {int_data[-5:]} != {expected_ints}"

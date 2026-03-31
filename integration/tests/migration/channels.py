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

IDX_NAME = "mig_channels_idx"
FLOAT_NAME = "mig_channels_float32"
INT_NAME = "mig_channels_int64"
EXPECTED_FLOATS = np.array([1.5, 2.7, 3.14, 4.0, 5.55], dtype=np.float32)
EXPECTED_INTS = np.array([10, 20, 30, 40, 50], dtype=np.int64)
SAMPLE_COUNT = len(EXPECTED_FLOATS)


class ChannelsSetup(TestCase):
    """Create channels and write known sample data for migration verification."""

    def run(self) -> None:
        self.test_create_channels()
        self.test_write_data()

    def test_create_channels(self) -> None:
        self.log("Testing: Create channels")
        client = self.client

        self.idx = client.channels.create(
            name=IDX_NAME,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.float_ch = client.channels.create(
            name=FLOAT_NAME,
            data_type=sy.DataType.FLOAT32,
            index=self.idx.key,
            retrieve_if_name_exists=True,
        )
        self.int_ch = client.channels.create(
            name=INT_NAME,
            data_type=sy.DataType.INT64,
            index=self.idx.key,
            retrieve_if_name_exists=True,
        )

    def test_write_data(self) -> None:
        self.log("Testing: Write sample data")
        start = sy.TimeStamp.now()
        timestamps = np.array(
            [start + i * sy.TimeSpan.SECOND for i in range(SAMPLE_COUNT)],
            dtype=np.int64,
        )

        with self.client.open_writer(
            start=start,
            channels=[self.idx.key, self.float_ch.key, self.int_ch.key],
            name="mig_channels_writer",
        ) as writer:
            writer.write(
                {
                    self.idx.key: timestamps,
                    self.float_ch.key: EXPECTED_FLOATS,
                    self.int_ch.key: EXPECTED_INTS,
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
        idx = self.client.channels.retrieve(IDX_NAME)
        assert (
            idx.data_type == sy.DataType.TIMESTAMP
        ), f"Expected TIMESTAMP, got {idx.data_type}"
        assert idx.is_index, "Expected index channel"

        float_ch = self.client.channels.retrieve(FLOAT_NAME)
        assert (
            float_ch.data_type == sy.DataType.FLOAT32
        ), f"Expected FLOAT32, got {float_ch.data_type}"
        assert (
            float_ch.index == idx.key
        ), f"Expected float index={idx.key}, got {float_ch.index}"

        int_ch = self.client.channels.retrieve(INT_NAME)
        assert (
            int_ch.data_type == sy.DataType.INT64
        ), f"Expected INT64, got {int_ch.data_type}"
        assert (
            int_ch.index == idx.key
        ), f"Expected int index={idx.key}, got {int_ch.index}"

    def test_data_integrity(self) -> None:
        self.log("Testing: Data integrity")
        float_ch = self.client.channels.retrieve(FLOAT_NAME)
        int_ch = self.client.channels.retrieve(INT_NAME)
        time_range = sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now())
        frame = self.client.read(time_range, [float_ch.key, int_ch.key])

        float_data = frame[float_ch.key].to_numpy()
        assert (
            len(float_data) >= SAMPLE_COUNT
        ), f"Expected >= {SAMPLE_COUNT} float samples, got {len(float_data)}"
        assert np.allclose(
            float_data[-SAMPLE_COUNT:], EXPECTED_FLOATS, atol=1e-5
        ), f"Float data mismatch: {float_data[-SAMPLE_COUNT:]}"

        int_data = frame[int_ch.key].to_numpy()
        assert (
            len(int_data) >= SAMPLE_COUNT
        ), f"Expected >= {SAMPLE_COUNT} int samples, got {len(int_data)}"
        assert np.array_equal(
            int_data[-SAMPLE_COUNT:], EXPECTED_INTS
        ), f"Int data mismatch: {int_data[-SAMPLE_COUNT:]}"

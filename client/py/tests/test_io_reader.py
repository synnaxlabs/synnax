#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pathlib

import numpy as np
import pytest
from nptdms import ChannelObject, GroupObject, RootObject, TdmsWriter

from synnax.io import IO_FACTORY
from synnax.io.protocol import ColumnFileReader, RowFileReader
from synnax.io.tdms import TDMSReader

BASE_DIR = pathlib.Path("./tests/testdata/io")

VALID_FILE_CHANNELS = [
    "thermoCouple01",
    "gseTimestamp01",
    "autoOn",
    "strainGauge22",
]


@pytest.mark.io
@pytest.mark.parametrize(
    "path",
    [
        "valid.csv",
        "valid_extra_headers.csv",
        "valid_semicolon_separated.csv",
    ],
)
class TestRowFileReaders:
    @pytest.fixture
    def valid_file(self, path):
        return IO_FACTORY.new_reader(BASE_DIR / pathlib.Path(path))

    def test_channels(self, valid_file: RowFileReader):
        """It should correctly return a list of the channel names in the file"""
        assert [c.name for c in valid_file.channels()] == VALID_FILE_CHANNELS

    def test_num_samples(self, valid_file: RowFileReader):
        """It should return the approximate number of samples in the file"""
        ns = valid_file.nsamples()
        assert ns >= 20 and ns <= 30

    def test_first_sample(self, valid_file: RowFileReader):
        """It should return the first sample in the file"""
        valid_file.seek_first()
        d = valid_file.read()
        assert d["thermoCouple01"].to_numpy()[0] == 1.0

    def test_read(self, valid_file: RowFileReader):
        """It should correctly iterate over the samples in the file"""
        valid_file.set_chunk_size(1)
        valid_file.seek_first()
        count = 0
        for d in valid_file:
            assert len(d) == 1
            count += 1
        assert count == 4


@pytest.mark.io
class TestAllStringFirstCol:
    @pytest.fixture
    def valid_file(self):
        return IO_FACTORY.new_reader(BASE_DIR / "valid_all_string_first_col.csv")

    def test_channels(self, valid_file: RowFileReader):
        """It should correctly return a list of the channel names in the file"""
        assert [c.name for c in valid_file.channels()] == [
            "randStringChan",
            *VALID_FILE_CHANNELS,
        ]

    def test_num_samples(self, valid_file: RowFileReader):
        """It should return the approximate number of samples in the file"""
        ns = valid_file.nsamples()
        assert ns >= 20 and ns <= 30

    def test_first_sample(self, valid_file: RowFileReader):
        """It should return the first sample in the file"""
        valid_file.seek_first()
        d = valid_file.read()
        assert d["thermoCouple01"].to_numpy()[0] == 1.0

    def test_read(self, valid_file: RowFileReader):
        """It should correctly iterate over the samples in the file"""
        valid_file.set_chunk_size(1)
        valid_file.seek_first()
        count = 0
        for d in valid_file:
            assert len(d) == 1
            count += 1
        assert count == 4


@pytest.mark.tdms
class TestTdmsReader:
    @pytest.fixture
    def create_test_file(self):
        # Create test file
        root = RootObject()
        groupA = GroupObject("groupA")
        groupB = GroupObject("groupB")
        channel0 = ChannelObject("groupA", "thermoCouple01", np.array([1, 1, 2, 3]))
        channel1 = ChannelObject("groupA", "gseTimestamp01", np.array([1, 2, 3, 4]))
        channel2 = ChannelObject("groupA", "autoOn", np.array([0, 1, 0, 1]))
        channel3 = ChannelObject(
            "groupB",
            "strainGauge22",
            np.array([150000.125, 125125152.12, 125125125.12, 1251251512.12]),
        )
        # Write it
        with TdmsWriter(f"{BASE_DIR / 'tdms'}.tdms") as writer:
            writer.write_segment(
                [root, groupA, groupB, channel0, channel1, channel2, channel3]
            )

    @pytest.fixture
    def valid_file(self, create_test_file):
        return TDMSReader(f"{BASE_DIR / 'tdms'}.tdms")

    def test_channels(self, valid_file: ColumnFileReader):
        """It should correctly return a list of the channel names in the file"""
        valid_file.set_chunk_size(1)
        assert [c.name for c in valid_file.channels()] == VALID_FILE_CHANNELS

    def test_num_samples(self, valid_file: ColumnFileReader):
        """It should return the approximate number of samples in the file"""
        valid_file.set_chunk_size(1)
        assert valid_file.nsamples() == 16

    def test_first_sample(self, valid_file: ColumnFileReader):
        """It should return the first sample in the file"""
        valid_file.seek_first()
        d = valid_file.read()
        assert d["thermoCouple01"].to_numpy()[0] == 1.0

    def test_read(self, valid_file: ColumnFileReader):
        """It should correctly iterate over the samples in the file"""
        valid_file.set_chunk_size(1)
        valid_file.seek_first()
        count = 0
        for d in valid_file:
            assert len(d) == 1
            count += 1
        assert count == 4

    def test_read_keys(self, valid_file: ColumnFileReader):
        """It should only read the fist 3 channels"""
        valid_file.set_chunk_size(2)
        valid_file.seek_first()
        for _ in range(2):
            d = valid_file.read(*VALID_FILE_CHANNELS[:3])
            assert len(d) == 2
            assert list(d.keys()) == VALID_FILE_CHANNELS[:3]

        d = valid_file.read(*VALID_FILE_CHANNELS[:3])
        assert d.empty

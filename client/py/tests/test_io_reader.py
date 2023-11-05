#  Copyright 2023 Synnax Labs, Inc.
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
VALID_FILE = BASE_DIR / "valid"

VALID_FILE_CHANNELS = [
    "thermoCouple01",
    "gseTimestamp01",
    "autoOn",
    "strainGauge22",
]


@pytest.mark.parametrize("ext", IO_FACTORY.extensions())
class TestRowFileReaders:
    @pytest.fixture
    def valid_file(self, ext):
        return IO_FACTORY.new_reader(pathlib.Path(f"{VALID_FILE}.{ext}"))

    def test_new_reader_valid_file(self, ext, valid_file: RowFileReader):
        """It should open a new reader for the given extension type"""
        assert ext in valid_file.extensions()

    def test_channels(self, valid_file: RowFileReader):
        """It should correctly return a list of the channel names in the file"""
        assert [c.name for c in valid_file.channels()] == VALID_FILE_CHANNELS

    def test_num_samples(self, valid_file: RowFileReader):
        """It should return the approximate number of samples in the file"""
        ns = valid_file.nsamples()
        assert ns > 20 and ns < 30

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

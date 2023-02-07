#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pathlib

import pytest

from synnax.io import IO_FACTORY
from synnax.io.protocol import RowFileReader

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

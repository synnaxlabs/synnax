#  Copyright 2026 Synnax Labs, Inc.
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
    def valid_file(self, path: str) -> RowFileReader:
        return IO_FACTORY.new_reader(BASE_DIR / pathlib.Path(path))

    def test_channels(self, valid_file: RowFileReader) -> None:
        """It should correctly return a list of the channel names in the file"""
        assert [c.name for c in valid_file.channels()] == VALID_FILE_CHANNELS

    def test_num_samples(self, valid_file: RowFileReader) -> None:
        """It should return the approximate number of samples in the file"""
        ns = valid_file.nsamples()
        assert ns >= 20
        assert ns <= 30

    def test_first_sample(self, valid_file: RowFileReader) -> None:
        """It should return the first sample in the file"""
        valid_file.seek_first()
        d = valid_file.read()
        assert d["thermoCouple01"].to_numpy()[0] == 1.0

    def test_read(self, valid_file: RowFileReader) -> None:
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
    def valid_file(self) -> RowFileReader:
        return IO_FACTORY.new_reader(BASE_DIR / "valid_all_string_first_col.csv")

    def test_channels(self, valid_file: RowFileReader) -> None:
        """It should correctly return a list of the channel names in the file"""
        assert [c.name for c in valid_file.channels()] == [
            "randStringChan",
            *VALID_FILE_CHANNELS,
        ]

    def test_num_samples(self, valid_file: RowFileReader) -> None:
        """It should return the approximate number of samples in the file"""
        ns = valid_file.nsamples()
        assert ns >= 20
        assert ns <= 30

    def test_first_sample(self, valid_file: RowFileReader) -> None:
        """It should return the first sample in the file"""
        valid_file.seek_first()
        d = valid_file.read()
        assert d["thermoCouple01"].to_numpy()[0] == 1.0

    def test_read(self, valid_file: RowFileReader) -> None:
        """It should correctly iterate over the samples in the file"""
        valid_file.set_chunk_size(1)
        valid_file.seek_first()
        count = 0
        for d in valid_file:
            assert len(d) == 1
            count += 1
        assert count == 4

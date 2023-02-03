#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Iterator

from pathlib import Path

import pandas as pd
from pandas.io.parsers import TextFileReader

from synnax.telem import Size
from synnax.io.matcher import new_extension_matcher
from synnax.io.protocol import ChannelMeta, ReaderType, RowFileReader, FileWriter


class CSVMatcher(new_extension_matcher(["csv"])):  # type: ignore
    ...


class CSVReader(CSVMatcher):
    """A RowReader implementation for CSV files."""

    channel_keys: list[str] | None
    chunk_size: int
    _reader: TextFileReader | None
    _path: Path
    _channels: list[ChannelMeta] | None
    _row_count: int | None

    # Doing a protocol implementation check here because
    # it's hard for pyright to handle factories that retun
    # protocol classes.
    def _(self) -> RowFileReader:
        return self

    def __init__(
        self,
        path: Path,
        keys: list[str] | None = None,
        chunk_size: int | None = None,
    ):
        self._path = path
        self.channel_keys = keys
        self._channels = None
        self._row_count = None
        self._reader = None
        self.chunk_size = chunk_size or 10 * Size.MEGABYTE

    def seek_first(self):
        self.close()
        self._reader = pd.read_csv(
            self._path,
            chunksize=self.chunk_size,
            usecols=self.channel_keys,
        )

    def channels(self) -> list[ChannelMeta]:
        if not self._channels:
            self._channels = [
                ChannelMeta(name=name, meta_data={})
                for name in pd.read_csv(self._path, nrows=0).columns
            ]
        return self._channels

    def set_chunk_size(self, chunk_size: int):
        self.chunk_size = chunk_size

    def read(self) -> pd.DataFrame:
        return next(self.reader)

    def __iter__(self) -> Iterator[pd.DataFrame]:
        return self.reader.__iter__()

    @classmethod
    def type(cls) -> ReaderType:
        return ReaderType.Row

    def path(self) -> Path:
        return self._path

    def nsamples(self) -> int:
        if not self._row_count:
            self._row_count = estimate_row_count(self._path)
        return self._row_count * len(self.channels())

    @property
    def reader(self) -> TextFileReader:
        if self._reader is None:
            self.seek_first()
        assert self._reader is not None
        return self._reader

    def close(self):
        if self._reader is not None:
            self._reader.close()
        self._reader = None


def estimate_row_count(path: Path) -> int:
    """Estimates the number of rows in a CSV file."""
    with open(path, "r") as f:
        f.readline()
        row = f.readline()
        row_size = len(row.encode("utf-8"))

    file_size = path.stat().st_size
    return file_size // row_size


class CSVWriter(CSVMatcher):
    """A Writer implementation for CSV files."""

    _path: Path
    _header: bool

    def __init__(
        self,
        path: Path,
    ):
        self._path = path
        self._header = True

    # Doing a protocol implementation check here because
    # it's hard for pyright to handle factories that retun
    # protocol classes.
    def _(self) -> FileWriter:
        return self

    def write(self, df: pd.DataFrame):
        df.to_csv(self._path, index=False, mode="a", header=self._header)
        self._header = False

    def path(self) -> Path:
        return self._path

    def close(self):
        pass

#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path
from typing import Iterator

import pandas as pd
from pandas.io.parsers import TextFileReader

from synnax.exceptions import ValidationError
from synnax.io.matcher import new_extension_matcher
from synnax.io.protocol import ChannelMeta, FileWriter, ReaderType, RowFileReader

CSVMatcher = new_extension_matcher(["csv"])


class CSVReader(CSVMatcher):  # type: ignore
    """A RowReader implementation for CSV files."""

    channel_keys: list[str] | None
    chunk_size: int

    __reader: TextFileReader | None
    _path: Path
    _channels: list[ChannelMeta] | None
    _row_count: int | None
    _calculated_skip_rows: bool
    _skip_rows: int

    # Doing a protocol implementation check here because
    # it's hard for pyright to handle factories that return
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
        self.chunk_size = chunk_size or int(5e5)
        self._channels = None
        self._row_count = None
        self._skip_rows = 0
        self._calculated_skip_rows = False
        self.__reader = None

    def seek_first(self):
        self.close()
        self.__reader = pd.read_csv(
            self._path,
            chunksize=self.chunk_size,
            usecols=self.channel_keys,
            header=0,
            skiprows=self.__get_skip_rows(),
        )

    def __get_skip_rows(self) -> int | tuple[int, int]:
        if self._calculated_skip_rows:
            return self._skip_rows

        r = pd.read_csv(
            self._path,
            chunksize=1,
            usecols=self.channel_keys,
        )
        self._skip_rows = 0

        while not self._calculated_skip_rows:
            try:
                df = next(r)
            except StopIteration:
                raise ValidationError("No valid data found in CSV file")

            # check if the first value is a string
            if isinstance(df.iloc[0, 0], str):
                self._skip_rows += 1
            else:
                self._calculated_skip_rows = True

        r.close()
        if self._skip_rows > 0:
            return 1, self._skip_rows
        return self._skip_rows

    def channels(self) -> list[ChannelMeta]:
        if not self._channels:
            cols = pd.read_csv(self._path, nrows=0).columns
            self._channels = [
                ChannelMeta(name=name.strip(), meta_data=dict()) for name in cols
            ]
        return self._channels

    def set_chunk_size(self, chunk_size: int):
        self.chunk_size = chunk_size

    def read(self) -> pd.DataFrame:
        return next(self._reader)

    def __iter__(self) -> Iterator[pd.DataFrame]:
        return self._reader.__iter__()

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
    def _reader(self) -> TextFileReader:
        if self.__reader is None:
            self.seek_first()
        assert self.__reader is not None
        return self.__reader

    def close(self):
        if self.__reader is not None:
            self.__reader.close()
        self.__reader = None


def estimate_row_count(path: Path) -> int:
    """Estimates the number of rows in a CSV file."""
    with open(path, "r") as f:
        f.readline()
        row = f.readline()
        row_size = len(row.encode("utf-8"))

    file_size = path.stat().st_size
    return (file_size // row_size) - 1


class CSVWriter(CSVMatcher):  # type: ignore
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
    # it's hard for pyright to handle factories that return
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

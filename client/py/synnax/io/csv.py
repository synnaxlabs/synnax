from datetime import datetime
from pathlib import Path

import pandas as pd
from pandas.io.parsers import TextFileReader

from synnax.io.matcher import new_extension_matcher
from synnax.io.protocol import ChannelMeta, ReaderType

CSVMatcher = new_extension_matcher(["csv"])


class CSVReader(CSVMatcher):
    """A RowReader implementation for CSV files.
    """
    reader: TextFileReader
    _path: Path
    _channels: list[ChannelMeta] | None
    _row_count: int | None
    channel_keys: list[str] | None

    def __init__(self,
                 path: Path,
                 channel_keys: list[str] = None,
                 chunk_size: int = None,
                 ):
        self._path = path
        self.channel_keys = channel_keys
        self._channels = None
        self._row_count = None

    def channels(self) -> list[ChannelMeta]:
        if not self._channels:
            self._channels = [ChannelMeta(name=name, meta_data={}) for name in
                              pd.read_csv(self._path, nrows=0).columns]
        return self._channels

    def set_chunk_size(self, chunk_size: int):
        self.reader = pd.read_csv(
            self._path,
            chunksize=chunk_size,
            usecols=self.channel_keys,
        )

    def read(self) -> pd.DataFrame:
        return next(self.reader)

    @classmethod
    def type(cls) -> ReaderType:
        return ReaderType.Row

    def path(self) -> Path:
        return self._path

    @property
    def nsamples(self) -> int:
        if not self._row_count:
            self._row_count = estimate_row_count(self._path)
        return self._row_count * len(self.channels())


def estimate_row_count(path: Path) -> int:
    """Estimates the number of rows in a CSV file.
    """
    with open(path, "r") as f:
        f.readline()
        row = f.readline()
        row_size = len(row.encode("utf-8"))

    file_size = path.stat().st_size
    return file_size // row_size


class CSVWriter(CSVMatcher):
    _path: Path

    def __init__(
        self,
        path: Path,
    ):
        self._path = path

    def write(self, df: pd.DataFrame):
        df.to_csv(self._path, index=False)

    def path(self) -> Path:
        return self._path

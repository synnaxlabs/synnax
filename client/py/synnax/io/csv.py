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
    channel_keys: list[str] | None

    def __init__(self,
                 path: Path,
                 channel_keys: list[str] = None,
                 chunk_size: int = None,
                 ):
        self._path = path
        self.channel_keys = channel_keys
        self._channels = None

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

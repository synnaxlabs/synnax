from pathlib import Path

import pandas as pd
from pandas.io.parsers import TextFileReader

from synnax.io.protocol import ChannelMeta, ReaderType


class CSVReader:
    """A RowReader implementation for CSV files.
    """
    reader: TextFileReader
    path: Path
    _channels: list[ChannelMeta] | None
    channel_keys: list[str] | None

    def __init__(self,
                 path: Path,
                 channel_keys: list[str] = None,
                 chunk_size: int = None,
                 ):
        self.path = path
        self.channel_keys = channel_keys
        self._channels = None

    def channels(self) -> list[ChannelMeta]:
        if not self._channels:
            self._channels = [ChannelMeta(name=name, meta_data={}) for name in
                              pd.read_csv(self.path, nrows=0).columns]
        return self._channels

    def set_chunk_size(self, chunk_size: int):
        self.reader = pd.read_csv(
            self.path,
            chunksize=chunk_size,
            usecols=self.channel_keys,
        )

    def read(self) -> pd.DataFrame:
        return next(self.reader)

    @classmethod
    def type(cls) -> ReaderType:
        return ReaderType.Row

    @classmethod
    def extensions(cls) -> list[str]:
        return ["csv"]

    @classmethod
    def match(cls, path: Path) -> bool:
        return path.suffix[1:] in cls.extensions()

from pathlib import Path

import pandas as pd
from pandas.io.parsers import TextFileReader

from synnax.io.reader import ChannelMeta


class CSVReader:
    reader: TextFileReader
    path: Path
    _channels: list[ChannelMeta] | None

    def __init__(self,
                 path: Path,
                 channel_keys: list[str] = None,
                 chunk_size: int = 1000,
                 ):
        self.path = path
        self.reader = pd.read_csv(
            path,
            chunksize=chunk_size,
            usecols=channel_keys,
        )
        self._channels = None

    def channels(self) -> list[ChannelMeta]:
        if not self._channels:
            self._channels = [ChannelMeta(name=name, meta_data={}) for name in
                              pd.read_csv(self.path, nrows=0).columns]
        return self._channels

    def read(self) -> pd.DataFrame:
        return next(self.reader)

    @classmethod
    def extensions(cls) -> list[str]:
        return ["csv"]

    @classmethod
    def match(cls, path: Path) -> bool:
        return path.suffix[1:] in cls.extensions()

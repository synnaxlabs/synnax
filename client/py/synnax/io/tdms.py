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
from synnax.exceptions import ValidationError

from synnax.io.matcher import new_extension_matcher
from synnax.io.protocol import ChannelMeta, ReaderType, File, BaseReader, ColumnFileReader

TDMLMatcher = new_extension_matcher(["tdml"])


class TDMLReader(TDMLMatcher):   # type: ignore
    """A ColReader implementation for TDML files."""

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
    def _(self) -> ColumnFileReader:
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
    
    def channels(self) -> list[ChannelMeta]:
        """:returns : a list of channel metadata for the file."""
        ...

    @classmethod
    def type(cls) -> ReaderType:
        """:returns : the type of reader."""
        ...

    def path(self) -> Path:
        """:returns: the path to the file."""
        ...

    def nsamples(self) -> int:
        """:returns: the number of samples in the file."""
        ...

    def seek_first(self):
        """Seeks the reader to the first  sampele in the file."""
        ...
    
    def read(self, *keys: str) -> pd.DataFrame:
        """Reads a dataframe with chunk size * number of columns samples. The returned
        dataframe contains columns for each key in keys.
        """
        ...
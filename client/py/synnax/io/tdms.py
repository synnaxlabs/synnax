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

from nptdms import TdmsFile, TdmsGroup, TdmsChannel

from math import ceil

from synnax.io.matcher import new_extension_matcher
from synnax.io.protocol import ChannelMeta, ReaderType, File, BaseReader, ColumnFileReader

TDMSMatcher = new_extension_matcher(["tdms"])


class TDMSReader(TDMSMatcher):   # type: ignore
    """A ColReader implementation for TDMS files."""

    channel_keys: list[str] | None
    chunk_size: int

    _chunk_location: int
    _n_channel_chunks: int
    
    _path: Path
    _channels: list[ChannelMeta] | None

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
        self._chunk_location = 0
        self._n_channel_chunks = None
    
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
        data = dict()
        
        with TdmsFile.open(self._path) as tdms_file:
            # read entire file, return empty DF
            if self._chunk_location > self.n_channel_chunks:
                return pd.DataFrame()
            
            for group in tdms_file.groups():
                for channel in group.channels():
                    data[channel.name] = channel[self._chunk_location*self.chunk_size:(self._chunk_location + 1)*self.chunk_size]
            self._chunk_location += 1
        
        return pd.DataFrame(data)
    
    @property
    def n_channel_chunks(self) -> int:
        """Returns number of chunk reads we can make from the TDMS file
        """
        if self._n_channel_chunks is None:
            # assume every channel has the same number of data
            with TdmsFile.open(self._path) as tdms_file:
                group0 = tdms_file.groups()[0]
                channel0 = group0.channels()[0]
                self._n_channel_chunks = ceil(len(channel0) / self.chunk_size)
                
        assert self._n_channel_chunks is not None
        return self._n_channel_chunks
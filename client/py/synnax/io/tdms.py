#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from math import ceil
from pathlib import Path
from typing import Iterator

import pandas as pd
from nptdms import TdmsChannel, TdmsFile, TdmsGroup
from pandas.io.parsers import TextFileReader

from synnax.exceptions import ValidationError
from synnax.io.matcher import new_extension_matcher
from synnax.io.protocol import (
    BaseReader,
    ChannelMeta,
    ColumnFileReader,
    File,
    ReaderType,
)

TDMSMatcher = new_extension_matcher(["tdms"])


class TDMSReader(TDMSMatcher):  # type: ignore
    """A ColReader implementation for TDMS files."""

    channel_keys: set[str]
    chunk_size: int

    _current_chunk: int
    _n_chunks: int | None

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
        self.chunk_size = chunk_size or int(1e5)

        self._channels = None
        self._current_chunk = 0
        self._n_chunks = None
        if keys is not None:
            self.channel_keys = set(keys)
        else:
            self.set_keys_from_file()

    def channels(self) -> list[ChannelMeta]:
        """:returns : a list of channel metadata for the file."""
        if self._channels is not None:
            return self._channels

        self._channels: list[ChannelMeta] = list()
        with TdmsFile.open(self._path) as tdms_file:
            for group in tdms_file.groups():
                for channel in group.channels():
                    self._channels.append(
                        ChannelMeta(name=channel.name, meta_data=dict())
                    )

        return self._channels

    def set_chunk_size(self, chunk_size: int):
        self.chunk_size = chunk_size

    def set_keys(self, keys: list[str]):
        self.channel_keys = set(keys)

    def set_keys_from_file(self) -> set[str]:
        self.channel_keys = set([ch.name for ch in self.channels()])
        return self.channel_keys

    @classmethod
    def type(cls) -> ReaderType:
        """:returns : the type of reader."""
        return ReaderType.Column

    def path(self) -> Path:
        """:returns: the path to the file."""
        return self._path

    def nsamples(self) -> int:
        """:returns: the number of samples in the file."""
        return self.chunk_size * self.n_chunks * len(self.channels())

    def seek_first(self):
        """Seeks the reader to the first sample in the file."""
        self._current_chunk = 0

    def read(self, *keys: str) -> pd.DataFrame:
        """Reads a dataframe with chunk size * number of columns samples. The returned
        dataframe contains columns for each key in keys.
        """
        # if we already read everything, return an empty chunk
        if self._current_chunk >= self.n_chunks:
            return pd.DataFrame()

        # if keys is empty, use default keys
        keys: set[str] = self.channel_keys if (len(keys) == 0) else set(keys)

        # https://nptdms.readthedocs.io/en/stable/reading.html
        # https://github.com/adamreeve/npTDMS/issues/263
        # https://nptdms.readthedocs.io/en/stable/reading.html
        data = dict()
        with TdmsFile.open(self._path) as tdms_file:
            for group in tdms_file.groups():
                for channel in group.channels():
                    if channel.name in keys:
                        data[channel.name] = channel[
                            self._current_chunk
                            * self.chunk_size : (self._current_chunk + 1)
                            * self.chunk_size
                        ]
            self._current_chunk += 1

        return pd.DataFrame(data)

    def __iter__(self) -> Iterator[pd.DataFrame]:
        return (self.read() for _ in range(self.n_chunks))

    @property
    def n_chunks(self) -> int:
        """Returns number of chunks in a channel.
        Assumes every channel has the same number of chunks
        """
        if self._n_chunks is None:
            # assume every channel has the same number of data
            with TdmsFile.open(self._path) as tdms_file:
                group0 = tdms_file.groups()[0]
                channel0 = group0.channels()[0]
                self._n_chunks = ceil(len(channel0) / self.chunk_size)

        assert self._n_chunks is not None
        return self._n_chunks

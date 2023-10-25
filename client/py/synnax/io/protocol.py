#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum
from pathlib import Path
from typing import Iterator, Protocol

from pandas import DataFrame

from synnax.io.meta import ChannelMeta


class ReaderType(Enum):
    Row = "row"
    Column = "column"


class Matcher(Protocol):
    @classmethod
    def extensions(cls) -> list[str]:
        """:returns: a list of file extensions that the reader can read. Sometimes the
        list is not exhaustive, and is mostly used for informational purposes. For
        accurate file extension matching, use the :meth:`match` method.
        """
        ...

    @classmethod
    def match(cls, path: Path) -> bool:
        """:returns: whether the reader can read the file at the given path."""
        ...


class Closer(Protocol):
    """Closer is a closable buffer"""

    def close(self):
        """Closes the buffer."""
        ...


class File(Matcher, Closer, Protocol):
    """File is the base protocol for all file protocols. It's used to provide common
    information and utilities for all file protocols."""

    def path(self) -> Path:
        """:returns: the path to the file."""
        ...


class BaseReader(File, Protocol):
    """The base reader protocol that all other reader protocols must implement.

    :param path: The path to the file to read.
    :param keys: A list of channel keys to read. If None, all channels are read.
    :param chunk_size: The number of rows to read at a time. If None, all rows are read.
    """

    def __init__(
        self,
        path: Path,
        keys: list[str] | None = None,
        chunk_size: int | None = None,
    ):
        ...

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


class RowFileReader(BaseReader, Protocol):
    """Row readers implement a strategy that reads a file row.py by row.py. Because Synnax
    is optimized for ingesting data in a columnar format, Row readers should
    only be used when files cannot be read using a :class:`ColumnReader` strategy (e.g.
    csv files).
    """

    def set_chunk_size(self, chunk_size: int):
        """Set the chunk size for the reader. It's generally unsafe to assume the reader
        position after calling set_chunk_size, so it's recommended to call reset
        afterwards.
        """
        ...

    def read(self) -> DataFrame:
        """Read returns a dataframe with chunk size * number of columns samples. The
        returned dataframe is guaranteed to contain a column for all channels in the file.
        """
        ...

    def __iter__(self) -> Iterator[DataFrame]:
        """Iterates over the file, returning a dataframe with chunk size * number of
        columns samples. The returned dataframe is guaranteed to contain a column for all
        channels in the file.
        """
        ...


class ColumnFileReader(BaseReader, Protocol):
    """Column readers implement a strategy that reads a file column by column. Synnax
    is optimized for ingesting data in a columnar format, so Column readers should
    be used whenever possible.
    """

    def read(self, *keys: str) -> DataFrame:
        """Reads a dataframe with chunk size * number of columns samples. The returned
        dataframe contains columns for each key in keys.
        """
        ...


class DataFrameWriter(Closer, Protocol):
    """A protocol for writing dataframes to a buffer. This protocol is kept separate
    from file protocols because it's also possible to write dataframes to other sources,
    such as a Synnax cluster.
    """

    def write(self, df: DataFrame) -> None | bool:
        """Writes the given dataframe to the buffer."""
        ...


class FileWriter(File, DataFrameWriter, Protocol):
    """A file protocol for writing dataframes"""

    def __init__(
        self,
        path: Path,
    ):
        """Creates a new file writer.

        :param path: The path to the file to write.
        """
        ...

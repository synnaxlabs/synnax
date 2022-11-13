from enum import Enum
from typing import Protocol
from pathlib import Path

from pandas import DataFrame

from .meta import ChannelMeta


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


class BaseReader(Matcher):
    """The base reader protocol that all other reader protocols must implement.

    :param path: The path to the file to read.
    :param keys: A list of channel keys to read. If None, all channels are read.
    :param chunk_size: The number of rows to read at a time. If None, all rows are read.
    """

    def __init__(
        self,
        path: Path,
        keys: list[str] = None,
        chunk_size: int = None,
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


class RowReader(BaseReader):
    """Row readers implement a strategy that reads a file row.py by row.py. Because Synnax
    is optimized for ingesting data in a columnar format, Row readers should
    only be used when files cannot be read using a :class:`ColumnReader` strategy (e.g.
    csv files).
    """

    def set_chunk_size(self, chunk_size: int):
        """Set the chunk size for the reader."""
        ...

    def read(self) -> DataFrame:
        """Read returns a dataframe with chunk size * number of columns samples. The
        returned dataframe is guaranteed to contain a column for all channels in the file.
        """
        ...


class ColumnReader(BaseReader):
    """Column readers implement a strategy that reads a file column by column. Synnax
    is optimized for ingesting data in a columnar format, so Column readers should
    be used whenever possible.
    """

    def read(self, *keys: str) -> DataFrame:
        """Reads a dataframe with chunk size * number of columns samples. The returned
        dataframe contains columns for each key in keys.
        """
        ...


class Writer(Matcher):
    def __init__(
        self,
        path: Path,
    ):
        ...

    def write(self, df: DataFrame):
        """Writes the given dataframe to the file."""
        ...

    def path(self) -> Path:
        """:returns: the path to the file."""

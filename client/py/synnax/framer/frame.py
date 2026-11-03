#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections import Counter
from collections.abc import Iterator, Mapping, Sequence
from typing import TypeAlias, overload, override

import numpy as np
import numpy.typing as npt
import pandas as pd
from pandas import DataFrame
from pandas.api.extensions import ExtensionDtype
from pydantic import BaseModel, Field
from typing_extensions import TypeVar

import synnax.channel.payload as channel
from synnax.exceptions import MultipleFoundError, ValidationError
from synnax.telem import CrudeSeries, MultiSeries, Series, TypedCrudeSeries


class FramePayload(BaseModel):
    keys: list[channel.Key]
    series: list[Series]

    def __init__(
        self,
        keys: list[int] | None = None,
        series: list[Series] | None = None,
    ):
        if series is None:
            series = []
        if keys is None:
            keys = []
        super().__init__(series=series, keys=keys)


class Frame:
    """
    A frame is a collection of telemetry series mapped to particular channels. Frames
    can be keyed by channel name or channel key, but not both.

    :ivar names: The names of channels in the frame, by channel key. Frames read from
        the Core carry the names of the channels they read.
    :ivar indexes: The key of the index channel of each channel, by channel key. An
        index channel is its own index. Frames read from the Core carry the indexes of
        the channels they read.
    """

    channels: list[channel.Key | str]
    series: list[Series] = Field(default_factory=list)
    names: dict[channel.Key, str]
    indexes: dict[channel.Key, channel.Key]

    @overload
    def __init__(
        self,
        channels: list[channel.Key | str],
        series: Sequence[TypedCrudeSeries] | None = None,
    ) -> None:
        """Builds a frame from a list that mixes channel keys and names."""

    @overload
    def __init__(
        self,
        channels: (
            list[channel.Key]
            | tuple[channel.Key, ...]
            | list[str]
            | tuple[str, ...]
            | DataFrame
            | Frame
            | FramePayload
            | Mapping[channel.Key, TypedCrudeSeries]
            | Mapping[str, TypedCrudeSeries]
            | None
        ) = None,
        series: Sequence[TypedCrudeSeries] | None = None,
    ) -> None: ...

    def __init__(
        self,
        channels: (
            list[channel.Key | str]
            | list[channel.Key]
            | tuple[channel.Key, ...]
            | list[str]
            | tuple[str, ...]
            | DataFrame
            | Frame
            | FramePayload
            | Mapping[channel.Key, TypedCrudeSeries]
            | Mapping[str, TypedCrudeSeries]
            | None
        ) = None,
        series: Sequence[TypedCrudeSeries] | None = None,
    ) -> None:
        self.names = {}
        self.indexes = {}
        if isinstance(channels, Frame):
            self.channels = channels.channels
            self.series = channels.series
            self.names = channels.names
            self.indexes = channels.indexes
        elif isinstance(channels, FramePayload):
            self.channels = list(channels.keys)
            self.series = channels.series
        elif isinstance(channels, DataFrame):
            self.channels = list(channels.columns)
            self.series = [Series(data=channels[k]) for k in self.channels]
        elif isinstance(channels, Mapping):
            self.channels = list(channels.keys())
            self.series = [Series(d) for d in channels.values()]
        elif channels is None or isinstance(channels, (list, tuple)):
            self.series = [] if series is None else [Series(d) for d in series]
            self.channels = list(channels) if channels else []
        else:
            raise ValueError(f"""
                [Frame] - invalid construction arguments. Received {channels}
                and {series}.
            """)

    @override
    def __str__(self) -> str:
        return self.to_df().__str__()

    @overload
    def append(self, col_or_frame: channel.Key | str, series: Series) -> None:
        """Adds a new series to the frame for the given channel.
        :param col_or_frame: The channel key or name to append the series to.
        :param series: The series to append to the frame
        """

    @overload
    def append(self, col_or_frame: Frame) -> None:
        """Appends the given frame to the current frame, modifying the current frame in
        place.
        :param col_or_frame: The frame to append to the current frame.
        """

    def append(
        self,
        col_or_frame: channel.Key | str | Frame,
        series: Series | None = None,
    ) -> None:
        """Appends a frame or series to the current frame. If a frame is provided, the
        series and channels from the frame are appended to the current frame. If a
        series is provided, the series is appended to the frame for the given channel.
        :param col_or_frame: The channel key or name to append the series to, or the
            frame to append to the current frame.
        :param series: The series to append to the frame, if col_or_frame is a channel
            key or name.
        """
        if isinstance(col_or_frame, Frame):
            self.series.extend(col_or_frame.series)
            self.channels.extend(col_or_frame.channels)
            self.names.update(col_or_frame.names)
            self.indexes.update(col_or_frame.indexes)
        else:
            if series is None:
                raise ValueError("series must be provided when appending a channel")
            self.series.append(series)
            self.channels.append(col_or_frame)

    def items(
        self,
    ) -> Iterator[tuple[channel.Key | str, Series]]:
        """
        Returns a generator of tuples containing the channel and series for each channel
        in the frame.
        """
        return zip(self.channels, self.series, strict=True)

    @overload
    def __getitem__(self, key: channel.Key | str) -> MultiSeries:
        """Gets the series of a channel by its key or name. A name also finds a
        channel keyed by key when the frame knows its name.

        :raises MultipleFoundError: If two channels in the frame share the name.
        """

    @overload
    def __getitem__(self, key: npt.NDArray[np.bool_]) -> DataFrame:
        """Selects the rows of to_df() where the boolean mask is true. The mask has one
        value for each row of to_df().
        """

    def __getitem__(
        self, key: channel.Key | str | npt.NDArray[np.bool_]
    ) -> MultiSeries | DataFrame:
        if not isinstance(key, (channel.Key, str)):
            return self.to_df()[key]
        cols = self._columns(key)
        return MultiSeries(
            [self.series[i] for i, k in enumerate(self.channels) if k in cols]
        )

    def _columns(self, key: channel.Key | str) -> set[channel.Key | str]:
        if isinstance(key, channel.Key):
            name = self.names.get(key)
            if name is None or list(self.names.values()).count(name) > 1:
                return {key}
            return {key, name}
        keys = [k for k, name in self.names.items() if name == key]
        if len(keys) > 1:
            raise MultipleFoundError(
                f"Channels {keys} in the frame share the name {key}. Use a key."
            )
        return {key, *keys}

    @overload
    def get(
        self, key: channel.Key | str, default: None = None
    ) -> MultiSeries | None: ...

    @overload
    def get(self, key: channel.Key | str, default: MultiSeries) -> MultiSeries: ...

    def get(
        self, key: channel.Key | str, default: MultiSeries | None = None
    ) -> MultiSeries | None:
        """Gets the series of a channel by its key or name.

        :param key: The channel key or name to get the series for.
        :param default: The value to return when the frame has no series for the
            channel.
        :returns: The series of the channel, or default when the frame has no series
            for the channel.
        """
        if key not in self:
            return default
        return self[key]

    def to_payload(self) -> FramePayload:
        """Converts the frame to its payload representation for transport over the
        network. This method should typically only be used internally.
        :raises: ValidationError if the frame is keyed by channel name instead of key.
        """
        if not all(isinstance(k, channel.Key) for k in self.channels):
            diff = [k for k in self.channels if not isinstance(k, channel.Key)]
            raise ValidationError(f"""
            Cannot convert a frame that contains channel names to a payload.
            The following channels are invalid: {diff}
            """)
        keys: list[int] = [k for k in self.channels if isinstance(k, int)]
        return FramePayload(keys=keys, series=self.series)

    def to_df(self) -> DataFrame:
        """Converts the frame to a pandas DataFrame with one column per channel. The
        columns are in the order the channels first appear in the frame. Each column
        is the channel name, or the channel key when the frame does not know the name
        or when two channels in the frame share the name.

        When the frame holds the index channel of every channel, the DataFrame index is
        a UTC DatetimeIndex and the index channels are not columns. The DatetimeIndex
        has the name of the index channel, or "time" when the frame holds more than
        one. Channels on different index channels are joined on time, and a cell is
        blank when its channel has no sample at that time. Integer columns with blanks
        use pandas nullable dtypes, such as Int64. When the frame does not hold an
        index channel, the DataFrame index is a RangeIndex.
        """
        cols = {
            k: self.names.get(k, k) if isinstance(k, channel.Key) else k
            for k in self.channels
        }
        if len(cols) == 0:
            return DataFrame()
        counts = Counter(cols.values())
        names = {k: name if counts[name] == 1 else k for k, name in cols.items()}
        indexes: dict[channel.Key | str, channel.Key] = {}
        for k in cols:
            idx = self._index(k)
            if idx is None or idx not in self:
                return DataFrame({names[k]: _column_values(self[k]) for k in cols})
            indexes[k] = idx
        index_cols = {idx: k for k, idx in indexes.items() if self._key(k) == idx}
        times = {idx: self[idx].to_numpy() for idx in set(indexes.values())}
        merged = _time_index(np.unique(np.concatenate(list(times.values()))))
        merged.name = (
            names[index_cols[next(iter(times))]] if len(times) == 1 else "time"
        )
        data: dict[channel.Key | str, pd.Series] = {}
        for k in cols:
            if k in index_cols.values():
                continue
            t = _time_index(_sample_times(self[k], self[indexes[k]]))
            values = _column_values(self[k])
            if len(t) == len(merged):
                data[names[k]] = pd.Series(values, index=t)
            else:
                col = pd.Series(values, index=t, dtype=_nullable_dtype(values))
                data[names[k]] = col.reindex(merged)
        return DataFrame(data, index=merged)

    def _key(self, key: channel.Key | str) -> channel.Key | None:
        if isinstance(key, channel.Key):
            return key
        keys = [k for k, name in self.names.items() if name == key]
        return keys[0] if len(keys) == 1 else None

    def _index(self, key: channel.Key | str) -> channel.Key | None:
        k = self._key(key)
        return None if k is None else self.indexes.get(k)

    def __contains__(self, key: channel.Key | str) -> bool:
        return not self._columns(key).isdisjoint(self.channels)


def _column_values(series: MultiSeries) -> npt.NDArray[np.generic] | list[object]:
    if series.data_type.has_np:
        return series.to_numpy()
    return list(series)


def _alignments(series: MultiSeries) -> npt.NDArray[np.uint64]:
    return np.concatenate(
        [
            np.empty(0, dtype=np.uint64),
            *(
                np.uint64(int(s.alignment))
                + np.arange(len(s), dtype=np.uint64) * np.uint64(s.alignment_multiple)
                for s in series.series
            ),
        ]
    )


def _sample_times(series: MultiSeries, index: MultiSeries) -> npt.NDArray[np.int64]:
    """Returns the timestamp of each sample in series. A channel can hold fewer samples
    than its index, so samples match index samples by alignment.
    """
    stamps = index.to_numpy()
    alignments = _alignments(series)
    index_alignments = _alignments(index)
    if np.array_equal(alignments, index_alignments):
        return stamps
    shared, _, positions = np.intersect1d(
        alignments, index_alignments, return_indices=True
    )
    if len(shared) != len(alignments):
        raise ValueError("A channel has samples at alignments its index does not hold.")
    return stamps[positions]


def _time_index(stamps: npt.NDArray[np.int64]) -> pd.DatetimeIndex:
    return pd.to_datetime(stamps, unit="ns", utc=True)


# A blank in a NumPy integer column makes it float64, which corrupts values above 2^53.
_NULLABLE_INTS: Mapping[np.dtype, ExtensionDtype] = {
    np.dtype(np.int8): pd.Int8Dtype(),
    np.dtype(np.int16): pd.Int16Dtype(),
    np.dtype(np.int32): pd.Int32Dtype(),
    np.dtype(np.int64): pd.Int64Dtype(),
    np.dtype(np.uint8): pd.UInt8Dtype(),
    np.dtype(np.uint16): pd.UInt16Dtype(),
    np.dtype(np.uint32): pd.UInt32Dtype(),
    np.dtype(np.uint64): pd.UInt64Dtype(),
}


def _nullable_dtype(
    values: npt.NDArray[np.generic] | list[object],
) -> ExtensionDtype | None:
    if isinstance(values, list):
        return None
    return _NULLABLE_INTS.get(values.dtype)


# Bound per call, so a typed mapping of keys, names, payloads, or a mix matches.
K = TypeVar(
    "K",
    bound=channel.Key | str | channel.Payload,
    default=channel.Key | str | channel.Payload,
)
CrudeFrame: TypeAlias = Frame | FramePayload | Mapping[K, CrudeSeries] | DataFrame

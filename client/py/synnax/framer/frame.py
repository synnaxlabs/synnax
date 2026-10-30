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
from typing import TypeAlias, overload

import numpy as np
import numpy.typing as npt
from pandas import DataFrame
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
    """

    channels: list[channel.Key | str]
    series: list[Series] = Field(default_factory=list)
    names: dict[channel.Key, str]

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
            | tuple[channel.Key]
            | list[str]
            | tuple[str]
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
            | tuple[channel.Key]
            | list[str]
            | tuple[str]
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
        if isinstance(channels, Frame):
            self.channels = channels.channels
            self.series = channels.series
            self.names = channels.names
        elif isinstance(channels, FramePayload):
            self.channels = list(channels.keys)
            self.series = channels.series
        elif isinstance(channels, DataFrame):
            self.channels = list(channels.columns)
            self.series = [Series(data=channels[k]) for k in self.channels]
        elif isinstance(channels, Mapping):
            self.channels = list(channels.keys())
            self.series = [Series(d) for d in channels.values()]
        elif (series is None or isinstance(series, list)) and (
            channels is None or isinstance(channels, list)
        ):
            self.series = [] if series is None else [Series(d) for d in series]
            self.channels = list(channels) if channels else []
        else:
            raise ValueError(f"""
                [Frame] - invalid construction arguments. Received {channels}
                and {series}.
            """)

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
        """Selects the rows where the boolean mask is true, through pandas."""

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
        """
        cols = {
            k: self.names.get(k, k) if isinstance(k, channel.Key) else k
            for k in self.channels
        }
        counts = Counter(cols.values())
        base: dict[channel.Key | str, object] = {}
        for k, name in cols.items():
            col = name if counts[name] == 1 else k
            v = self[k]
            try:
                base[col] = v.__array__()
            except TypeError:
                base[col] = list(v)
        return DataFrame(base)

    def __contains__(self, key: channel.Key | str) -> bool:
        return not self._columns(key).isdisjoint(self.channels)


# Bound per call, so a typed mapping of keys, names, payloads, or a mix matches.
K = TypeVar(
    "K",
    bound=channel.Key | str | channel.Payload,
    default=channel.Key | str | channel.Payload,
)
CrudeFrame: TypeAlias = Frame | FramePayload | Mapping[K, CrudeSeries] | DataFrame

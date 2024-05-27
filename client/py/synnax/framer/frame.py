#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import overload

from freighter import Payload
from pandas import DataFrame
from pydantic import Field

from synnax.channel.payload import (
    ChannelKey,
    ChannelKeys,
    ChannelName,
    ChannelNames,
)
from synnax.exceptions import ValidationError
from synnax.telem import Series, MultiSeries, TypedCrudeSeries


class FramePayload(Payload):
    keys: ChannelKeys
    series: list[Series]

    def __init__(
        self,
        keys: list[int] | None = None,
        series: list[Series] | None = None,
    ):
        # This is a workaround to allow for a None value to be
        # passed to the arrays field, but still have required
        # type hinting.
        if series is None:
            series = list()
        if keys is None:
            keys = list()
        super().__init__(series=series, keys=keys)


class Frame:
    channels: list[ChannelKey | ChannelName]
    series: list[Series] = Field(default_factory=list)

    def __new__(cls, *args, **kwargs):
        return super().__new__(cls)
        # return super().__new__(overload_np_array_operators(cls, "to_df"))

    def __init__(
        self,
        channels: ChannelKeys
        | ChannelNames
        | DataFrame
        | Frame
        | FramePayload
        | dict[ChannelKey, TypedCrudeSeries]
        | None = None,
        series: list[TypedCrudeSeries] | None = None,
    ):
        if isinstance(channels, Frame):
            self.channels = channels.channels
            self.series = channels.series
        elif isinstance(channels, FramePayload):
            self.channels = channels.keys
            self.series = channels.series
        elif isinstance(channels, DataFrame):
            self.channels = channels.columns.to_list()
            self.series = [Series(data=channels[k]) for k in self.channels]
        elif isinstance(channels, dict):
            self.channels = list(channels.keys())
            self.series = [Series(d) for d in channels.values()]
        elif (series is None or isinstance(series, list)) and (
            channels is None or isinstance(channels, list)
        ):
            self.series = list() if series is None else [Series(d) for d in series]
            self.channels = channels or list[ChannelKey]()
        else:
            raise ValueError(
                f"""
                [Frame] - invalid construction arguments. Received {channels}
                and {series}.
            """
            )

    def __str__(self) -> str:
        return self.to_df().__str__()

    @overload
    def append(self, col_or_frame: ChannelKey | ChannelName, array: Series) -> None:
        ...

    @overload
    def append(self, col_or_frame: Frame) -> None:
        ...

    def append(
        self,
        col_or_frame: ChannelKey | ChannelName | Frame,
        series: Series | None = None,
    ) -> None:
        if isinstance(col_or_frame, Frame):
            self.series.extend(col_or_frame.series)  # type: ignore
            self.channels.extend(col_or_frame.channels)  # type: ignore
        else:
            self.series.append(series)
            self.channels.append(col_or_frame)  # type: ignore

    def items(
        self,
    ) -> list[tuple[ChannelKey, Series]] | list[tuple[ChannelName, Series]]:
        return zip(self.channels, self.series)  # type: ignore

    def __getitem__(self, key: ChannelKey | ChannelName | any) -> MultiSeries:
        if not isinstance(key, (ChannelKey, ChannelName)):
            return self.to_df()[key]
        indexes = [i for i, k in enumerate(self.channels) if k == key]
        return MultiSeries([self.series[i] for i in indexes])

    def get(
        self, key: ChannelKey | ChannelName, default: Series | None = None
    ) -> MultiSeries | None:
        try:
            return self[key]
        except ValueError:
            return default

    def to_payload(self):
        if not all(isinstance(k, ChannelKey) for k in self.channels):
            diff = [k for k in self.channels if not isinstance(k, ChannelKey)]
            raise ValidationError(
                f"""
            Cannot convert a frame that contains channel names to a payload.
            The following channels are invalid: {diff}
            """
            )
        return FramePayload(keys=self.channels, series=self.series)

    def to_df(self) -> DataFrame:
        return DataFrame({k: s for k, s in self.items()})

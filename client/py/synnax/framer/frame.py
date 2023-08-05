#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Literal, overload

from pandas import DataFrame

from freighter import Payload
from pydantic import Field

from synnax.telem import Series, TimeRange
from synnax.channel.payload import (
    ChannelKeys,
    ChannelNames,
    ChannelKey,
    ChannelName,
    ChannelParams,
)
from synnax.util.normalize import normalize
from synnax.exceptions import ValidationError


class FramePayload(Payload):
    keys: ChannelKeys
    series: list[Series]

    def __init__(
        self,
        keys: list[str] | None = None,
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


_ColumnTypeT = Literal["keys", "names", None]


class ColumnType:
    v: _ColumnTypeT

    def __init__(self, v: _ColumnTypeT):
        self.v = v

    @classmethod
    def from_channel_params(cls, columns: ChannelParams) -> ColumnType:
        if len(columns) == 0:
            return cls(None)
        first = normalize(columns)[0]
        if isinstance(first, ChannelKey):
            return cls("keys")
        return cls("names")

    def __eq__(self, other: _ColumnTypeT):
        c = ColumnType(other)
        return self.v is None or c.v is None or c.v == self.v


class Frame:
    columns: ChannelKeys | ChannelNames
    series: list[Series] = Field(default_factory=list)

    def __init__(
        self,
        columns_or_data: ChannelKeys
        | ChannelNames
        | DataFrame
        | Frame
        | FramePayload
        | None = None,
        series: list[Series] | None = None,
    ):
        if isinstance(columns_or_data, Frame):
            self.columns = columns_or_data.columns
            self.series = columns_or_data.series
        elif isinstance(columns_or_data, FramePayload):
            self.columns = columns_or_data.keys
            self.series = columns_or_data.series
        elif isinstance(columns_or_data, DataFrame):
            self.columns = columns_or_data.columns.to_list()
            self.series = [Series(data=columns_or_data[k]) for k in self.columns]
        else:
            self.series = series or list()
            self.columns = columns_or_data or list()

    def __str__(self) -> str:
        return self.to_df().__str__()

    def compact(self) -> Frame:
        # compact together arrays that have the same key

        if self.series is None:
            return self

        keys = self.columns
        unique_keys = list(set(keys))

        next_arrays = []

        for key in unique_keys:
            indices = [i for i, x in enumerate(keys) if x == key]
            if len(indices) == 1:
                next_arrays.append(self.series[indices[0]])
                continue

            first = self.series[indices[0]]
            rest = [self.series[i] for i in indices[1:]]
            rest.sort(key=lambda x: x.time_range.from_)
            combined = Series(
                time_range=TimeRange(
                    start=first.time_range.start,
                    end=rest[-1].time_range.end,
                ),
                data=b"".join([x.data for x in rest]),
                data_type=first.data_type,
            )
            next_arrays.append(combined)

        self.series = next_arrays
        self.columns = unique_keys
        return self

    @property
    def col_type(self) -> ColumnType:
        return ColumnType.from_channel_params(self.columns)

    @overload
    def append(self, label: ChannelKey | ChannelName, array: Series) -> None:
        ...

    @overload
    def append(self, frame: Frame) -> None:
        ...

    def append(
        self,
        key_or_frame: ChannelKey | ChannelName | Frame,
        array: Series | None = None,
    ) -> None:
        if isinstance(key_or_frame, Frame):
            if self.col_type != key_or_frame.col_type:
                raise ValidationError(
                    f"""
                    Cannot append frame with different label type
                    {self.col_type} != {key_or_frame.col_type}
                """
                )
            self.series.extend(key_or_frame.series)
            self.columns.extend(key_or_frame.columns)
        else:
            if array is None:
                raise ValidationError("Cannot append key without array")
            if self.col_type != ColumnType.from_channel_params([key_or_frame]):
                raise ValidationError("Cannot append array with different label type")
            self.series.append(array)
            self.columns.append(key_or_frame)

    def items(
        self,
    ) -> list[tuple[ChannelKey, Series]] | list[tuple[ChannelName, Series]]:
        return zip(self.columns, self.series)

    def __getitem__(self, key: ChannelKey | ChannelName) -> Series:
        return self.series[self.columns.index(key)]

    def get(
        self, key: ChannelKey | ChannelName, default: Series | None = None
    ) -> Series | None:
        try:
            return self[key]
        except ValueError:
            return default

    def to_payload(self):
        if self.col_type == "names":
            raise ValidationError(
                "Cannot convert a frame labeled by names to a payload"
            )
        return FramePayload(keys=self.columns, series=self.series)

    def to_df(self) -> DataFrame:
        return DataFrame({k: s for k, s in self.items()})

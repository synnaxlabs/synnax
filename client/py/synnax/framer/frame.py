#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Literal, cast, overload

from freighter import Payload
from pandas import DataFrame
from pydantic import Field

from synnax.channel.payload import (
    ChannelKey,
    ChannelKeys,
    ChannelName,
    ChannelNames,
    ChannelParams,
)
from synnax.exceptions import ValidationError
from synnax.telem import Series, TimeRange, TypedCrudeSeries
from synnax.util.normalize import normalize


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


CrudeColumnType = Literal["keys", "names", None]


class ColumnType:
    v: CrudeColumnType

    def __init__(self, v: CrudeColumnType):
        self.v = v

    @classmethod
    def from_channel_params(cls, columns: ChannelParams) -> ColumnType:
        normal = normalize(columns)
        if len(normal) == 0:
            return cls(None)
        first = normal[0]
        if isinstance(first, ChannelKey):
            return cls("keys")
        return cls("names")

    def __eq__(self, rhs: object):
        if rhs is None or rhs == "keys" or rhs == "names":
            c = ColumnType(cast(CrudeColumnType, rhs))
            return self.v is None or c.v is None or c.v == self.v
        return False


class Frame:
    columns: ChannelKeys | ChannelNames
    series: list[Series] = Field(default_factory=list)

    def __new__(cls, *args, **kwargs):
        return super().__new__(cls)
        # return super().__new__(overload_np_array_operators(cls, "to_df"))

    def __init__(
        self,
        columns_or_data: ChannelKeys
        | ChannelNames
        | DataFrame
        | Frame
        | FramePayload
        | dict[ChannelKey, TypedCrudeSeries]
        | None = None,
        series: list[TypedCrudeSeries] | None = None,
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
        elif isinstance(columns_or_data, dict):
            self.columns = list(columns_or_data.keys())
            self.series = [Series(d) for d in columns_or_data.values()]
        elif (series is None or isinstance(series, list)) and (
            columns_or_data is None or isinstance(columns_or_data, list)
        ):
            self.series = series or list[Series]()
            self.columns = columns_or_data or list[ChannelKey]()
        else:
            raise ValueError(
                f"""
                [Frame] - invalid construction arguments. Received {columns_or_data}
                and {series}.
            """
            )

    def __str__(self) -> str:
        return self.to_df().__str__()

    @property
    def col_type(self) -> ColumnType:
        return ColumnType.from_channel_params(self.columns)

    @overload
    def append(self, col_or_frame: ChannelKey | ChannelName, array: Series) -> None:
        ...

    @overload
    def append(self, col_or_frame: Frame) -> None:
        ...

    def append(
        self,
        col_or_frame: ChannelKey | ChannelName | Frame,
        array: Series | None = None,
    ) -> None:
        if isinstance(col_or_frame, Frame):
            if self.col_type != col_or_frame.col_type:
                raise ValidationError(
                    f"""
                    Cannot append frame with different label type
                    {self.col_type} != {col_or_frame.col_type}
                """
                )
            self.series.extend(col_or_frame.series)  # type: ignore
            self.columns.extend(col_or_frame.columns)  # type: ignore
        else:
            if array is None:
                raise ValidationError("Cannot append key without array")
            if self.col_type != ColumnType.from_channel_params(
                [col_or_frame]
            ):  # type: ignore
                raise ValidationError("Cannot append array with different label type")
            self.series.append(array)
            self.columns.append(col_or_frame)  # type: ignore

    def items(
        self,
    ) -> list[tuple[ChannelKey, Series]] | list[tuple[ChannelName, Series]]:
        return zip(self.columns, self.series)  # type: ignore

    def __getitem__(self, key: ChannelKey | ChannelName | any) -> Series:
        if not isinstance(key, (ChannelKey, ChannelName)):
            return self.to_df()[key]
        return self.series[self.columns.index(key)]  # type: ignore

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

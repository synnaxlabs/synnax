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


LabeledBy = Literal["keys", "names", None]


def labeled_by(labels: ChannelParams) -> LabeledBy:
    if len(labels) == 0:
        return None
    first = normalize(labels)[0]
    if isinstance(first, ChannelKey):
        return "keys"
    return "names"


def labeled_by_equal(first: LabeledBy, second: LabeledBy) -> bool:
    return first is None or second is None or first == second


class Frame:
    labels: ChannelKeys | ChannelNames
    series: list[Series] = Field(default_factory=list)

    def __init__(
        self,
        keys: ChannelKeys
        | ChannelNames
        | DataFrame
        | Frame
        | FramePayload
        | None = None,
        series: list[Series] | None = None,
    ):
        if isinstance(keys, Frame):
            self.labels = keys.labels
            self.series = keys.series
        elif isinstance(keys, FramePayload):
            self.labels = keys.keys
            self.series = keys.series
        elif isinstance(keys, DataFrame):
            self.labels = keys.columns.to_list()
            self.series = [Series(data=keys[k]) for k in self.labels]
        else:
            self.series = series or list()
            self.labels = keys or list()

    def __str__(self) -> str:
        return self.to_df().__str__()

    def compact(self) -> Frame:
        # compact together arrays that have the same key

        if self.series is None:
            return self

        keys = self.labels
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
        self.labels = unique_keys
        return self

    @property
    def labeled_by(self) -> LabeledBy:
        return labeled_by(self.labels)

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
            if not labeled_by_equal(self.labeled_by, key_or_frame.labeled_by):
                raise ValidationError(
                    f"""
                    Cannot append frame with different label type {self.labeled_by} != {key_or_frame.labeled_by}
                """
                )
            self.series.extend(key_or_frame.series)
            self.labels.extend(key_or_frame.labels)
        else:
            if array is None:
                raise ValidationError("Cannot append key without array")
            if not labeled_by_equal(self.labeled_by, labeled_by([key_or_frame])):
                raise ValidationError("Cannot append array with different label type")
            self.series.append(array)
            self.labels.append(key_or_frame)

    def items(
        self,
    ) -> list[tuple[ChannelKey, Series]] | list[tuple[ChannelName, Series]]:
        return zip(self.labels, self.series)

    def __getitem__(self, key: ChannelKey | ChannelName) -> Series:
        return self.series[self.labels.index(key)]

    def get(
        self, key: ChannelKey | ChannelName, default: Series | None = None
    ) -> Series | None:
        try:
            return self[key]
        except ValueError:
            return default

    def to_payload(self):
        if self.labeled_by == "names":
            raise ValidationError(
                "Cannot convert a frame labeled by names to a payload"
            )
        return FramePayload(keys=self.labels, series=self.series)

    def to_df(self) -> DataFrame:
        return DataFrame({k: s for k, s in self.items()})

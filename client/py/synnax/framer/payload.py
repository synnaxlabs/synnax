#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from freighter import Payload
from pandas import DataFrame
from pydantic import Field

from synnax.channel.payload import ChannelPayload
from synnax.telem import BinaryArray, NumpyArray, TimeRange


class FrameHeader(Payload):
    keys: list[str] 
     
    def __init__(self, keys: list[str] | None = None, **kwargs):
        # This is a workaround to allow for a None value to be
        # passed to the keys field, but still have required
        # type hinting. 
        if keys is None:
            keys = list()
        super().__init__(keys=keys, **kwargs)

class BinaryFrame(FrameHeader):
    arrays: list[BinaryArray] = Field(default_factory=list)

    def __init__(self, arrays: list[BinaryArray] | None = None, **kwargs):
        # This is a workaround to allow for a None value to be
        # passed to the arrays field, but still have required
        # type hinting. 
        if arrays is None:
            arrays = list()
        super().__init__(arrays=arrays, **kwargs)

    def compact(self):
        # compact together arrays that have the same key

        if self.arrays is None:
            return

        keys = self.keys
        unique_keys = list(set(keys))

        next_arrays = []

        for key in unique_keys:
            indices = [i for i, x in enumerate(keys) if x == key]
            if len(indices) == 1:
                next_arrays.append(self.arrays[indices[0]])
                continue

            first = self.arrays[indices[0]]
            rest = [self.arrays[i] for i in indices[1:]]
            rest.sort(key=lambda x: x.time_range.start)
            combined = BinaryArray(
                time_range=TimeRange(
                    start=first.time_range.start,
                    end=rest[-1].time_range.end,
                ),
                data=b"".join([x.data for x in rest]),
                data_type=first.data_type,
            )
            next_arrays.append(combined)

        self.arrays = next_arrays
        self.keys = unique_keys


class NumpyFrame(FrameHeader):
    arrays: list[NumpyArray]

    class Config:
        arbitrary_types_allowed = True

    @classmethod
    def from_binary(cls, frame: BinaryFrame) -> NumpyFrame:
        return NumpyFrame(
            keys=frame.keys,
            arrays=[NumpyArray.from_binary(arr) for arr in frame.arrays],
        )

    def to_dataframe(self) -> DataFrame:
        return DataFrame({key: arr.data for key, arr in zip(self.keys, self.arrays)})

    def to_binary(self) -> BinaryFrame:
        return BinaryFrame(
            keys=self.keys, arrays=[arr.to_binary() for arr in self.arrays]
        )

    def __getitem__(self, key: str) -> NumpyArray:
        return self.arrays[self.keys.index(key)]


def pandas_to_frame(channels: list[ChannelPayload], df: DataFrame) -> NumpyFrame:
    keys = []
    arrays = []
    for column in df.columns:
        ch = [ch for ch in channels if ch.name == column or ch.key == column]
        if len(ch) > 0:
            ch = ch[0]
            keys.append(ch.key)
            arrays.append(
                NumpyArray(
                    time_range=TimeRange(0, 0),
                    data=df[column].to_numpy(),
                    data_type=ch.data_type,
                )
            )

    return NumpyFrame(keys=keys, arrays=arrays)

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

    def __init__(
        self,
        arrays: list[BinaryArray] | None = None,
        keys: list[str] | None = None,
    ):
        # This is a workaround to allow for a None value to be
        # passed to the arrays field, but still have required
        # type hinting.
        if arrays is None:
            arrays = list()
        if keys is None:
            keys = list()
        super().__init__(arrays=arrays, keys=keys)

    def compact(self) -> BinaryFrame:
        # compact together arrays that have the same key

        if self.arrays is None:
            return self

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
        return self

    def append_arr(self, key: str, array: BinaryArray) -> None:
        self.keys.append(key)
        self.arrays.append(array)

    def append_frame(self, frame: BinaryFrame) -> None:
        self.keys.extend(frame.keys)
        self.arrays.extend(frame.arrays)


class NumpyFrame(FrameHeader):
    arrays: list[NumpyArray]

    class Config:
        arbitrary_types_allowed = True

    def __init__(
        self, arrays: list[NumpyArray] | None = None, keys: list[str] | None = None
    ):
        # This is a workaround to allow for a None value to be
        # passed to the arrays field, but still have required
        # type hinting.
        if arrays is None:
            arrays = list()
        super().__init__(arrays=arrays, keys=keys)

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

    def append(self, key: str, array: NumpyArray) -> None:
        self.keys.append(key)
        self.arrays.append(array)

    def __getitem__(self, key: str) -> NumpyArray:
        return self.arrays[self.keys.index(key)]

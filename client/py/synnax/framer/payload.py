from __future__ import annotations

from freighter import Payload
from pandas import DataFrame
from synnax.channel.payload import ChannelPayload

from synnax.telem import NumpyArray, BinaryArray, TimeRange


class FrameHeader(Payload):
    keys: list[str] | None


class BinaryFrame(FrameHeader):
    arrays: list[BinaryArray] | None


class NumpyFrame(FrameHeader):
    arrays: list[NumpyArray] | None

    class Config:
        arbitrary_types_allowed = True

    @classmethod
    def from_binary(cls, frame: BinaryFrame) -> NumpyFrame:
        return NumpyFrame(keys=frame.keys, arrays=[NumpyArray.from_binary(arr) for arr in frame.arrays])

    def to_dataframe(self) -> DataFrame:
        return DataFrame({key: arr.data for key, arr in zip(self.keys, self.arrays)})

    def to_binary(self) -> BinaryFrame:
        return BinaryFrame(keys=self.keys, arrays=[arr.to_binary() for arr in self.arrays])

    def __getitem__(self, key: str) -> NumpyArray:
        return self.arrays[self.keys.index(key)]


def pandas_to_frame(channels: list[ChannelPayload], df: DataFrame) -> NumpyFrame:
    return NumpyFrame(
        keys=[ch.key for ch in channels],
        arrays=[NumpyArray(
            time_range=TimeRange(0, 0),
            data=df[ch.key].to_numpy(),
            data_type=ch.data_type,
        ) for ch in channels],
    )

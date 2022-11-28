from __future__ import annotations

from freighter import Payload

from synnax.telem import NumpyArray, BinaryArray


class FrameHeader(Payload):
    keys: list[str]


class BinaryFrame(FrameHeader):
    arrays: list[BinaryArray]


class NumpyFrame(FrameHeader):
    arrays: list[NumpyArray]

    @classmethod
    def from_binary(cls, frame: BinaryFrame) -> NumpyFrame:
        return NumpyFrame(arrays=[NumpyArray.from_binary(arr) for arr in frame.arrays])

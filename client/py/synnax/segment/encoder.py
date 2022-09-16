from synnax.channel import Channel
from synnax.segment import NumpySegment, BinarySegment
import numpy as np


class NumpyEncoderDecoder:
    @staticmethod
    def encode(segment: NumpySegment) -> BinarySegment:
        return BinarySegment(
            channel_key=segment.channel.key,
            start=segment.start,
            data=segment.data.tobytes(),
        )

    @staticmethod
    def decode(channel: Channel, segment: BinarySegment) -> NumpySegment:
        return NumpySegment(
            channel=channel,
            start=segment.start,
            data=np.frombuffer(segment.data, dtype=channel.numpy_type),
        )

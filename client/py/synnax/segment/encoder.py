import numpy as np

from synnax.channel.payload import ChannelPayload

from .payload import SegmentPayload
from .sugared import NumpySegment


class NumpyEncoderDecoder:
    @staticmethod
    def encode(segment: NumpySegment) -> SegmentPayload:
        return SegmentPayload(
            channel_key=segment.channel.key,
            start=segment.start,
            data=segment.data.tobytes(),
        )

    @staticmethod
    def decode(channel: ChannelPayload, segment: SegmentPayload) -> NumpySegment:
        return NumpySegment(
            channel=channel,
            start=segment.start,
            data=np.frombuffer(segment.data, dtype=channel.data_type.numpy_type),
        )

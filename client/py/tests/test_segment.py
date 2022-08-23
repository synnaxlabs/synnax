from dataclasses import asdict

import numpy
import pytest
from freighter.encoder import ENCODER_DECODERS, EncoderDecoder

from delta import telem
from delta.segment import Segment


class TestSegment:
    @pytest.mark.parametrize("ecd", ENCODER_DECODERS)
    def test_encode_decode(self, ecd: EncoderDecoder):
        segment = Segment(
            channel_key="1-1",
            start=telem.TimeStamp(1),
            data=numpy.ndarray([1, 2, 3]),
        )
        encoded = ecd.encode(segment)
        decoded = Segment()
        ecd.decode(encoded, decoded)
        assert asdict(segment) == asdict(decoded)

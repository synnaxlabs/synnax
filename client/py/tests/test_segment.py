from dataclasses import asdict

import numpy
import pytest
from freighter.encoder import EncoderDecoder, MsgpackEncoderDecoder

from delta import telem
from delta.segment import Segment


class TestSegment:
    @pytest.mark.parametrize("ecd", [MsgpackEncoderDecoder])
    def test_encode_decode(self, ecd: EncoderDecoder):
        segment = Segment(
            channel_key="1-1",
            start=telem.TimeStamp(1),
            data=b'12345',
        )
        encoded = ecd.encode(segment)
        decoded = Segment()
        ecd.decode(encoded, decoded)
        assert asdict(segment) == asdict(decoded)

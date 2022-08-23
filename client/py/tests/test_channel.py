import pytest

from delta import telem
from delta.channel import Channel
from dataclasses import asdict
from freighter.encoder import ENCODER_DECODERS, EncoderDecoder


class TestChannel:
    @pytest.mark.parametrize("ecd", ENCODER_DECODERS)
    def test_encode_decode(self, ecd: EncoderDecoder):
        ch = Channel(
            key="1-1",
            name="test",
            node_id=1,
            data_rate=telem.DataRate(25),
            data_type=telem.DataType(8),
        )
        encoded = ecd.encode(ch)
        decoded = Channel()
        ecd.decode(encoded, decoded)
        assert asdict(ch) == asdict(decoded)

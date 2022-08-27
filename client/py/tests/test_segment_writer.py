import pytest
from freighter.ws import WSClient
from freighter.sync import StreamClient
from freighter import (
    MsgpackEncoderDecoder,
    Endpoint
)

import delta.errors
from delta import telem
from delta.channel import Client, Channel
from delta.segment import (
    BinarySegment
)
from delta.segment.writer import (
    Core,
    WriterRequest,
    WriterResponse
)


def new_core(endpoint: Endpoint) -> Core:
    transport = StreamClient[WriterRequest, WriterResponse](
        WSClient[WriterRequest, WriterResponse](
            encoder=MsgpackEncoderDecoder(),
            endpoint=endpoint,
        ))
    return Core(transport=transport)


class TestCore:
    @pytest.fixture(scope="class")
    def core(self, endpoint: Endpoint) -> Core:
        return new_core(endpoint)

    @pytest.fixture(scope="class")
    def channel(self, core: Core, channel_client: Client) -> Channel:
        return channel_client.create(
            Channel(
                name="test",
                node_id=1,
                rate=25 * telem.HZ,
                data_type=telem.FLOAT64,
            ),
            1
        )[0]

    def test_basic_write(self, core: Core, channel: Channel):
        core.open([channel.key])
        core.write([BinarySegment(
            channel_key=channel.key,
            start=telem.now(),
            data=b'12345678'
        )])
        core.close()

    def test_nonexistent_channel_key(self, core: Core) :
        with pytest.raises(delta.errors.QueryError):
            core.open(["1241-241"])

    def test_write_lock_acquired(
            self, endpoint: Endpoint, core: Core, channel: Channel,
    ):
        core2 = new_core(endpoint)
        core_err = None
        core2_err = None
        try:
            core.open([channel.key])
        except Exception as e:
            core_err = e
        try:
            core2.open([channel.key])
        except Exception as e:
            core2_err = e
        if core_err is None:
            core.close()
        if core2_err is None:
            core2.close()
        assert core_err is not None or core2_err is not None
import numpy
import pytest
import numpy as np
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
    WriterResponse, NumpyWriter
)


def new_core(endpoint: Endpoint) -> Core:
    transport = StreamClient[WriterRequest, WriterResponse](
        WSClient[WriterRequest, WriterResponse](
            encoder=MsgpackEncoderDecoder(),
            endpoint=endpoint,
        ),
    )
    return Core(transport=transport)


@pytest.fixture
def core(endpoint: Endpoint) -> Core:
    return new_core(endpoint)


@pytest.fixture
def channel(core: Core, channel_client: Client) -> Channel:
    return channel_client.create(
        Channel(
            name="test",
            node_id=1,
            rate=25 * telem.HZ,
            data_type=telem.FLOAT64,
        ),
        1
    )[0]


class TestCore:
    def test_basic_write(self, core: Core, channel: Channel):
        core.open([channel.key])
        core.write([BinarySegment(
            channel_key=channel.key,
            start=telem.now(),
            data=b'12345678'
        )])
        core.close()

    def test_nonexistent_channel_key(self, core: Core):
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


class TestNumpy:
    @pytest.fixture
    def writer(self, core: Core, channel_client: Client) -> NumpyWriter:
        return NumpyWriter(core=core, channel_client=channel_client)

    def test_basic_write(self, channel: Channel, writer: NumpyWriter):
        writer.open([channel.key])
        try:
            data = np.random.rand(10).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_invalid_data_type(self, channel: Channel, writer: NumpyWriter):
        writer.open([channel.key])
        try:
            data = np.random.rand(10).astype(np.int64)
            with pytest.raises(delta.errors.ValidationError):
                writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_invalid_data_shape(self, channel: Channel, writer: NumpyWriter):
        writer.open([channel.key])
        try:
            data = np.random.rand(10, 10).astype(np.float64)
            with pytest.raises(delta.errors.ValidationError):
                writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_non_contiguous_segments(self, channel: Channel, writer: NumpyWriter):
        writer.open([channel.key])
        try:
            data = np.random.rand(10).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
            with pytest.raises(delta.errors.ContiguityError):
                writer.write(to=channel.key, data=data, start=1)
        finally:
            writer.close()

    def test_multi_segment_write(self, channel: Channel, writer: NumpyWriter):
        writer.open([channel.key])
        n_samples = 1000
        n_writes = 100
        try:
            for i in range(0, n_writes):
                data = np.random.rand(n_samples).astype(np.float64)
                writer.write(
                    to=channel.key,
                    data=data,
                    start=channel.rate.span(n_samples) * i
                )
        finally:
            writer.close()

    def test_segment_split(self, channel: Channel, writer: NumpyWriter):
        span = channel.rate.size_span(telem.Size(9e6), telem.BIT64)
        n_samples = channel.rate.sample_count(span)
        writer.open([channel.key])
        try:
            data = np.random.rand(n_samples).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()



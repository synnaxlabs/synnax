import pytest
from freighter import Endpoint

from delta import telem
from delta.channel import Channel, Client, new_http_client


@pytest.fixture(scope="session")
def channel_client(endpoint: Endpoint) -> Client:
    return new_http_client(endpoint)


class TestClient:

    def test_create(self, channel_client: Client) -> None:
        ch = Channel(
            name="test",
            node_id=1,
            rate=25 * telem.HZ,
            data_type=telem.FLOAT64,
        )
        channels = channel_client.create(ch, 1)
        assert len(channels) == 1
        assert channels[0].name == "test"
        assert channels[0].key != ""


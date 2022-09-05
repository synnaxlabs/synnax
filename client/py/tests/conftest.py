import pytest

import arya
from arya import Channel, telem


@pytest.fixture(scope="session")
def client() -> arya.Client:
    return arya.Client(host="localhost", port=8080)


@pytest.fixture
def channel(client: arya.Client) -> Channel:
    return client.channel.create_n(
        Channel(
            name="test",
            node_id=1,
            rate=25 * telem.HZ,
            data_type=telem.FLOAT64,
        ),
        1,
    )[0]

import pytest

import synnax
from synnax import Channel, telem


@pytest.fixture(scope="session")
def client() -> synnax.Client:
    return synnax.Client(host="localhost", port=8080)


@pytest.fixture
def channel(client: synnax.Client) -> Channel:
    return client.channel.create_n(
        Channel(
            name="test",
            node_id=1,
            rate=25 * telem.HZ,
            data_type=telem.FLOAT64,
        ),
        1,
    )[0]

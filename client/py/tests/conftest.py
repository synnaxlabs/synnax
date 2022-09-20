import pytest

import synnax
from synnax import Channel, telem


@pytest.fixture(scope="session")
def client() -> synnax.Synnax:
    return synnax.Synnax(host="localhost", port=8080)


@pytest.fixture
def channel(client: synnax.Synnax) -> Channel:
    return client.channel.create_n(
        Channel(
            name="test",
            node_id=1,
            rate=25 * telem.HZ,
            data_type=telem.FLOAT64,
        ),
        1,
    )[0]

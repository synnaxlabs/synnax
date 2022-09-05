import pytest
from freighter import Endpoint

from arya import channel
from arya.transport import Transport


@pytest.fixture(scope="session")
def endpoint() -> Endpoint:
    host = "localhost"
    port = 8080
    return Endpoint(host, port)


@pytest.fixture(scope="session")
def transport(endpoint: Endpoint) -> Transport:
    return Transport(endpoint)


@pytest.fixture(scope="session")
def channel_client(transport: Transport) -> channel.Client:
    return channel.Client(transport=transport)

import pytest
from freighter import Endpoint

from delta import channel


@pytest.fixture(scope="session")
def endpoint() -> Endpoint:
    protocol = "http"
    host = "localhost"
    port = 8080
    path_prefix = "/api/v1/"
    return Endpoint(protocol, host, port, path_prefix)


@pytest.fixture(scope="session")
def channel_client(endpoint: Endpoint) -> channel.Client:
    return channel.new_http_client(endpoint)

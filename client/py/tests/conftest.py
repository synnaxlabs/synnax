import pytest
from freighter import Endpoint


@pytest.fixture(scope="session")
def endpoint() -> Endpoint:
    protocol = "http"
    host = "localhost"
    port = 8080
    path_prefix = "/api/v1/"
    return Endpoint(protocol, host, port, path_prefix)

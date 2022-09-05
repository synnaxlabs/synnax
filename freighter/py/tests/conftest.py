import pytest

from freighter import Endpoint


@pytest.fixture
def endpoint() -> Endpoint:
    return Endpoint("localhost", 8080)


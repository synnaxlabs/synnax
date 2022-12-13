import pytest

from freighter import URL


@pytest.fixture
def endpoint() -> URL:
    return URL("localhost", 8080)

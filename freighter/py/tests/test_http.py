import pytest

from freighter import URL, encoder
from freighter.http import GETClient, HTTPClient, POSTClient

from .interface import Message


@pytest.fixture
def client(endpoint: URL) -> HTTPClient:
    http_endpoint = endpoint.child("http")
    return HTTPClient(http_endpoint, encoder.JSON())


@pytest.fixture
def get_client(client: HTTPClient) -> GETClient[Message, Message]:
    return client.client_get(Message, Message)


@pytest.fixture
def post_client(client: HTTPClient) -> POSTClient[Message, Message]:
    return client.client_post(Message, Message)


class TestGETClient:
    def test_echo(self, get_client: GETClient):
        """
        Should echo an incremented ID back to the caller.
        """
        res, err = get_client.send("/echo", Message(id=1, message="hello"))
        assert err is None
        assert res.id == 2
        assert res.message == "hello"


class TestPOSTClient:
    def test_echo(self, post_client: POSTClient):
        """
        Should echo an incremented ID back to the caller.
        """
        res, err = post_client.send("/echo", Message(id=1, message="hello"))
        assert err is None
        assert res.id == 2
        assert res.message == "hello"

import pytest

from freighter import Endpoint, JSONEncoderDecoder
from freighter import http
from .interface import Message, Error, message_factory


@pytest.fixture
def client(endpoint: Endpoint) -> http.Client:
    http_endpoint = endpoint.child("http", protocol="http")
    return http.Client(http_endpoint, JSONEncoderDecoder())


@pytest.fixture
def get_client(client: http.Client) -> http.GETClient[Message, Message]:
    return client.get(Message, message_factory)


@pytest.fixture
def post_client(client: http.Client) -> http.POSTClient[Message, Message]:
    return client.post(Message, message_factory)


class TestGETClient:
    def test_echo(self, get_client: http.GETClient):
        res, err = get_client.send("/echo", Message(1, "hello"))
        assert err is None
        assert res.id == 2
        assert res.message == "hello"


class TestPOSTClient:
    def test_echo(self, post_client: http.POSTClient):
        res, err = post_client.send("/echo", Message(1, "hello"))
        assert err is None
        assert res.id == 2
        assert res.message == "hello"

import pytest

from freighter import URL, encoder
from freighter.http import GETClient, HTTPClientFactory, POSTClient

from .interface import Message


@pytest.fixture
def http_factory(endpoint: URL) -> HTTPClientFactory:
    http_endpoint = endpoint.child("http")
    return HTTPClientFactory(http_endpoint, encoder.JSONEncoder())


class TestGETClient:
    def test_echo(self, http_factory: HTTPClientFactory):
        """Should echo an incremented ID back to the caller.
        """
        res, err = http_factory.get_client().send("/echo",
                                                  Message(id=1, message="hello"),
                                                  Message)
        assert err is None
        assert res.id == 2
        assert res.message == "hello"


class TestPOSTClient:
    def test_echo(self, http_factory: HTTPClientFactory):
        """Should echo an incremented ID back to the caller.
        """
        res, err = http_factory.post_client().send("/echo",
                                                   Message(id=1, message="hello"),
                                                   Message)
        assert err is None
        assert res.id == 2
        assert res.message == "hello"

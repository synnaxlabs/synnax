import pytest

from freighter import Endpoint


@pytest.mark.focus
class TestEndpoint:
    def test_path(self):
        """
        Should generate a correct path.
        """
        endpoint = Endpoint("localhost", 8080, "/api", "http")
        assert endpoint.path("echo") == "http://localhost:8080/api/echo/"

    def test_path_with_trailing_slash(self):
        """
        Should generate a correct path.
        """
        endpoint = Endpoint("localhost", 8080, "/api/", "http")
        assert endpoint.path("echo/") == "http://localhost:8080/api/echo/"

    def test_path_with_trailing_slash_and_path(self):
        """
        Should generate a correct path.
        """
        endpoint = Endpoint("localhost", 8080, "/api/", "http")
        assert endpoint.path("/echo/") == "http://localhost:8080/api/echo/"

    def test_child(self):
        """
        Should generate a correct child endpoint.
        """
        endpoint = Endpoint("localhost", 8080, "/api", "http")
        child = endpoint.child("echo")
        assert child.path("") == "http://localhost:8080/api/echo/"

    def test_child_replace_protocol(self):
        """
        Should generate a correct child endpoint.
        """
        endpoint = Endpoint("localhost", 8080, "/api", "http")
        child = endpoint.child("echo", "https")
        assert endpoint.path("") == "http://localhost:8080/api/"
        assert child.path("") == "https://localhost:8080/api/echo/"

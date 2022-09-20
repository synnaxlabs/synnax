from freighter import URL


class TestURL:
    def test_child(self):
        """
        Should generate a correct path.
        """
        endpoint = URL("localhost", 8080, "/api", "http")
        assert endpoint.child("echo").stringify() == "http://localhost:8080/api/echo/"

    def test_child_with_trailing_slash(self):
        """
        Should generate a correct path.
        """
        endpoint = URL("localhost", 8080, "/api/", "http")
        assert endpoint.child("echo/").stringify() == "http://localhost:8080/api/echo/"

    def test_child_with_trailing_slash_and_path(self):
        """
        Should generate a correct path.
        """
        endpoint = URL("localhost", 8080, "/api/", "http")
        assert endpoint.child("/echo/").stringify() == "http://localhost:8080/api/echo/"

    def test_child_replace_protocol(self):
        """
        Should generate a correct child endpoint.
        """
        endpoint = URL("localhost", 8080, "/api", "http")
        child = endpoint.child("echo").replace(protocol="https")
        assert endpoint.child("").stringify() == "http://localhost:8080/api/"
        assert child.child("").stringify() == "https://localhost:8080/api/echo/"

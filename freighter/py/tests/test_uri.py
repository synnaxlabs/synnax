#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from freighter import URL


class TestURL:
    def test_child(self) -> None:
        """
        Should generate a correct path.
        """
        url = URL(host="localhost", port=8080, path="/api", protocol="http")
        assert url.child("echo").stringify() == "http://localhost:8080/api/echo/"

    def test_child_with_trailing_slash(self) -> None:
        """
        Should generate a correct path.
        """
        url = URL(host="localhost", port=8080, path="/api/", protocol="http")
        assert url.child("echo/").stringify() == "http://localhost:8080/api/echo/"

    def test_child_with_trailing_slash_and_path(self) -> None:
        """
        Should generate a correct path.
        """
        url = URL(host="localhost", port=8080, path="/api/", protocol="http")
        assert url.child("/echo/").stringify() == "http://localhost:8080/api/echo/"

    def test_child_replace_protocol(self) -> None:
        """
        Should generate a correct child endpoint.
        """
        url = URL(host="localhost", port=8080, path="/api", protocol="http")
        child = url.child("echo").replace(protocol="https")
        assert url.child("").stringify() == "http://localhost:8080/api/"
        assert child.child("").stringify() == "https://localhost:8080/api/echo/"

    def test_parse_with_port(self) -> None:
        """Should parse the host, port, path, and protocol."""
        url = URL.parse("https://localhost:9090/api")
        assert url.stringify() == "https://localhost:9090/api/"

    @pytest.mark.parametrize(
        ("raw", "expected"),
        [
            ("http://localhost/api", "http://localhost:80/api/"),
            ("https://localhost", "https://localhost:443/"),
            ("ws://localhost", "ws://localhost:80/"),
            ("wss://localhost", "wss://localhost:443/"),
        ],
    )
    def test_parse_default_port(self, raw: str, expected: str) -> None:
        """Should use the default port of the protocol when the URL has no port."""
        assert URL.parse(raw).stringify() == expected

    def test_parse_unknown_protocol_no_port(self) -> None:
        """Should raise a ValueError when the protocol has no default port."""
        with pytest.raises(ValueError, match="has no port"):
            URL.parse("tcp://localhost")

    def test_replace_port_zero(self) -> None:
        """Should replace the port with 0."""
        url = URL(host="localhost", port=8080, path="/api", protocol="http")
        assert url.replace(port=0).port == 0

    def test_replace_empty_path(self) -> None:
        """Should replace the path with an empty path."""
        url = URL(host="localhost", port=8080, path="/api", protocol="http")
        assert url.replace(path="").stringify() == "http://localhost:8080/"

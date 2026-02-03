#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import URL


class TestURL:
    def test_child(self) -> None:
        """
        Should generate a correct path.
        """
        url = URL("localhost", 8080, "/api", "http")
        assert url.child("echo").stringify() == "http://localhost:8080/api/echo/"

    def test_child_with_trailing_slash(self) -> None:
        """
        Should generate a correct path.
        """
        url = URL("localhost", 8080, "/api/", "http")
        assert url.child("echo/").stringify() == "http://localhost:8080/api/echo/"

    def test_child_with_trailing_slash_and_path(self) -> None:
        """
        Should generate a correct path.
        """
        url = URL("localhost", 8080, "/api/", "http")
        assert url.child("/echo/").stringify() == "http://localhost:8080/api/echo/"

    def test_child_replace_protocol(self) -> None:
        """
        Should generate a correct child endpoint.
        """
        url = URL("localhost", 8080, "/api", "http")
        child = url.child("echo").replace(protocol="https")
        assert url.child("").stringify() == "http://localhost:8080/api/"
        assert child.child("").stringify() == "https://localhost:8080/api/echo/"

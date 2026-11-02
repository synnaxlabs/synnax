#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from functools import reduce
from urllib.parse import urljoin, urlsplit

_DEFAULT_PORTS = {"http": 80, "https": 443, "ws": 80, "wss": 443}


class URL:
    """
    URL is a simple class for building and extending URLs.

    :param host: The host name or IP address of the server.
    :param port: The port number of the server.
    :param path: The path prefix to use for all requests. Defaults to \"\".
    :param protocol: The protocol to use for all requests.
    """

    protocol: str
    host: str
    port: int
    path: str

    def __init__(
        self, *, host: str, port: int, path: str = "", protocol: str = ""
    ) -> None:
        self.protocol = protocol
        self.host = host
        self.port = port
        self.path = _format_path(path)

    @staticmethod
    def parse(url: str) -> URL:
        """Parses a URL string into a URL object. A URL without a port gets the default
        port of its protocol.

        :raises ValueError: If the URL has no host, or has no port and its protocol has
            no default port.
        """
        parsed = urlsplit(url)
        if parsed.hostname is None:
            raise ValueError(f"URL {url!r} has no host")
        port = parsed.port
        if port is None:
            if parsed.scheme not in _DEFAULT_PORTS:
                raise ValueError(f"URL {url!r} has no port")
            port = _DEFAULT_PORTS[parsed.scheme]
        return URL(
            host=parsed.hostname, port=port, path=parsed.path, protocol=parsed.scheme
        )

    def replace(
        self,
        *,
        host: str | None = None,
        port: int | None = None,
        path: str | None = None,
        protocol: str | None = None,
    ) -> URL:
        """Returns a new URL with the specified fields replaced."""
        return URL(
            host=self.host if host is None else host,
            port=self.port if port is None else port,
            path=self.path if path is None else path,
            protocol=self.protocol if protocol is None else protocol,
        )

    def child(self, path: str) -> URL:
        """Creates a new URL with the given path appended to the current path."""
        return URL(
            host=self.host,
            port=self.port,
            path=self._child_prefix(path),
            protocol=self.protocol,
        )

    def stringify(self) -> str:
        """Returns the URL as a string."""
        return f"{self.protocol}://{self.host}:{self.port}/{self.path}"

    def _child_prefix(self, path: str) -> str:
        return reduce(urljoin, [self.path, _format_path(path)])

    def __str__(self) -> str:
        return self.stringify()


def _format_path(path: str) -> str:
    path = path if path.endswith("/") else f"{path}/"
    return path[1:] if path.startswith("/") else path

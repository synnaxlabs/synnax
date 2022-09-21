from __future__ import annotations

from functools import reduce
from urllib.parse import urljoin


class URL:
    """URI is a simple class for building and extending URIs.

    :param host: The host name or IP address of the server.
    :param port: The port number of the server.
    :param path: The path prefix to use for all requests. Defaults to "".
    :param protocol: The protocol to use for all requests.
    """

    protocol: str
    host: str
    port: int
    path: str

    def __init__(
        self, host: str, port: int, path: str = "", protocol: str = ""
    ) -> None:
        self.protocol = protocol
        self.host = host
        self.port = port
        self.path = format_path(path)

    def replace(
        self,
        host: str = None,
        port: int = None,
        path: str = None,
        protocol: str = None,
    ) -> URL:
        """Replace returns a new URI with the specified fields replaced."""
        return URL(
            host or self.host,
            port or self.port,
            path or self.path,
            protocol or self.protocol,
        )

    def child(self, path: str) -> URL:
        return URL(self.host, self.port, self._child_prefix(path), self.protocol)

    def stringify(self) -> str:
        return f"{self.protocol}://{self.host}:{self.port}/{self.path}"

    def _child_prefix(self, path: str):
        return reduce(urljoin, [self.path, format_path(path)])

    def __str__(self) -> str:
        return self.stringify()


def format_path(path: str):
    path = path if path.endswith("/") else f"{path}/"
    path = path[1:] if path.startswith("/") else path
    return path

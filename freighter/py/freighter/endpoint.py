from __future__ import annotations
from urllib.parse import urljoin
from functools import reduce


class Endpoint:
    protocol: str
    host: str
    port: int
    path_prefix: str

    def __init__(
            self, host: str, port: int, path_prefix: str = "", protocol: str = ""
    ) -> None:
        self.protocol = protocol
        self.host = host
        self.port = port
        self.path_prefix = format_path(path_prefix)

    def path(self, path: str) -> str:
        return reduce(urljoin, [self.uri(), self.path_prefix, format_path(path)])

    def child(self, path: str, protocol: str = "") -> Endpoint:
        if protocol == "":
            protocol = self.protocol
        return Endpoint(self.host, self.port, self._child_prefix(path), protocol)

    def uri(self) -> str:
        return f"{self.protocol}://{self.host}:{self.port}"

    def _child_prefix(self, path: str):
        return reduce(urljoin, [self.path_prefix, format_path(path)])

    def __str__(self) -> str:
        return self.uri()


def format_path(path: str):
    path = path if path.endswith("/") else f"{path}/"
    path = path[1:] if path.startswith("/") else path
    return path

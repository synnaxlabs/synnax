from urllib.parse import urljoin
from functools import reduce


class Endpoint:
    protocol: str
    host: str
    port: int
    path_prefix: str

    def __init__(
        self, protocol: str, host: str, port: int, path_prefix: str = ""
    ) -> None:
        self.protocol = protocol
        self.host = host
        self.port = port
        self.path_prefix = format_path(path_prefix)

    def build(self, path: str) -> str:
        return reduce(urljoin, [self.uri(), self.path_prefix, format_path(path)])

    def uri(self) -> str:
        return f"{self.protocol}://{self.host}:{self.port}"

    def __str__(self) -> str:
        return self.uri()


def format_path(path: str):
    path = path if path.endswith("/") else f"{path}/"
    path = path[1:] if path.startswith("/") else path
    return path

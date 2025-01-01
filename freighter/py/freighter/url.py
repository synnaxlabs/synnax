#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from functools import reduce
from urllib.parse import urljoin


class URL:
    """URI is a simple class for building and extending URLs.

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

    @staticmethod
    def parse(url: str) -> URL:
        """Parses a URL string into a URL object."""
        split = url.split("://")
        protocol = split[0]
        split = split[1].split(":")
        host = split[0]
        split = split[1].split("/")
        port = int(split[0])
        path = "/".join(split[1:])
        return URL(host, port, path, protocol)

    def replace(
        self,
        host: str = None,
        port: int = None,
        path: str = None,
        protocol: str = None,
    ) -> URL:
        """Replace returns a new URL with the specified fields replaced."""
        return URL(
            host or self.host,
            port or self.port,
            path or self.path,
            protocol or self.protocol,
        )

    def child(self, path: str) -> URL:
        """Creates a new URL with the given path appended to the current path."""
        return URL(self.host, self.port, self.__child_prefix(path), self.protocol)

    def stringify(self) -> str:
        """Returns the URL as a string."""
        return f"{self.protocol}://{self.host}:{self.port}/{self.path}"

    def __child_prefix(self, path: str):
        return reduce(urljoin, [self.path, format_path(path)])

    def __str__(self) -> str:
        return self.stringify()


def format_path(path: str):
    path = path if path.endswith("/") else f"{path}/"
    path = path[1:] if path.startswith("/") else path
    return path

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import os
from typing import Protocol, TypeAlias, overload

from freighter.codec import Codec
from freighter.transport import RQ, RS, Transport

FilePath: TypeAlias = str | os.PathLike[str]
"""A filesystem path accepted by the streaming-body transport Protocols.

Equivalent to ``str | os.PathLike[str]`` — the same shape ``open()`` accepts for paths.
"""


class FileCodec(Codec, Protocol):
    """A Codec that also has an on-disk file representation.

    A FileClient uses file_extension to negotiate a content type from a file path during
    upload and download. Codecs that only exist on a wire (e.g., a WebSocket framer
    codec) need only satisfy Codec.
    """

    def file_extension(self) -> str:
        """:returns: the file extension (without leading dot) associated with the
        Codec's on-disk format, e.g. "json" or "msgpack".
        """
        ...


class FileClient(Transport, Protocol):
    """Protocol for streaming files to and from a server when a payload could be too
    large to buffer in memory.

    Each method accepts either a file path, which is streamed to or from disk, or an
    in-memory typed payload. The transport infers any wire-format metadata from the file
    path (e.g., from its extension).
    """

    def upload(self, target: str, req: FilePath | RQ, res_t: type[RS]) -> RS:
        """Sends req to target and decodes the response into res_t.

        When req is a file path, its contents are streamed from disk as the request body
        and the wire format is inferred from the path's extension. When req is a typed
        payload, it is encoded and sent inline.

        :param target: the target address of the server.
        :param req: a file path streamed from disk, or a typed request payload.
        :param res_t: the expected response payload type.
        :return: the response returned by the server.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

    @overload
    def download(self, target: str, req: RQ, res_t: type[RS]) -> RS: ...
    @overload
    def download(self, target: str, req: RQ, *, dest: FilePath) -> None: ...
    def download(
        self,
        target: str,
        req: RQ,
        res_t: type[RS] | None = None,
        *,
        dest: FilePath | None = None,
    ) -> RS | None:
        """Sends req to target and returns the response.

        When dest is provided, the response body is streamed straight into that file
        path and the call returns None — the on-disk format is driven by the
        destination's extension. When res_t is provided, the response is decoded into an
        in-memory payload of that type. Exactly one of res_t or dest must be given.

        :param target: the target address of the server.
        :param req: the typed request payload.
        :param res_t: the expected response payload type, for an in-memory response.
        :param dest: file path to stream the response body into.
        :return: the decoded response when res_t is given; otherwise None.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

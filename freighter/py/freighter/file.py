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
from typing import Protocol, TypeAlias

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

    The request or response body is streamed to or from disk, and any wire-format
    metadata is inferred from the file path (e.g., from its extension).
    """

    def upload(self, target: str, req: FilePath, res_t: type[RS]) -> RS:
        """Streams the file at req to target and decodes the response into res_t.

        The contents of req are streamed from disk as the request body and the wire
        format is inferred from the path's extension.

        :param target: the target address of the server.
        :param req: a file path streamed from disk as the request body.
        :param res_t: the expected response payload type.
        :return: the response returned by the server.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

    def download(self, target: str, req: RQ, dest: FilePath) -> None:
        """Sends req to target and streams the response body into dest.

        The response body is streamed straight into dest as it arrives — the on-disk
        format is driven by the destination's extension.

        :param target: the target address of the server.
        :param req: the typed request payload.
        :param dest: file path to stream the response body into.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

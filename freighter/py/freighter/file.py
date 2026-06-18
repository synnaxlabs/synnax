#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Protocol

from freighter.transport import RQ, RS, FilePath, Transport


class FileClient(Transport, Protocol):
    """Protocol for streaming files to and from a server when a payload could be too
    large to buffer in memory. Callers with an already-typed payload use
    UnaryClient.send instead.

    The transport infers any wire-format metadata from the file path (e.g., from its
    extension).
    """

    def upload(self, target: str, req: FilePath, res_t: type[RS]) -> RS:
        """Streams the file at req to target and decodes the response into res_t.

        :param target: the target address of the server.
        :param req: file path whose contents are streamed as the request body.
        :param res_t: the expected response payload type.
        :return: the response returned by the server.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

    def download(self, target: str, req: RQ, dest: FilePath) -> None:
        """Sends req to target and streams the response body into dest.

        :param target: the target address of the server.
        :param req: the typed request payload.
        :param dest: file path to stream the response body into.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

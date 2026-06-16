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

from freighter.transport import RS, FilePath, Transport


class UploadClient(Transport, Protocol):
    """
    Protocol for streaming a file as the request body and decoding a typed response. Use
    when the body could be too large to buffer in memory; callers with an already-typed
    payload use UnaryClient.send.
    """

    def upload(
        self,
        target: str,
        req: FilePath,
        res_t: type[RS],
    ) -> RS:
        """
        Streams req to target and decodes the response into res_t. The transport infers
        any wire-format metadata from the path (e.g., from its extension).

        :param target: the target address of the server.
        :param req: file path whose contents are streamed as the request body.
        :param res_t: the expected response payload type.
        :return: the response returned by the server.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

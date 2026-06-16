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

from freighter.transport import RQ, FilePath, Transport


class DownloadClient(Transport, Protocol):
    """
    Protocol for sending a typed request RQ and streaming the response directly into a
    destination file. Used when the response can be too large to buffer in memory. The
    transport infers any wire-format metadata from the destination (e.g., from its
    extension).
    """

    def download(
        self,
        target: str,
        req: RQ,
        dest: FilePath,
    ) -> None:
        """
        Sends req to target and streams the response body into dest.

        :param target: the target address of the server.
        :param req: the typed request payload.
        :param dest: file path to stream the response body into.
        :raises Unreachable: when the target cannot be reached.
        :raises Exception: any error returned by the server.
        """
        ...

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

from freighter.transport import RQ, RS, AsyncTransport, Transport

from _typeshed import FileDescriptorOrPath


class UnaryClient(Transport, Protocol):
    """
    Protocol for an entity that implements a simple request-response transport between
    two entities.
    """

    def send(
        self,
        target: str,
        req: RQ,
        res_t: type[RS],
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Sends a request to the target server and waits until a response is returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the response type expected from the server. Implementations can
            use this to validate the response.
        :return: any errors encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...


def send_required(client: UnaryClient, target: str, req: RQ, res_t: type[RS]) -> RS:
    """Utility wrapper that throws an exception if the request returns an error."""
    res = client.send(target, req, res_t)
    if res[1] is not None:
        raise res[1]
    return res[0]


class Upload(Transport, Protocol):
    """
    Protocol for streaming a file or open file descriptor as the request body and
    decoding a typed response. Use when the body could be too large to buffer in memory;
    callers with an already-typed payload use UnaryClient.send.
    """

    def upload(
        self,
        target: str,
        req: FileDescriptorOrPath,
        res_t: type[RS],
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Streams req to target and decodes the response into res_t. The transport infers
        any wire-format metadata from req (e.g., from a path extension); file
        descriptors fall back to a transport-default format.

        :param target: the target address of the server.
        :param req: a file path or open file descriptor to stream as the body.
        :param res_t: the expected response payload type.
        :raises Unreachable: when the target cannot be reached.
        """
        ...


class Download(Transport, Protocol):
    """
    Protocol for sending a typed request RQ and streaming the response directly into a
    destination file or file descriptor. Used when the response can be too large to
    buffer in memory. The transport infers any wire-format metadata from the destination
    (e.g., from a path extension).
    """

    def download(
        self,
        target: str,
        req: RQ,
        dest: FileDescriptorOrPath,
    ) -> Exception | None:
        """
        Sends req to target and streams the response body into dest. Returns None on
        success, or the encountered exception.

        :param target: the target address of the server.
        :param req: the typed request payload.
        :param dest: file path or file descriptor to stream the response into.
        :raises Unreachable: when the target cannot be reached.
        """
        ...


class AsyncUnaryClient(AsyncTransport, Protocol):
    """
    Protocol for an entity that implements a simple asynchronous request-response
    transport between two entities.
    """

    async def send(
        self,
        target: str,
        req: RQ,
        res_t: type[RS],
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Sends a request to the target server and waits until a response is returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the response from the server. Implementations can use this to
            validate the response.
        :return: any errors encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...

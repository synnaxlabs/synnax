#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Protocol, Type

from freighter.transport import RQ, RS, AsyncTransport, Transport


class UnaryClient(Transport, Protocol):
    """Protocol for an entity that implements a simple request-response transport
    between two entities.
    """

    def send(
        self,
        target: str,
        req: RQ,
        res_t: Type[RS],
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Sends a request to the target server and waits until a response is
        returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the response type expected from the server. Implementations can use
        this to validate the response.
        :return: any errors encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...


def send_required(client: UnaryClient, target: str, req: RQ, res_t: Type[RS]) -> RS:
    """Utility wrapper that throws an exception if the request returns an error.
    """
    res, exc = client.send(target, req, res_t)
    if exc is not None:
        raise exc
    return res


class AsyncUnaryClient(AsyncTransport, Protocol):
    """Protocol for an entity that implements a simple asynchronous request-response
    transport between two entities.
    """

    async def send(
        self,
        target: str,
        req: RQ,
        res_t: Type[RS],
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Sends a request to the target server and waits until a response is
        returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the response from the server. Implementations can use
        this to validate the response.
        :return: any errors encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...

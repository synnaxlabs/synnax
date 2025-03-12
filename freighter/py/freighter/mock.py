#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Generic, List, Type

from freighter.context import Context
from freighter.transport import RQ, RS, MiddlewareCollector, Payload
from freighter.unary import UnaryClient


class MockUnaryClient(UnaryClient, Generic[RQ, RS]):
    """MockUnaryClient implements a mock unary client with a pre-configured set of
    responses.
    """

    requests: List[RQ]
    responses: List[RS]
    response_errors: List[Exception]
    _mw: MiddlewareCollector

    def __init__(
        self,
        responses: List[RS] | None = None,
        response_errors: List[Exception] | None = None,
    ):
        """
        Initialize a new MockUnaryClient.

        :param responses: List of responses to return
        :param response_errors: List of errors to return
        """
        self.requests: List[RQ] = []
        self.responses = responses or []
        self.response_errors = response_errors or []
        self._mw = MiddlewareCollector()

    def use(self, *middleware) -> None:
        """Implements the Transport protocol."""
        self._mw.use(*middleware)

    def send(
        self,
        target: str,
        req: RQ,
        res_t: Type[RS],
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Mock implementation of send that returns pre-configured responses or errors.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the response type expected from the server
        :return: a tuple of (response, error)
        :raises RuntimeError: when no more responses are available
        """
        self.requests.append(req)
        if not self.responses:
            raise RuntimeError("mock unary client has no responses left!")

        ctx = Context(protocol="mock", target=target, role="client")

        def finalizer(ctx: Context) -> tuple[Context, Exception | None]:
            error = None
            if self.response_errors:
                error = self.response_errors.pop(0)
            return ctx, error

        _, exc = self._mw.exec(ctx, finalizer)
        if exc is not None:
            return None, exc

        return self.responses.pop(0), None

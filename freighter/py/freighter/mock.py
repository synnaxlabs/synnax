#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Sequence

from pydantic import BaseModel

from freighter.context import Context
from freighter.transport import Middleware, MiddlewareCollector, Transport


class MockUnaryClient(Transport):
    """
    MockUnaryClient implements a mock unary client with a pre-configured set of
    responses.
    """

    requests: list[BaseModel]
    responses: list[BaseModel]
    response_errors: list[Exception | None]
    _mw: MiddlewareCollector

    def __init__(
        self,
        *,
        responses: Sequence[BaseModel] | None = None,
        response_errors: list[Exception | None] | None = None,
    ):
        """
        Initialize a new MockUnaryClient.

        :param responses: List of responses to return
        :param response_errors: List of errors to return
        """
        self.requests = []
        self.responses = list(responses or [])
        self.response_errors = response_errors or []
        self._mw = MiddlewareCollector()

    def use(self, *middleware: Middleware) -> None:
        """Implements the Transport protocol."""
        self._mw.use(*middleware)

    def send[RQ: BaseModel, RS: BaseModel](
        self, target: str, req: RQ, res_t: type[RS]
    ) -> RS:
        """
        Mock implementation of send that returns pre-configured responses or raises
        pre-configured errors.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the expected response type.
        :returns: the next pre-configured response.
        :raises RuntimeError: when no more responses are available
        :raises TypeError: when the next response is not an instance of res_t.
        :raises Exception: the next pre-configured error, if any.
        """
        self.requests.append(req)
        if not self.responses:
            raise RuntimeError("mock unary client has no responses left!")

        ctx = Context(protocol="mock", target=target, role="client")

        def finalizer(ctx: Context) -> Context:
            if self.response_errors:
                error = self.response_errors.pop(0)
                if error is not None:
                    raise error
            return ctx

        self._mw.exec(ctx, finalizer=finalizer)

        res = self.responses.pop(0)
        if not isinstance(res, res_t):
            raise TypeError(f"mock response {res!r} is not a {res_t.__name__}")
        return res

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import warnings

from pydantic import BaseModel

from freighter import (
    AsyncMiddleware,
    AsyncNext,
    Context,
    Middleware,
    Next,
    UnaryClient,
)
from synnax.exceptions import ExpiredToken, InvalidToken
from synnax.user.payload import User
from synnax.util.send_required import send_required
from x.deprecation import deprecated_getattr
from x.telem import TimeSpan, TimeStamp
from x.telem.clock_skew import ClockSkewCalculator


class InsecureCredentials(BaseModel):
    username: str
    password: str


class ClusterInfo(BaseModel):
    cluster_key: str = ""
    node_version: str = ""
    node_key: int = 0
    node_time: TimeStamp = TimeStamp(0)


class TokenResponse(BaseModel):
    token: str
    user: User
    cluster_info: ClusterInfo = ClusterInfo()


AUTHORIZATION_HEADER = "Authorization"
TOKEN_REFRESH_HEADER = "Refresh-Token"
RETRY_ON_ERRORS = (InvalidToken, ExpiredToken)
TOKEN_PREFIX = "Bearer "


class Client:
    client: UnaryClient
    username: str
    password: str
    token: str
    user: User
    authenticated: bool

    def __init__(
        self,
        transport: UnaryClient,
        username: str,
        password: str,
        clock_skew_threshold: TimeSpan = TimeSpan.SECOND,
    ) -> None:
        self.client = transport
        self.username = username
        self.password = password
        self.authenticated = False
        self._skew_calc = ClockSkewCalculator()
        self._clock_skew_threshold = clock_skew_threshold

    @property
    def clock_skew(self) -> TimeSpan:
        return self._skew_calc.skew

    def authenticate(self) -> None:
        self._skew_calc.start()
        res = send_required(
            self.client,
            "/auth/login",
            InsecureCredentials(username=self.username, password=self.password),
            TokenResponse,
        )
        self._skew_calc.end(res.cluster_info.node_time)
        if self._skew_calc.exceeds(self._clock_skew_threshold):
            direction = "ahead of" if int(self._skew_calc.skew) > 0 else "behind"
            warnings.warn(
                f"Measured excessive clock skew between this host and the "
                f"Synnax cluster. This host is {direction} the cluster "
                f"by approximately {abs(self._skew_calc.skew)}.",
                UserWarning,
                stacklevel=2,
            )
        self.token = res.token
        self.user = res.user
        self.authenticated = True

    def middleware(self) -> Middleware:
        def mw(ctx: Context, _next: Next) -> tuple[Context, Exception | None]:
            if not self.authenticated:
                self.authenticate()

            ctx.set(AUTHORIZATION_HEADER, TOKEN_PREFIX + self.token)
            out_ctx, exc = _next(ctx)

            if isinstance(exc, RETRY_ON_ERRORS):
                self.authenticated = False
                out_ctx, exc = mw(ctx, _next)

            self.maybe_refresh_token(out_ctx)
            return out_ctx, exc

        return mw

    def async_middleware(self) -> AsyncMiddleware:
        async def mw(
            ctx: Context, _next: AsyncNext
        ) -> tuple[Context, Exception | None]:
            if not self.authenticated:
                self.authenticate()

            ctx.set(AUTHORIZATION_HEADER, TOKEN_PREFIX + self.token)
            out_ctx, exc = await _next(ctx)

            if isinstance(exc, RETRY_ON_ERRORS):
                self.authenticated = False
                out_ctx, exc = await mw(ctx, _next)

            self.maybe_refresh_token(out_ctx)
            return out_ctx, exc

        return mw

    def maybe_refresh_token(
        self,
        ctx: Context,
    ) -> None:
        refresh = ctx.get(TOKEN_REFRESH_HEADER, None)
        if refresh is not None:
            self.token = refresh


_DEPRECATED = {
    "AuthenticationClient": "Client",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

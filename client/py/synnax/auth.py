#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import socket
import warnings

from freighter import (
    AsyncMiddleware,
    AsyncNext,
    Context,
    Middleware,
    Next,
    Payload,
    UnaryClient,
)

from synnax.exceptions import ExpiredToken, InvalidToken
from synnax.telem import TimeSpan, TimeStamp
from synnax.telem.clock_skew import ClockSkewCalculator
from synnax.user.payload import User
from synnax.util.send_required import send_required


class InsecureCredentials(Payload):
    username: str
    password: str


class ClusterInfo(Payload):
    cluster_key: str = ""
    node_version: str = ""
    node_key: int = 0
    node_time: int = 0


class TokenResponse(Payload):
    token: str
    user: User
    cluster_info: ClusterInfo = ClusterInfo()


AUTHORIZATION_HEADER = "Authorization"
TOKEN_REFRESH_HEADER = "Refresh-Token"
RETRY_ON_ERRORS = (InvalidToken, ExpiredToken)
TOKEN_PREFIX = "Bearer "


class AuthenticationClient:
    client: UnaryClient
    username: str
    password: str
    token: str
    user: User
    authenticated: bool
    clock_skew_threshold: TimeSpan

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
        self.clock_skew_threshold = clock_skew_threshold

    def authenticate(self) -> None:
        skew_calc = ClockSkewCalculator()
        skew_calc.start()
        res = send_required(
            self.client,
            "/auth/login",
            InsecureCredentials(username=self.username, password=self.password),
            TokenResponse,
        )
        self.token = res.token
        self.user = res.user
        node_time = res.cluster_info.node_time
        if node_time != 0:
            skew_calc.end(TimeStamp(node_time))
            if skew_calc.exceeds(self.clock_skew_threshold):
                host = socket.gethostname()
                skew = skew_calc.skew()
                direction = "behind" if skew > TimeSpan.ZERO else "ahead of"
                warnings.warn(
                    f"Measured excessive clock skew between this host "
                    f"and the Synnax cluster. This host ({host}) is "
                    f"{direction} the cluster by approximately "
                    f"{abs(skew)}. This may cause problems "
                    f"with time-series data consistency. We highly "
                    f"recommend synchronizing your clock with the Synnax "
                    f"cluster."
                )
        self.authenticated = True

    def middleware(self) -> list[Middleware]:
        def mw(ctx: Context, _next: Next):
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

    def async_middleware(self) -> list[AsyncMiddleware]:
        async def mw(ctx: Context, _next: AsyncNext):
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

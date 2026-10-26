#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


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
from x.telem import TimeStamp


class _Credentials(BaseModel):
    username: str
    password: str


class _ClusterInfo(BaseModel):
    cluster_key: str = ""
    node_version: str = ""
    node_key: int = 0
    node_time: TimeStamp = TimeStamp(0)


class _TokenResponse(BaseModel):
    token: str
    user: User
    cluster_info: _ClusterInfo = _ClusterInfo()


_AUTHORIZATION_HEADER = "Authorization"
_TOKEN_REFRESH_HEADER = "Refresh-Token"
_RETRY_ON_ERRORS = (InvalidToken, ExpiredToken)
_TOKEN_PREFIX = "Bearer "


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
    ) -> None:
        self.client = transport
        self.username = username
        self.password = password
        self.authenticated = False

    def authenticate(self) -> None:
        res = send_required(
            self.client,
            "/auth/login",
            _Credentials(username=self.username, password=self.password),
            _TokenResponse,
        )
        self.token = res.token
        self.user = res.user
        self.authenticated = True

    def middleware(self) -> Middleware:
        def mw(ctx: Context, _next: Next) -> Context:
            if not self.authenticated:
                self.authenticate()

            ctx.set(_AUTHORIZATION_HEADER, _TOKEN_PREFIX + self.token)
            try:
                out_ctx = _next(ctx)
            except _RETRY_ON_ERRORS:
                self.authenticated = False
                out_ctx = mw(ctx, _next)

            self.maybe_refresh_token(out_ctx)
            return out_ctx

        return mw

    def async_middleware(self) -> AsyncMiddleware:
        async def mw(ctx: Context, _next: AsyncNext) -> Context:
            if not self.authenticated:
                self.authenticate()

            ctx.set(_AUTHORIZATION_HEADER, _TOKEN_PREFIX + self.token)
            try:
                out_ctx = await _next(ctx)
            except _RETRY_ON_ERRORS:
                self.authenticated = False
                out_ctx = await mw(ctx, _next)

            self.maybe_refresh_token(out_ctx)
            return out_ctx

        return mw

    def maybe_refresh_token(
        self,
        ctx: Context,
    ) -> None:
        refresh = ctx.get(_TOKEN_REFRESH_HEADER, None)
        if refresh is not None:
            self.token = refresh

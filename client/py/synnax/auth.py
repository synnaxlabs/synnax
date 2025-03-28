#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


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
from synnax.user.payload import User
from synnax.util.send_required import send_required


class InsecureCredentials(Payload):
    username: str
    password: str


class TokenResponse(Payload):
    token: str
    user: User


AUTHORIZATION_HEADER = "Authorization"
TOKEN_REFRESH_HEADER = "Refresh-Token"
RETRY_ON_ERRORS = (InvalidToken, ExpiredToken)
TOKEN_PREFIX = "Bearer "


class AuthenticationClient:
    _LOGIN_ENDPOINT = "/auth/login"

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
            self._LOGIN_ENDPOINT,
            InsecureCredentials(username=self.username, password=self.password),
            TokenResponse,
        )
        self.token = res.token
        self.user = res.user
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

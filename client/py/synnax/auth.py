#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Callable

from freighter import (
    AsyncMiddleware,
    AsyncNext,
    Context,
    Middleware,
    Next,
    Payload,
    UnaryClient,
)

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


def token_middleware(
    token_provider: Callable[[], str], set_token: Callable[[str], None]
) -> Middleware:
    def mw(ctx: Context, _next: Next):
        ctx.set(AUTHORIZATION_HEADER, "Bearer " + token_provider())
        out_ctx, exc = _next(ctx)
        maybe_refresh_token(out_ctx, set_token)
        return out_ctx, exc

    return mw


def async_token_middleware(
    token_provider: Callable[[], str], set_token: Callable[[str], None]
) -> AsyncMiddleware:
    async def mw(ctx: Context, _next: AsyncNext):
        ctx.set(AUTHORIZATION_HEADER, "Bearer " + token_provider())
        out_ctx, exc = await _next(ctx)
        maybe_refresh_token(out_ctx, set_token)
        return out_ctx, exc

    return mw


def maybe_refresh_token(
    ctx: Context,
    set_token: Callable[[str], None],
) -> None:
    refresh = ctx.get(TOKEN_REFRESH_HEADER, None)
    if refresh is not None:
        set_token(refresh)


class AuthenticationClient:
    _LOGIN_ENDPOINT = "/auth/login"

    client: UnaryClient
    username: str
    password: str
    token: str
    user: User

    def __init__(
        self,
        transport: UnaryClient,
        username: str,
        password: str,
    ) -> None:
        self.client = transport
        self.username = username
        self.password = password

    def authenticate(self) -> None:
        res = send_required(
            self.client,
            self._LOGIN_ENDPOINT,
            InsecureCredentials(username=self.username, password=self.password),
            TokenResponse,
        )
        self.token = res.token
        self.user = res.user

    def get_token(self) -> str:
        return self.token

    def set_token(self, token: str) -> None:
        self.token = token

    def middleware(self) -> list[Middleware]:
        return [token_middleware(self.get_token, self.set_token)]

    def async_middleware(self) -> list[AsyncMiddleware]:
        return [async_token_middleware(self.get_token, self.set_token)]

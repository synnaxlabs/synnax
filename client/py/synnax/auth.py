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
    MetaData,
    Middleware,
    Next,
    Payload,
    UnaryClient,
)

from synnax.user.payload import UserPayload


class InsecureCredentials(Payload):
    username: str
    password: str


class TokenResponse(Payload):
    token: str
    user: UserPayload


AUTHORIZATION_HEADER = "Authorization"
TOKEN_REFRESH_HEADER = "Refresh-Token"


def token_middleware(
    token_provider: Callable[[], str], set_token: Callable[[str], None]
) -> Middleware:
    def mw(md: MetaData, _next: Next):
        md.set(AUTHORIZATION_HEADER, "Bearer " + token_provider())
        out_md, exc = _next(md)
        maybe_refresh_token(out_md, set_token)
        return out_md, exc

    return mw


def async_token_middleware(
    token_provider: Callable[[], str], set_token: Callable[[str], None]
) -> AsyncMiddleware:
    async def mw(md: MetaData, _next: AsyncNext):
        md.set(AUTHORIZATION_HEADER, "Bearer " + token_provider())
        out_md, exc = await _next(md)
        maybe_refresh_token(out_md, set_token)
        return out_md, exc

    return mw


def maybe_refresh_token(
    md: MetaData,
    set_token: Callable[[str], None],
) -> None:
    refresh = md.get(TOKEN_REFRESH_HEADER, None)
    if refresh is not None:
        set_token(refresh)


class AuthenticationClient:
    _ENDPOINT = "/auth/login"

    client: UnaryClient
    username: str
    password: str
    token: str
    user: UserPayload

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
        res, exc = self.client.send(
            self._ENDPOINT,
            InsecureCredentials(username=self.username, password=self.password),
            TokenResponse,
        )
        if exc is not None:
            raise exc
        assert res is not None
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

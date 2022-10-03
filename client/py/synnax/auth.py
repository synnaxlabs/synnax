from typing import Callable
from freighter import (
    UnaryClient,
    Payload,
    MetaData,
    Next,
    Middleware,
    AsyncMiddleware,
    AsyncNext,
)

from synnax.user.payload import UserPayload


class InsecureCredentials(Payload):
    username: str
    password: str


class TokenResponse(Payload):
    token: str
    user: UserPayload


AUTHORIZATION_HEADER = "Authorization"


def auth_middleware(token: Callable[[], str]) -> Middleware:
    def mw(md: MetaData, next: Next) -> Middleware:
        md.set(AUTHORIZATION_HEADER, "Bearer " + token())
        return next(md)

    return mw


def async_auth_middleware(token: Callable[[], str]) -> Middleware:
    async def mw(md: MetaData, next: AsyncNext) -> Middleware:
        md.set(AUTHORIZATION_HEADER, "Bearer " + token())
        return await next(md)

    return mw


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
        self.token = res.token
        self.user = res.user

    def get_token(self) -> str:
        return self.token

    def middleware(self) -> Middleware:
        return auth_middleware(self.get_token)

    def async_middleware(self) -> AsyncMiddleware:
        return async_auth_middleware(self.get_token)

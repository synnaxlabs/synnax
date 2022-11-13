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
TOKEN_REFRESH_HEADER = "Refresh-Token"


def token_middleware(token: Callable[[], str],
                     set_token: Callable[[str], None]) -> Middleware:
    def mw(md: MetaData, _next: Next):
        md.set(AUTHORIZATION_HEADER, "Bearer " + token())
        out_md, exc = _next(md)
        if TOKEN_REFRESH_HEADER in out_md.params:
            tk = out_md.get(TOKEN_REFRESH_HEADER)
            set_token(tk)
        return out_md, exc

    return mw


def async_token_middleware(token: Callable[[], str],
                           set_token: Callable[[str], None]) -> Middleware:
    async def mw(md: MetaData, _next: AsyncNext):
        md.set(AUTHORIZATION_HEADER, "Bearer " + token())
        out_md, exc = await _next(md)
        if TOKEN_REFRESH_HEADER in out_md:
            token = out_md.get(TOKEN_REFRESH_HEADER)
            set_token(token)
        return out_md, exc

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

    def set_token(self, token: str) -> None:
        self.token = token

    def middleware(self) -> list[Middleware]:
        return [token_middleware(self.get_token, self.set_token)]

    def async_middleware(self) -> list[AsyncMiddleware]:
        return [async_token_middleware(self.get_token, self.set_token)]

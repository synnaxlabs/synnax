class TokenAuthorizationMiddleware:
    """Middleware that adds a token to the Authorization header.

    :param token: Either the token to use for authorization or a callable that returns the
    token.
    """
    _HEADER_KEY = "Authorization"
    _HEADER_VALUE = "Bearer {token}"

    def __init__(self, token: str | typing.Callable[[], str]):
        self.token = token

    def __call__(self, headers: typing.MutableMapping[str, str]) -> None:
        headers[self._HEADER_KEY] = self._HEADER_VALUE.format(token=self._get_token())

    def _get_token(self) -> str:
        return self.token if type(self.token) == "str" else self.token()

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
from contextvars import ContextVar
from uuid import UUID

from pydantic import BaseModel

from freighter import (
    AsyncMiddleware,
    AsyncNext,
    Context,
    Empty,
    Middleware,
    Next,
    UnaryClient,
)
from synnax import ontology
from synnax.auth import apikey, password, session
from synnax.auth.types_gen import PasswordProof, Proof, Session
from synnax.exceptions import ExpiredToken, InvalidToken, SessionEnded
from synnax.user.payload import User
from synnax.util.send_required import send_required

_API_KEY_PREFIX = "syk_"


def api_key_id(token: str) -> UUID:
    """Returns the key of the API key record that token belongs to.

    :raises InvalidToken: if token is not an API key.
    """
    if not token.startswith(_API_KEY_PREFIX):
        raise InvalidToken("not an API key")
    body = token.removeprefix(_API_KEY_PREFIX)
    try:
        raw = base64.urlsafe_b64decode(body + "=" * (-len(body) % 4))
    except ValueError as e:
        raise InvalidToken("malformed API key") from e
    if len(raw) != 48:
        raise InvalidToken("malformed API key")
    return UUID(bytes=raw[:16])


class _MethodsResponse(BaseModel):
    methods: list[str]


class _SessionCreateRequest(BaseModel):
    proof: Proof


class _SessionCreateResponse(BaseModel):
    token: str
    session: Session
    user: User | None = None


class _SessionRenewResponse(BaseModel):
    session: Session
    user: User | None = None


_METHODS_ENDPOINT = "/auth/methods"
_SESSION_CREATE_ENDPOINT = "/auth/session/create"
_SESSION_RENEW_ENDPOINT = "/auth/session/renew"

_AUTHORIZATION_HEADER = "Authorization"
_RETRY_ON_ERRORS = (InvalidToken, ExpiredToken, SessionEnded)
_TOKEN_PREFIX = "Bearer "


class Client:
    """Logs in with a proof, holds the session, and attaches its token to every
    request.
    """

    client: UnaryClient
    proof: Proof
    token: str
    session: Session | None
    user: User | None
    authenticated: bool

    def __init__(
        self,
        *,
        transport: UnaryClient,
        proof: Proof,
    ) -> None:
        self.client = transport
        self.proof = proof
        self.token = ""
        self.session = None
        self.user = None
        self.authenticated = False
        self.api_keys = apikey.Client(transport)
        self.sessions = session.Client(transport)
        self.passwords = password.Client(transport, self._on_rename)
        # Scoped to the caller's thread or task, so only the logout's own request
        # carries the ended token.
        self._logout_token: ContextVar[str | None] = ContextVar(
            "logout_token", default=None
        )

    def authenticate(self) -> None:
        """Opens a session with the stored proof."""
        res = send_required(
            self.client,
            _SESSION_CREATE_ENDPOINT,
            _SessionCreateRequest(proof=self.proof),
            _SessionCreateResponse,
        )
        self.token = res.token
        self.session = res.session
        self.user = res.user
        self.authenticated = True

    def retrieve_methods(self) -> list[str]:
        """Returns the login methods the Core accepts. Needs no session."""
        return send_required(
            self.client, _METHODS_ENDPOINT, Empty(), _MethodsResponse
        ).methods

    @property
    def subject(self) -> ontology.ID | None:
        """The subject the client logged in as, once logged in."""
        return None if self.session is None else self.session.subject

    def renew(self) -> Session:
        """Reports activity on the session so it does not idle out, and returns the
        session the Core sees.
        """
        res = send_required(
            self.client, _SESSION_RENEW_ENDPOINT, Empty(), _SessionRenewResponse
        )
        self.session = res.session
        self.user = res.user
        return res.session

    def _on_rename(self, subject: ontology.ID | None, username: str) -> None:
        own = subject is None or subject == self.subject
        if own and isinstance(self.proof, PasswordProof):
            self.proof = self.proof.model_copy(update={"username": username})

    def logout(self) -> None:
        """Ends the client's own session and forgets it. A later request logs in
        again. A session the Core already ended counts as logged out.
        """
        if not self.authenticated or self.session is None:
            return
        key, reset = self.session.key, self._logout_token.set(self.token)
        self.authenticated = False
        self.token = ""
        self.session = None
        try:
            self.sessions.delete(key)
        except _RETRY_ON_ERRORS:
            pass
        finally:
            self._logout_token.reset(reset)

    def _forget(self, token: str) -> None:
        # A failure on an older session must not erase one logged in since.
        if self.token == token:
            self.authenticated = False

    def _prepare(self, ctx: Context) -> bool:
        """Attaches the token and reports whether a rejected token may log in again."""
        target = ctx.target.rstrip("/")
        # The Core serves these without a token, so they must not trigger a login.
        if target.endswith((_METHODS_ENDPOINT, _SESSION_CREATE_ENDPOINT)):
            return False
        # A logout ends the session its token belongs to; a dead one must not log in
        # again.
        logout_token = self._logout_token.get()
        if logout_token is not None and target.endswith(session.DELETE_ENDPOINT):
            ctx.set(_AUTHORIZATION_HEADER, _TOKEN_PREFIX + logout_token)
            return False
        if not self.authenticated:
            self.authenticate()
        ctx.set(_AUTHORIZATION_HEADER, _TOKEN_PREFIX + self.token)
        return True

    def middleware(self) -> Middleware:
        def mw(ctx: Context, _next: Next) -> Context:
            retry = self._prepare(ctx)
            sent = self.token
            try:
                return _next(ctx)
            except _RETRY_ON_ERRORS:
                if not retry:
                    raise
                self._forget(sent)
                return mw(ctx, _next)

        return mw

    def async_middleware(self) -> AsyncMiddleware:
        async def mw(ctx: Context, _next: AsyncNext) -> Context:
            retry = self._prepare(ctx)
            sent = self.token
            try:
                return await _next(ctx)
            except _RETRY_ON_ERRORS:
                if not retry:
                    raise
                self._forget(sent)
                return await mw(ctx, _next)

        return mw

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from contextvars import ContextVar

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
from synnax.auth.types_gen import PasswordProof, Proof, Session, SessionKey
from synnax.exceptions import ExpiredToken, InvalidToken, SessionEnded
from synnax.user.payload import User
from synnax.util.send_required import send_required


class _SessionCreateRequest(BaseModel):
    proof: Proof


class _SessionCreateResponse(BaseModel):
    token: str
    session: Session
    user: User | None = None


class _SessionRetrieveRequest(BaseModel):
    subjects: list[ontology.ID] = []


class _SessionRetrieveResponse(BaseModel):
    sessions: list[Session] = []


class _SessionDeleteRequest(BaseModel):
    keys: list[SessionKey] = []


_SESSION_CREATE_ENDPOINT = "/auth/session/create"
_SESSION_RENEW_ENDPOINT = "/auth/session/renew"
_SESSION_RETRIEVE_ENDPOINT = "/auth/session/retrieve"
_SESSION_DELETE_ENDPOINT = "/auth/session/delete"

_AUTHORIZATION_HEADER = "Authorization"
_RETRY_ON_ERRORS = (InvalidToken, ExpiredToken, SessionEnded)
_TOKEN_PREFIX = "Bearer "


class Client:
    """Logs in with a username and password, holds the session, and attaches its token
    to every request.
    """

    client: UnaryClient
    username: str
    password: str
    token: str
    session: Session | None
    user: User | None
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
        self.token = ""
        self.session = None
        self.user = None
        self.authenticated = False
        # Scoped to the caller's thread or task, so only the logout's own request
        # carries the ended token.
        self._logout_token: ContextVar[str | None] = ContextVar(
            "logout_token", default=None
        )

    def authenticate(self) -> None:
        """Opens a session with the stored username and password."""
        res = send_required(
            self.client,
            _SESSION_CREATE_ENDPOINT,
            _SessionCreateRequest(
                proof=PasswordProof(username=self.username, password=self.password)
            ),
            _SessionCreateResponse,
        )
        self.token = res.token
        self.session = res.session
        self.user = res.user
        self.authenticated = True

    @property
    def subject(self) -> ontology.ID | None:
        """The subject the client logged in as, once logged in."""
        return None if self.session is None else self.session.subject

    def renew(self) -> None:
        """Reports activity on the session so it does not idle out."""
        send_required(self.client, _SESSION_RENEW_ENDPOINT, Empty(), Empty)

    def retrieve_sessions(
        self, subjects: list[ontology.ID] | None = None
    ) -> list[Session]:
        """Lists the caller's sessions, or those of the given subjects when the caller
        may retrieve them.
        """
        res = send_required(
            self.client,
            _SESSION_RETRIEVE_ENDPOINT,
            _SessionRetrieveRequest(subjects=subjects or []),
            _SessionRetrieveResponse,
        )
        return res.sessions

    def delete_sessions(self, *keys: SessionKey) -> None:
        """Ends the given sessions. Unknown keys are ignored."""
        send_required(
            self.client,
            _SESSION_DELETE_ENDPOINT,
            _SessionDeleteRequest(keys=list(keys)),
            Empty,
        )

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
            self.delete_sessions(key)
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
        # The login request itself carries no token, and must not trigger a login.
        if target.endswith(_SESSION_CREATE_ENDPOINT):
            return False
        # A logout ends the session its token belongs to; a dead one must not log in
        # again.
        logout_token = self._logout_token.get()
        if logout_token is not None and target.endswith(_SESSION_DELETE_ENDPOINT):
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

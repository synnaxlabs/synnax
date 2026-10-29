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
from synnax.auth.types_gen import Proof, Session, SessionKey
from synnax.exceptions import ExpiredToken, InvalidToken, SessionEnded
from synnax.user.payload import User
from synnax.util.send_required import send_required
from x.telem import TimeStamp

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


class _SessionCreateRequest(BaseModel):
    proof: Proof


class _SessionCreateResponse(BaseModel):
    token: str
    session: Session
    user: User | None = None


class _SessionRenewResponse(BaseModel):
    session: Session
    user: User | None = None


class _SessionRetrieveRequest(BaseModel):
    subjects: list[ontology.ID] = []


class _SessionRetrieveResponse(BaseModel):
    sessions: list[Session] = []


class _SessionDeleteRequest(BaseModel):
    keys: list[SessionKey] = []


class APIKey(BaseModel):
    """An API key as the API returns it: the stored record without its hash."""

    key: UUID
    """The unique identifier for this API key."""
    subject: ontology.ID
    """The user or rack the key logs in as."""
    name: str
    """A label that tells the subject's keys apart."""
    created: TimeStamp
    """When the key was created."""


class _KeyCreateRequest(BaseModel):
    subject: ontology.ID | None = None
    name: str


class _KeyCreateResponse(BaseModel):
    token: str
    key: APIKey


class _KeyRetrieveRequest(BaseModel):
    subjects: list[ontology.ID] = []


class _KeyRetrieveResponse(BaseModel):
    keys: list[APIKey] = []


class _KeyDeleteRequest(BaseModel):
    keys: list[UUID] = []


_SESSION_CREATE_ENDPOINT = "/auth/session/create"
_SESSION_RENEW_ENDPOINT = "/auth/session/renew"
_SESSION_RETRIEVE_ENDPOINT = "/auth/session/retrieve"
_SESSION_DELETE_ENDPOINT = "/auth/session/delete"
_KEY_CREATE_ENDPOINT = "/auth/key/create"
_KEY_RETRIEVE_ENDPOINT = "/auth/key/retrieve"
_KEY_DELETE_ENDPOINT = "/auth/key/delete"

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

    def create_key(
        self, name: str, subject: ontology.ID | None = None
    ) -> tuple[str, APIKey]:
        """Creates an API key for the subject, or for the caller when none is given.

        :returns: the token to log in with, the only copy of its secret, and the key.
        :raises AccessDenied: if the caller may not create a key for the subject.
        :raises NotFoundError: if the subject does not exist.
        """
        res = send_required(
            self.client,
            _KEY_CREATE_ENDPOINT,
            _KeyCreateRequest(subject=subject, name=name),
            _KeyCreateResponse,
        )
        return res.token, res.key

    def retrieve_keys(self, subjects: list[ontology.ID] | None = None) -> list[APIKey]:
        """Lists the caller's API keys, or those of the given subjects when the caller
        may retrieve them.
        """
        res = send_required(
            self.client,
            _KEY_RETRIEVE_ENDPOINT,
            _KeyRetrieveRequest(subjects=subjects or []),
            _KeyRetrieveResponse,
        )
        return res.keys

    def delete_keys(self, *keys: UUID) -> None:
        """Deletes API keys and ends their sessions. Unknown keys are ignored."""
        send_required(
            self.client, _KEY_DELETE_ENDPOINT, _KeyDeleteRequest(keys=list(keys)), Empty
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

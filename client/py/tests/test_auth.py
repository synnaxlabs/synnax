#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import freighter
import synnax as sy
from freighter import Context, Empty, Next
from freighter.mock import MockUnaryClient
from synnax.auth.client import _SessionCreateResponse
from synnax.auth.password import Password
from synnax.exceptions import create_registry
from tests.proxy import Proxy
from tests.user import create_user


@pytest.mark.auth
class TestClusterAuth:
    def test_invalid_credentials(self, login_info: tuple[str, int, str, str]) -> None:
        """
        Should raise an AuthError
        """
        host, port, username, _ = login_info
        with pytest.raises(sy.AuthError):
            sy.Synnax(
                host=host,
                port=port,
                username=username,
                password="wrong",
            )

    def test_no_password_provided(self, login_info: tuple[str, int, str, str]) -> None:
        """
        Should raise a ValidationError
        """
        host, port, username, _ = login_info
        with pytest.raises(sy.PathError):
            sy.Synnax(
                host=host,
                port=port,
                username=username,
            )


@pytest.mark.auth
class TestAuthRetry:
    @pytest.fixture
    def auth_setup(self) -> MockUnaryClient:
        """Fixture that provides the mock clients and auth setup for retry tests."""
        # Setup mock login client with two successful login responses
        user_key = uuid4()
        res = _SessionCreateResponse(
            token="abc",
            session=sy.auth.Session(
                key=uuid4(),
                subject=sy.ontology.ID(key=str(user_key), type="user"),
                credential=uuid4(),
                last_active=sy.TimeStamp(0),
            ),
            user=sy.User(key=user_key, name="Synnax Labs"),
        )
        mock_login_client = MockUnaryClient(
            responses=[res, res], response_errors=[None, None]
        )

        # Create auth client
        auth = sy.auth.Client(
            transport=mock_login_client,
            proof=sy.auth.PasswordProof(username="synnax", password="seldon"),
        )

        # Create base mock client
        mock_client = MockUnaryClient(responses=[Empty(), Empty()])
        mock_client.use(auth.middleware())

        return mock_client

    @pytest.mark.parametrize(
        ("type_", "exc"),
        [
            ("sy.auth", sy.AuthError),
            ("sy.auth.invalid-credentials", sy.exceptions._InvalidCredentials),
            ("sy.auth.invalid_token", sy.InvalidToken),
            ("sy.auth.expired_token", sy.ExpiredToken),
            ("sy.auth.access_denied", sy.AccessDenied),
            ("sy.auth.session_ended", sy.SessionEnded),
            ("sy.auth.repeated-username", sy.AuthError),
        ],
    )
    def test_decode(self, type_: str, exc: type[sy.AuthError]) -> None:
        """Should decode each auth error type the Core sends."""
        err = create_registry().decode(
            freighter.ExceptionPayload(type=type_, data="refused")
        )
        assert type(err) is exc
        assert str(err) == "refused"

    @pytest.mark.parametrize(
        "type_",
        ["sy.auth.invalid_token", "sy.auth.expired_token", "sy.auth.session_ended"],
    )
    def test_retry_on_decoded_token_error(
        self, auth_setup: MockUnaryClient, type_: str
    ) -> None:
        """Should re-authenticate and retry after a token error from the Core."""
        mock_client = auth_setup
        mock_client.response_errors = [
            create_registry().decode(
                freighter.ExceptionPayload(type=type_, data="refused")
            ),
            None,
        ]
        assert mock_client.send("", Empty(), Empty) == Empty()

    def test_stale_failure_keeps_newer_session(self) -> None:
        """Should keep a session logged in while a request on an older one fails."""
        session = sy.auth.Session(
            key=uuid4(),
            subject=sy.ontology.ID(key=str(uuid4()), type="user"),
            credential=uuid4(),
            last_active=sy.TimeStamp(0),
        )
        login = MockUnaryClient(
            responses=[
                _SessionCreateResponse(token=t, session=session) for t in ("abc", "def")
            ]
        )
        auth = sy.auth.Client(
            transport=login,
            proof=sy.auth.PasswordProof(username="synnax", password="seldon"),
        )
        mock_client = MockUnaryClient(responses=[Empty()])
        failed = False

        def relogin_then_end(ctx: Context, _next: Next) -> Context:
            nonlocal failed
            if failed:
                return _next(ctx)
            failed = True
            auth.authenticate()
            raise sy.SessionEnded("ended")

        mock_client.use(auth.middleware())
        mock_client.use(relogin_then_end)
        assert mock_client.send("", Empty(), Empty) == Empty()
        assert auth.token == "def"
        assert len(login.requests) == 2


@pytest.mark.auth
class TestMethods:
    def test_retrieve_methods(self, login_info: tuple[str, int, str, str]) -> None:
        """Should list the login methods the Core accepts."""
        host, port, username, password = login_info
        client = sy.Synnax(host=host, port=port, username=username, password=password)
        assert sorted(client.auth.retrieve_methods()) == ["api_key", "password"]
        client.close()

    def test_retrieve_methods_logged_out(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should list the methods after logout without logging in again."""
        host, port, username, password = login_info
        client = sy.Synnax(host=host, port=port, username=username, password=password)
        client.auth.logout()
        assert sorted(client.auth.retrieve_methods()) == ["api_key", "password"]
        assert not client.auth.authenticated
        client.close()


@pytest.mark.auth
class TestSessions:
    def test_session_lifecycle(self, login_info: tuple[str, int, str, str]) -> None:
        """Should open a session on login, renew it, list it, and end it on logout."""
        host, port, username, password = login_info
        client = sy.Synnax(host=host, port=port, username=username, password=password)
        session = client.auth.session
        assert session is not None
        assert client.auth.subject is not None
        assert client.auth.subject.type == "user"
        assert client.auth.renew().key == session.key
        keys = [s.key for s in client.auth.sessions.retrieve()]
        assert session.key in keys
        client.auth.logout()
        assert not client.auth.authenticated
        # The next request logs in again with a fresh session.
        keys = [s.key for s in client.auth.sessions.retrieve()]
        assert session.key not in keys
        assert client.auth.session is not None
        assert client.auth.session.key in keys
        client.close()

    def test_logout_after_session_ended(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should not open a new session when logging out of a session the Core
        already ended.
        """
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        user_name = str(uuid4())
        create_user(admin, user_name, "seldon")
        ending = sy.Synnax(host=host, port=port, username=user_name, password="seldon")
        other = sy.Synnax(host=host, port=port, username=user_name, password="seldon")
        assert ending.auth.session is not None
        assert other.auth.session is not None
        other.auth.sessions.delete(ending.auth.session.key)
        ending.auth.logout()
        assert not ending.auth.authenticated
        keys = [s.key for s in other.auth.sessions.retrieve()]
        assert keys == [other.auth.session.key]
        ending.close()
        other.close()
        admin.close()

    def test_close_leaves_no_session(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should end the client's only session on close, so none stays open behind
        the connectivity thread.
        """
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        user_name = str(uuid4())
        create_user(admin, user_name, "seldon")
        closing = sy.Synnax(host=host, port=port, username=user_name, password="seldon")
        other = sy.Synnax(host=host, port=port, username=user_name, password="seldon")
        assert other.auth.session is not None
        assert len(other.auth.sessions.retrieve()) == 2
        closing.close()
        assert not closing.connectivity._thread.is_alive()
        keys = [s.key for s in other.auth.sessions.retrieve()]
        assert keys == [other.auth.session.key]
        other.close()
        admin.close()

    def test_close_stops_connectivity_when_unreachable(
        self, proxied_client: sy.Synnax, proxy: Proxy
    ) -> None:
        """Should close without error and stop the connectivity thread when logout
        cannot reach the Core.
        """
        proxy.close()
        proxied_client.close()
        assert not proxied_client.connectivity._thread.is_alive()


@pytest.mark.auth
class TestAPIKeys:
    def test_key_and_password(self, login_info: tuple[str, int, str, str]) -> None:
        """Should reject an API key given with a username and password."""
        host, port, username, password = login_info
        with pytest.raises(sy.ValidationError):
            sy.Synnax(
                host=host,
                port=port,
                username=username,
                password=password,
                api_key="syk_key",
            )

    def test_key_without_host(self) -> None:
        """Should reject an API key given without a host."""
        with pytest.raises(sy.ValidationError):
            sy.Synnax(api_key="syk_key")

    def test_lifecycle(self, login_info: tuple[str, int, str, str]) -> None:
        """Should create a key, log in with it, list it, and delete it."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        new_username = str(uuid4())
        create_user(admin, new_username, "seldon")
        owner = sy.Synnax(
            host=host, port=port, username=new_username, password="seldon"
        )
        token, key = owner.auth.api_keys.create("script")
        assert token.startswith("syk_")
        assert key.name == "script"
        assert key.subject == owner.auth.subject
        keyed = sy.Synnax(host=host, port=port, api_key=token)
        assert keyed.auth.subject == owner.auth.subject
        assert keyed.auth.session is not None
        assert keyed.auth.session.credential == key.key
        assert [k.key for k in owner.auth.api_keys.retrieve()] == [key.key]
        owner.auth.api_keys.delete(key.key)
        assert owner.auth.api_keys.retrieve() == []
        keyed.close()
        owner.close()
        admin.close()

    def test_deleted_key(self, login_info: tuple[str, int, str, str]) -> None:
        """Should fail the next request of a client whose key was deleted."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        token, key = admin.auth.api_keys.create("deleted")
        keyed = sy.Synnax(host=host, port=port, api_key=token)
        admin.auth.api_keys.delete(key.key)
        with pytest.raises(sy.AuthError):
            keyed.auth.sessions.retrieve()
        keyed.close()
        admin.close()

    def test_wrong_key(self, login_info: tuple[str, int, str, str]) -> None:
        """Should reject a key the Core never issued."""
        host, port, _, _ = login_info
        with pytest.raises(sy.AuthError):
            sy.Synnax(host=host, port=port, api_key=f"syk_{uuid4()}secret")


@pytest.mark.auth
class TestPasswords:
    def test_create_existing(self, login_info: tuple[str, int, str, str]) -> None:
        """Should refuse a first password for a user that already has one."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        with pytest.raises(sy.ValidationError):
            admin.auth.passwords.create(str(uuid4()), "p")
        admin.close()

    def test_create_for_new_user(self, login_info: tuple[str, int, str, str]) -> None:
        """Should let an owner give a new user its first password, and let a client
        log in with it.
        """
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        user = admin.users.create(name="First Password")
        new_username = str(uuid4())
        admin.auth.passwords.create(new_username, "first", subject=user.ontology_id)
        client = sy.Synnax(
            host=host, port=port, username=new_username, password="first"
        )
        assert client.auth.subject == user.ontology_id
        client.close()
        admin.close()

    def test_create_repeated_username(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should refuse a username another user holds."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        user = admin.users.create(name="Repeated Username")
        with pytest.raises(sy.AuthError):
            admin.auth.passwords.create(username, "p", subject=user.ontology_id)
        admin.close()

    def test_retrieve_own(self, login_info: tuple[str, int, str, str]) -> None:
        """Should retrieve the caller's own password by default."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        new_username = str(uuid4())
        user = create_user(admin, new_username, "test")
        client = sy.Synnax(host=host, port=port, username=new_username, password="test")
        assert client.auth.passwords.retrieve() == [
            Password(subject=user.ontology_id, username=new_username)
        ]
        client.close()
        admin.close()

    def test_retrieve_by_username(self, login_info: tuple[str, int, str, str]) -> None:
        """Should retrieve a password by its username and ignore unknown ones."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        new_username = str(uuid4())
        user = create_user(admin, new_username, "test")
        res = admin.auth.passwords.retrieve(usernames=[new_username, str(uuid4())])
        assert res == [Password(subject=user.ontology_id, username=new_username)]
        admin.close()

    def test_retrieve_by_subject(self, login_info: tuple[str, int, str, str]) -> None:
        """Should retrieve a password by its subject."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        new_username = str(uuid4())
        user = create_user(admin, new_username, "test")
        res = admin.auth.passwords.retrieve(subjects=[user.ontology_id])
        assert res == [Password(subject=user.ontology_id, username=new_username)]
        admin.close()

    def test_rename(self, login_info: tuple[str, int, str, str]) -> None:
        """Should let an owner rename another user's password, and let a client log
        in with the new username.
        """
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        user = create_user(admin, str(uuid4()), "test")
        new_username = str(uuid4())
        admin.auth.passwords.rename(new_username, subject=user.ontology_id)
        client = sy.Synnax(host=host, port=port, username=new_username, password="test")
        assert client.auth.subject == user.ontology_id
        client.close()
        admin.close()

    def test_rename_own(self, login_info: tuple[str, int, str, str]) -> None:
        """Should let a user rename its own password and keep its session."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        old_username = str(uuid4())
        user = create_user(admin, old_username, "test")
        client = sy.Synnax(host=host, port=port, username=old_username, password="test")
        new_username = str(uuid4())
        client.auth.passwords.rename(new_username)
        assert client.auth.passwords.retrieve() == [
            Password(subject=user.ontology_id, username=new_username)
        ]
        client.close()
        admin.close()

    def test_rename_repeated_username(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should refuse a username another user holds."""
        host, port, username, password = login_info
        admin = sy.Synnax(host=host, port=port, username=username, password=password)
        user = create_user(admin, str(uuid4()), "test")
        with pytest.raises(sy.AuthError):
            admin.auth.passwords.rename(username, subject=user.ontology_id)
        admin.close()

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path
from uuid import UUID, uuid4

import pytest
from keyring.backend import KeyringBackend
from keyring.errors import KeyringError, PasswordDeleteError

import synnax as sy
from synnax.cli import connect, console
from synnax.cli.flow import Context
from synnax.cli.login import run as login
from synnax.config import Cluster, Clusters, File
from synnax.options import Options


class MemoryKeyring(KeyringBackend):
    priority = 1

    def __init__(self) -> None:
        super().__init__()  # type: ignore[no-untyped-call]
        self.values: dict[tuple[str, str], str] = {}

    def get_password(self, service: str, username: str) -> str | None:
        return self.values.get((service, username))

    def set_password(self, service: str, username: str, password: str) -> None:
        self.values[(service, username)] = password

    def delete_password(self, service: str, username: str) -> None:
        if self.values.pop((service, username), None) is None:
            raise PasswordDeleteError(username)


class UndeletableKeyring(MemoryKeyring):
    def delete_password(self, service: str, username: str) -> None:
        raise KeyringError("locked")


@pytest.fixture
def keyring() -> MemoryKeyring:
    return MemoryKeyring()


@pytest.fixture
def clusters(tmp_path: Path, keyring: MemoryKeyring) -> Clusters:
    return Clusters(File(config_dir=tmp_path), keyring)


def save_legacy_password(
    clusters: Clusters, keyring: MemoryKeyring, opts: Options
) -> None:
    clusters.internal.set(
        "clusters.default",
        {"options": opts.model_dump(exclude={"password", "api_key"})},
    )
    keyring.set_password("synnax", "default", opts.password)


def block_config_writes(clusters: Clusters) -> None:
    path = clusters.internal.config_file
    path.unlink()
    path.mkdir()


@pytest.mark.cli
class TestClustersConfig:
    def test_set_saves_api_key(
        self, clusters: Clusters, keyring: MemoryKeyring
    ) -> None:
        """Should keep the key in the keyring and out of the config file."""
        clusters.set(Cluster(options=Options(host="h", port=1, api_key="syk_secret")))
        assert keyring.values == {("synnax.api_key", "default@h:1"): "syk_secret"}
        assert "api_key" not in clusters.internal.get("clusters.default.options")
        got = clusters.get()
        assert got is not None
        assert got.options == Options(host="h", port=1, api_key="syk_secret")

    def test_set_deletes_password(
        self, clusters: Clusters, keyring: MemoryKeyring
    ) -> None:
        """Should delete a password an older CLI saved."""
        save_legacy_password(
            clusters,
            keyring,
            Options(host="h", port=1, username="u", password="p"),
        )
        clusters.set(Cluster(options=Options(host="h", port=1, api_key="k")))
        assert ("synnax", "default") not in keyring.values
        assert "username" not in clusters.internal.get("clusters.default.options")

    def test_get_after_interrupted_migration(
        self, clusters: Clusters, keyring: MemoryKeyring
    ) -> None:
        """Should connect with a saved key even when the config file still holds the
        old username and the keyring the old password.
        """
        save_legacy_password(
            clusters,
            keyring,
            Options(host="h", port=1, username="u", password="p"),
        )
        keyring.set_password("synnax.api_key", "default@h:1", "syk_secret")
        got = clusters.get()
        assert got is not None
        assert got.options == Options(host="h", port=1, api_key="syk_secret")

    def test_set_new_address(self, clusters: Clusters, keyring: MemoryKeyring) -> None:
        """Should delete the key of the old address once the new one is saved."""
        clusters.set(Cluster(options=Options(host="a", port=1, api_key="ka")))
        clusters.set(Cluster(options=Options(host="b", port=1, api_key="kb")))
        assert keyring.values == {("synnax.api_key", "default@b:1"): "kb"}

    def test_set_failed_write_keeps_old_pair(
        self, clusters: Clusters, keyring: MemoryKeyring
    ) -> None:
        """Should keep the old address with its own key when the config write fails."""
        clusters.set(Cluster(options=Options(host="a", port=1, api_key="ka")))
        block_config_writes(clusters)
        with pytest.raises(IsADirectoryError):
            clusters.set(Cluster(options=Options(host="b", port=1, api_key="kb")))
        assert keyring.values == {("synnax.api_key", "default@a:1"): "ka"}
        got = clusters.get()
        assert got is not None
        assert got.options == Options(host="a", port=1, api_key="ka")

    def test_set_failed_write_keeps_password(
        self, clusters: Clusters, keyring: MemoryKeyring
    ) -> None:
        """Should keep a password an older CLI saved when the config write fails."""
        opts = Options(host="h", port=1, username="u", password="p")
        save_legacy_password(clusters, keyring, opts)
        block_config_writes(clusters)
        with pytest.raises(IsADirectoryError):
            clusters.set(Cluster(options=Options(host="h", port=1, api_key="k")))
        assert keyring.values == {("synnax", "default"): "p"}
        got = clusters.get()
        assert got is not None
        assert got.options == opts

    def test_get_keeps_ca_cert(self, clusters: Clusters) -> None:
        """Should keep the CA certificate saved with an API key."""
        opts = Options(host="h", port=1, secure=True, ca_cert="/ca.pem", api_key="k")
        clusters.set(Cluster(options=opts))
        got = clusters.get()
        assert got is not None
        assert got.options == opts

    def test_get_legacy_password(
        self, clusters: Clusters, keyring: MemoryKeyring
    ) -> None:
        """Should read a password an older CLI saved."""
        opts = Options(host="h", port=1, username="u", password="p")
        save_legacy_password(clusters, keyring, opts)
        got = clusters.get()
        assert got is not None
        assert got.options == opts


def delete_new_keys(client: sy.Synnax, before: set[UUID]) -> None:
    client.auth.delete_keys(
        *[k.key for k in client.auth.retrieve_keys() if k.key not in before]
    )


@pytest.mark.cli
class TestLogin:
    def test_saves_api_key(
        self,
        clusters: Clusters,
        keyring: MemoryKeyring,
        login_info: tuple[str, int, str, str],
        client: sy.Synnax,
    ) -> None:
        """Should save an API key in place of the password it logged in with."""
        host, port, username, password = login_info
        before = {k.key for k in client.auth.retrieve_keys()}
        mock = console.Mock(responses=[host, port, username, password, False])
        login(Context(console=mock), clusters)
        saved = clusters.get()
        assert saved is not None
        assert saved.options.api_key.startswith("syk_")
        assert saved.options.username == ""
        assert ("synnax", "default") not in keyring.values
        delete_new_keys(client, before)

    def test_replaces_previous_key(
        self,
        clusters: Clusters,
        login_info: tuple[str, int, str, str],
        client: sy.Synnax,
    ) -> None:
        """Should delete the key an earlier login saved once the new one is saved."""
        host, port, username, password = login_info
        before = {k.key for k in client.auth.retrieve_keys()}
        responses = [host, port, username, password, False]
        login(Context(console=console.Mock(responses=list(responses))), clusters)
        first = clusters.get()
        assert first is not None
        login(Context(console=console.Mock(responses=list(responses))), clusters)
        second = clusters.get()
        assert second is not None
        assert second.options.api_key != first.options.api_key
        remaining = {k.key for k in client.auth.retrieve_keys()} - before
        assert remaining == {sy.auth.api_key_id(second.options.api_key)}
        delete_new_keys(client, before)

    def test_keeps_another_accounts_key(
        self,
        clusters: Clusters,
        login_info: tuple[str, int, str, str],
        client: sy.Synnax,
    ) -> None:
        """Should log in as a second account and keep the key the first one saved."""
        host, port, username, password = login_info
        before = {k.key for k in client.auth.retrieve_keys()}
        login(
            Context(
                console=console.Mock(responses=[host, port, username, password, False])
            ),
            clusters,
        )
        first = clusters.get()
        assert first is not None
        other_username = f"cli-{uuid4()}"
        other = client.users.create(username=other_username, password="password123")
        login(
            Context(
                console=console.Mock(
                    responses=[host, port, other_username, "password123", False]
                )
            ),
            clusters,
        )
        first_key = sy.auth.api_key_id(first.options.api_key)
        assert first_key in {k.key for k in client.auth.retrieve_keys()}
        client.users.delete(other.key)
        delete_new_keys(client, before)


@pytest.mark.cli
class TestFromClusters:
    def test_prompts_without_saved_options(
        self, clusters: Clusters, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should prompt for options and save nothing when none are saved."""
        host, port, username, password = login_info
        mock = console.Mock(responses=[host, port, username, password, False])
        assert connect.from_clusters(Context(console=mock), clusters) is not None
        assert clusters.get() is None

    def test_replaces_saved_password(
        self,
        clusters: Clusters,
        keyring: MemoryKeyring,
        login_info: tuple[str, int, str, str],
        client: sy.Synnax,
    ) -> None:
        """Should replace a saved password with an API key after connecting."""
        host, port, username, password = login_info
        before = {k.key for k in client.auth.retrieve_keys()}
        save_legacy_password(
            clusters,
            keyring,
            Options(host=host, port=port, username=username, password=password),
        )
        connected = connect.from_clusters(Context(console=console.Mock()), clusters)
        assert connected is not None
        assert ("synnax", "default") not in keyring.values
        saved = clusters.get()
        assert saved is not None
        assert saved.options.api_key.startswith("syk_")
        keyed = connect.from_clusters(Context(console=console.Mock()), clusters)
        assert keyed is not None
        assert keyed.auth.session is not None
        created = {k.key for k in client.auth.retrieve_keys()} - before
        assert keyed.auth.session.credential in created
        delete_new_keys(client, before)

    def test_keeps_password_when_save_fails(
        self,
        clusters: Clusters,
        keyring: MemoryKeyring,
        login_info: tuple[str, int, str, str],
        client: sy.Synnax,
    ) -> None:
        """Should connect, keep the saved password, and delete the new key when the
        config write fails.
        """
        host, port, username, password = login_info
        before = {k.key for k in client.auth.retrieve_keys()}
        save_legacy_password(
            clusters,
            keyring,
            Options(host=host, port=port, username=username, password=password),
        )
        block_config_writes(clusters)
        connected = connect.from_clusters(Context(console=console.Mock()), clusters)
        assert connected is not None
        assert keyring.values == {("synnax", "default"): password}
        assert {k.key for k in client.auth.retrieve_keys()} == before

    def test_keeps_saved_key_when_cleanup_fails(
        self,
        tmp_path: Path,
        login_info: tuple[str, int, str, str],
        client: sy.Synnax,
    ) -> None:
        """Should keep the saved key on the Core when the old password cannot be
        deleted.
        """
        host, port, username, password = login_info
        before = {k.key for k in client.auth.retrieve_keys()}
        keyring = UndeletableKeyring()
        clusters = Clusters(File(config_dir=tmp_path), keyring)
        save_legacy_password(
            clusters,
            keyring,
            Options(host=host, port=port, username=username, password=password),
        )
        assert (
            connect.from_clusters(Context(console=console.Mock()), clusters) is not None
        )
        keyed = connect.from_clusters(Context(console=console.Mock()), clusters)
        assert keyed is not None
        assert keyed.auth.session is not None
        assert keyed.auth.session.credential not in before
        delete_new_keys(client, before)

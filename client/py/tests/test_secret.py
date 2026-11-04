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

import synnax as sy
from synnax import secret
from x.testutil import assert_eventually


@pytest.mark.secret
class TestSecret:
    def test_create_stored(self, client: sy.Synnax) -> None:
        """Should create a stored secret without returning its value."""
        s = client.secrets.create(secret.NewStored(name="Password", value="hunter2"))
        assert isinstance(s, secret.StoredSecret)
        assert s.name == "Password"
        assert "value" not in s.model_dump()
        assert s.updated > 0

    def test_create_file(self, client: sy.Synnax) -> None:
        """Should create a file secret."""
        s = client.secrets.create(
            secret.NewFile(name="Client key", path="/etc/synnax/client.key")
        )
        assert isinstance(s, secret.FileSecret)
        assert s.path == "/etc/synnax/client.key"

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple secrets."""
        secrets = client.secrets.create(
            [
                secret.NewStored(name="Token", value="abc"),
                secret.NewFile(name="Key file", path="/tmp/key.pem"),
            ]
        )
        assert isinstance(secrets[0], secret.StoredSecret)
        assert isinstance(secrets[1], secret.FileSecret)

    def test_replace(self, client: sy.Synnax) -> None:
        """Should replace a secret that has the same key."""
        first = client.secrets.create(secret.NewStored(name="API key", value="one"))
        second = client.secrets.create(
            secret.NewFile(key=first.key, name="API key v2", path="/opt/api.key")
        )
        assert second.key == first.key
        assert second.updated > first.updated
        assert client.secrets.retrieve(key=first.key) == second

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve secrets by key."""
        created = client.secrets.create(
            [
                secret.NewStored(name="Multi 1", value="a"),
                secret.NewStored(name="Multi 2", value="b"),
            ]
        )
        retrieved = client.secrets.retrieve(keys=[s.key for s in created])
        assert sorted(s.name for s in retrieved) == ["Multi 1", "Multi 2"]

    def test_retrieve_with_search_term(self, client: sy.Synnax) -> None:
        """Should search for secrets by name."""
        name = f"searchable-secret-{uuid4()}"
        created = client.secrets.create(secret.NewStored(name=name, value="abc"))

        def check() -> None:
            assert created in client.secrets.retrieve(search_term=name)

        assert_eventually(check)

    def test_retrieve_missing(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when no secret has the key."""
        with pytest.raises(sy.NotFoundError):
            client.secrets.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a secret."""
        s = client.secrets.create(secret.NewStored(name="Before", value="abc"))
        client.secrets.rename(s.key, "After")
        assert client.secrets.retrieve(key=s.key).name == "After"

    def test_rename_empty(self, client: sy.Synnax) -> None:
        """Should reject an empty name."""
        s = client.secrets.create(secret.NewStored(name="Keep", value="abc"))
        with pytest.raises(sy.PathError, match=r"^name: required$"):
            client.secrets.rename(s.key, "")
        assert client.secrets.retrieve(key=s.key).name == "Keep"

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete secrets."""
        secrets = client.secrets.create(
            [
                secret.NewStored(name="Delete 1", value="a"),
                secret.NewFile(name="Delete 2", path="/tmp/b"),
            ]
        )
        client.secrets.delete([s.key for s in secrets])
        with pytest.raises(sy.NotFoundError):
            client.secrets.retrieve(key=secrets[0].key)

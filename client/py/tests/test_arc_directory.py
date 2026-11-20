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


@pytest.mark.arc
class TestArcDirectory:
    """Tests for the directories that hold Arcs."""

    def test_root(self, client: sy.Synnax) -> None:
        """Should find the Arcs root."""
        root = client.arcs.directories.root()
        assert client.arcs.directories.root() == root

    def test_create_and_delete(self, client: sy.Synnax) -> None:
        """Should create a directory under the root and delete it."""
        name = f"dir_{uuid4().hex}"
        d = client.arcs.directories.create(name, parent=client.arcs.directories.root())
        assert d.name == name
        client.arcs.directories.delete(d.key)

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a directory so its old name is free and its new one taken."""
        dirs = client.arcs.directories
        root = dirs.root()
        name = f"dir_{uuid4().hex}"
        d = dirs.create(name, parent=root)
        renamed = f"dir_{uuid4().hex}"
        dirs.rename(d.key, renamed)
        with pytest.raises(sy.ValidationError):
            dirs.create(renamed, parent=root)
        freed = dirs.create(name, parent=root)
        dirs.delete([d.key, freed.key])

    def test_move_module_into_directory(self, client: sy.Synnax) -> None:
        """Should move a module from the root into a directory and back."""
        dirs = client.arcs.directories
        root = dirs.root()
        d = dirs.create(f"dir_{uuid4().hex}", parent=root)
        module = client.arcs.create(
            name=f"mod_{uuid4().hex}", mode="text", type="module"
        )
        dirs.move(d.key, module.resource_id)
        assert client.arcs.retrieve(key=module.key).directory == d.key
        dirs.move(root, module.resource_id)
        assert client.arcs.retrieve(key=module.key).directory == root
        client.arcs.delete(module.key)
        dirs.delete(d.key)

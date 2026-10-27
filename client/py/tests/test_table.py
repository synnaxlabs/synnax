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
from synnax.exceptions import NotFoundError


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name="Table Project")


@pytest.mark.table
class TestTable:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create an empty table from a name."""
        t = client.tables.create(project.key, name="Test Table")
        assert t.name == "Test Table"
        assert t.rows == []

    def test_create_from_object(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a table with rows, columns, and cells."""
        row = sy.table.Row(cells=["a", "b"])
        t = client.tables.create(
            project.key,
            sy.table.Table(
                name="Full Table",
                rows=[row],
                columns=[sy.table.Column(), sy.table.Column(size=120)],
                cells={
                    "a": sy.table.TextCellConfig(value="Pressure"),
                    "b": sy.table.ValueCellConfig(units="psi"),
                },
            ),
        )
        retrieved = client.tables.retrieve(key=t.key)
        assert retrieved.rows == [row]
        assert retrieved.columns[1].size == 120
        assert retrieved.cells["a"] == sy.table.TextCellConfig(value="Pressure")
        assert retrieved.cells["b"] == sy.table.ValueCellConfig(units="psi")

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple tables."""
        tables = client.tables.create(
            project.key,
            [sy.table.Table(name="Multi 1"), sy.table.Table(name="Multi 2")],
        )
        assert [t.name for t in tables] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple tables by key."""
        t1 = client.tables.create(project.key, name="Table A")
        t2 = client.tables.create(project.key, name="Table B")
        tables = client.tables.retrieve(keys=[t1.key, t2.key])
        assert {t.key for t in tables} == {t1.key, t2.key}

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when a table does not exist."""
        with pytest.raises(NotFoundError):
            client.tables.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a table."""
        t = client.tables.create(project.key, name="Before Rename")
        client.tables.rename(t.key, "After Rename")
        assert client.tables.retrieve(key=t.key).name == "After Rename"

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple tables."""
        t1 = client.tables.create(project.key, name="Delete 1")
        t2 = client.tables.create(project.key, name="Delete 2")
        client.tables.delete([t1.key, t2.key])
        with pytest.raises(NotFoundError):
            client.tables.retrieve(keys=[t1.key, t2.key])

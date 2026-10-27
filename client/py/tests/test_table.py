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
from pydantic import ValidationError

import synnax as sy
from synnax import table
from synnax.exceptions import NotFoundError


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name="Table Project")


class TestTable:
    def test_round_trip(self) -> None:
        """Should parse rows, columns, and cells and dump them back unchanged."""
        t = table.Table.model_validate(
            {
                "name": "Readings",
                "rows": [{"key": "r", "cells": ["a", "b"]}],
                "columns": [{"key": "c1"}, {"key": "c2", "size": 100}],
                "cells": {
                    "a": {"variant": "text", "value": "Pressure"},
                    "b": {"variant": "value", "channel": 1, "units": "psi"},
                },
            }
        )
        assert isinstance(t.cells["a"], table.TextCellConfig)
        assert isinstance(t.cells["b"], table.ValueCellConfig)
        assert t.rows[0].size == 36
        assert t.columns[0].size == 72
        assert table.Table.model_validate_json(t.model_dump_json()) == t

    def test_cell(self) -> None:
        """Should select the cell config variant of a keyed cell."""
        cell = table.Cell.model_validate({"key": "a", "config": {"variant": "value"}})
        assert isinstance(cell.config, table.ValueCellConfig)
        assert cell.config.redline.bands == []

    def test_row_size(self) -> None:
        """Should reject a row smaller than one pixel."""
        with pytest.raises(ValidationError):
            table.Row(size=0)

    def test_ontology_id(self) -> None:
        """Should build the ontology ID of a table."""
        key = uuid4()
        assert table.ontology_id(key).type == table.ONTOLOGY_TYPE.type
        assert table.ontology_id(key).key == str(key)
        t = table.Table(name="Readings")
        assert hash(t) == hash(t.key)


@pytest.mark.table
class TestClient:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create an empty table from a name."""
        t = client.tables.create(project.key, name="Test Table")
        assert t.name == "Test Table"
        assert t.rows == []

    def test_create_from_object(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should persist the rows, columns, and cells of a table."""
        row = table.Row(cells=["a", "b"])
        t = client.tables.create(
            project.key,
            table.Table(
                name="Full Table",
                rows=[row],
                columns=[table.Column(), table.Column(size=120)],
                cells={
                    "a": table.TextCellConfig(value="Pressure"),
                    "b": table.ValueCellConfig(units="psi"),
                },
            ),
        )
        retrieved = client.tables.retrieve(key=t.key)
        assert retrieved.rows == [row]
        assert retrieved.columns[1].size == 120
        assert retrieved.cells["a"] == table.TextCellConfig(value="Pressure")
        assert retrieved.cells["b"] == table.ValueCellConfig(units="psi")

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple tables."""
        tables = client.tables.create(
            project.key,
            [table.Table(name="Multi 1"), table.Table(name="Multi 2")],
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

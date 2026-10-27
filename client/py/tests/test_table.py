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

from synnax import table


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

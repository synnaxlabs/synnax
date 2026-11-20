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
from synnax import calculation
from x.testutil import untyped


def _virtual(client: sy.Synnax) -> sy.Channel:
    return client.channels.create(
        name=f"calc_{uuid4().hex[:8]}", data_type=sy.DataType.FLOAT32, virtual=True
    )


@pytest.mark.calculation
class TestCalculationClient:
    """Tests for the calculation client."""

    def test_create_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create a calculation from an output and an expression."""
        out = _virtual(client)
        created = client.calculations.create(output=out.key, expression="return 2")
        assert created.output == out.key
        assert client.calculations.retrieve(key=created.key) == created

    def test_create_requires_output(self, client: sy.Synnax) -> None:
        """Should reject a keyword create that leaves out the output."""
        with pytest.raises(sy.ValidationError, match="needs an output"):
            untyped(client.calculations.create)(expression="return 2")

    def test_create_on_stored_output(self, client: sy.Synnax) -> None:
        """Should refuse an output that stores its samples."""
        index = client.channels.create(
            name=f"time_{uuid4().hex[:8]}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        stored = client.channels.create(
            name=f"stored_{uuid4().hex[:8]}",
            data_type=sy.DataType.FLOAT32,
            index=index.key,
        )
        with pytest.raises(sy.ValidationError, match="stores its samples"):
            client.calculations.create(output=stored.key, expression="return 2")

    def test_create_on_written_output(self, client: sy.Synnax) -> None:
        """Should refuse an output that another calculation writes."""
        out = _virtual(client)
        client.calculations.create(output=out.key, expression="return 2")
        with pytest.raises(sy.ValidationError, match="already writes"):
            client.calculations.create(output=out.key, expression="return 3")

    def test_retrieve_by_output(self, client: sy.Synnax) -> None:
        """Should retrieve the calculation of an output."""
        out = _virtual(client)
        created = client.calculations.create(output=out.key, expression="return 2")
        assert client.calculations.retrieve(output=out.key, one=True) == created

    def test_set_expression(self, client: sy.Synnax) -> None:
        """Should set the expression of a calculation."""
        out = _virtual(client)
        created = client.calculations.create(output=out.key, expression="return 2")
        client.calculations.dispatch(
            created.key,
            calculation.set(
                calculation.SetPayload(path=["expression"], value="return 4")
            ),
        )
        assert client.calculations.retrieve(key=created.key).expression == "return 4"

    def test_delete_keeps_output(self, client: sy.Synnax) -> None:
        """Should delete a calculation and keep its output."""
        out = _virtual(client)
        created = client.calculations.create(output=out.key, expression="return 2")
        client.calculations.delete(created.key)
        assert client.calculations.retrieve(keys=[created.key]) == []
        assert client.channels.retrieve(out.key).key == out.key

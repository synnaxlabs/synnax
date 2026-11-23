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
        """Should reject a keyword create that names neither an output nor a name."""
        with pytest.raises(sy.ValidationError, match="name"):
            untyped(client.calculations.create)(expression="return 2")

    def test_create_with_name(self, client: sy.Synnax) -> None:
        """Should make a virtual output with the given name, typed by the expression."""
        name = f"calc_{uuid4().hex[:8]}"
        created = client.calculations.create(name=name, expression="return 2.5")
        output = client.channels.retrieve(created.output)
        assert output.name == name
        assert output.virtual
        assert output.data_type == sy.DataType.FLOAT64

    def test_create_with_name_and_group(self, client: sy.Synnax) -> None:
        """Should make the output in the given group."""
        group = client.groups.create(f"calc_{uuid4().hex[:8]}", scope="channel")
        created = client.calculations.create(
            name=f"calc_{uuid4().hex[:8]}", expression="return 2", group=group.key
        )
        assert client.channels.retrieve(created.output).group == group.key

    def test_create_rejects_output_and_name(self, client: sy.Synnax) -> None:
        """Should reject a keyword create that names an output and a name."""
        out = _virtual(client)
        with pytest.raises(sy.ValidationError, match="name"):
            untyped(client.calculations.create)(
                output=out.key, name="doubled", expression="return 2"
            )

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

    def test_set_expression_retypes_output(self, client: sy.Synnax) -> None:
        """Should give the output the data type of a new expression."""
        created = client.calculations.create(
            name=f"calc_{uuid4().hex[:8]}", expression="return 1 + 1"
        )
        assert client.channels.retrieve(created.output).data_type == sy.DataType.INT64
        client.calculations.dispatch(
            created.key,
            calculation.set(
                calculation.SetPayload(path=["expression"], value="return 1.5 * 2.0")
            ),
        )
        output = client.channels.retrieve(created.output)
        assert output.data_type == sy.DataType.FLOAT64

    @pytest.mark.parametrize(
        "operation",
        [
            calculation.AvgOperation(duration=sy.TimeSpan.SECOND * 10),
            calculation.MaxOperation(duration=sy.TimeSpan.SECOND * 5),
            calculation.DerivativeOperation(),
        ],
    )
    def test_keep_operations(
        self, client: sy.Synnax, operation: calculation.Operation
    ) -> None:
        """Should keep the operations of a calculation through a retrieve."""
        src = _virtual(client)
        created = client.calculations.create(
            output=_virtual(client).key,
            expression=f"return {src.name}",
            operations=[operation],
        )
        assert client.calculations.retrieve(key=created.key).operations == [operation]

    def test_keep_reset_channel(self, client: sy.Synnax) -> None:
        """Should keep the channel that resets the window of an operation."""
        src = _virtual(client)
        reset = client.channels.create(
            name=f"reset_{uuid4().hex[:8]}", data_type=sy.DataType.UINT8, virtual=True
        )
        operation = calculation.MinOperation(reset=reset.key)
        created = client.calculations.create(
            output=_virtual(client).key,
            expression=f"return {src.name}",
            operations=[operation],
        )
        assert client.calculations.retrieve(key=created.key).operations == [operation]

    def test_delete_keeps_output(self, client: sy.Synnax) -> None:
        """Should delete a calculation and keep its output."""
        out = _virtual(client)
        created = client.calculations.create(output=out.key, expression="return 2")
        client.calculations.delete(created.key)
        assert client.calculations.retrieve(keys=[created.key]) == []
        assert client.channels.retrieve(out.key).key == out.key

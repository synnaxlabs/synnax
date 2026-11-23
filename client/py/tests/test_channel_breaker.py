#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from uuid import uuid4

import pytest

import synnax as sy
from x.testutil import untyped


def _name() -> str:
    return f"brk_{uuid4().hex[:8]}"


@pytest.mark.channel
class TestChannelBreaker:
    def test_validate_dump_of_new_channel(self) -> None:
        """Should load a channel from its own dump."""
        ch = sy.Channel(name=_name(), data_type=sy.DataType.FLOAT32, virtual=True)
        assert sy.Channel.model_validate(ch.model_dump()).name == ch.name

    def test_validate_json_of_retrieved_channel(self, client: sy.Synnax) -> None:
        """Should load a retrieved channel from its own JSON."""
        created = client.channels.create(
            name=_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        ch = client.channels.retrieve(created.key)
        assert sy.Channel.model_validate_json(ch.model_dump_json()).key == ch.key

    def test_to_payload_of_calculated_output(self, client: sy.Synnax) -> None:
        """Should keep what the Core says about a calculated channel in its payload."""
        calc = client.calculations.create(name=_name(), expression="return 1 + 1")
        ch = client.channels.retrieve(calc.output)
        assert ch.calculated is True
        payload = ch.to_payload()
        assert payload.calculated is True
        assert payload.expression == "return 1 + 1"

    def test_rename_retrieved_channel(self, client: sy.Synnax) -> None:
        """Should rename a channel that the client returned."""
        ch = client.channels.create(
            name=_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        name = _name()
        ch.rename(name)
        assert client.channels.retrieve(ch.key).name == name

    def test_reset_channel_named_as_reference_docs(self, client: sy.Synnax) -> None:
        """Should keep a reset channel set by the name the reference page gives."""
        src = client.channels.create(
            name=_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        reset = client.channels.create(
            name=_name(), data_type=sy.DataType.BOOLEAN, virtual=True
        )
        operation = untyped(sy.calculation.AvgOperation)(reset_channel=reset.key)
        calc = client.calculations.create(
            name=_name(), expression=f"return {src.name}", operations=[operation]
        )
        retrieved = client.calculations.retrieve(key=calc.key).operations[0]
        assert isinstance(retrieved, sy.calculation.AvgOperation)
        assert retrieved.reset == reset.key

    def test_calculated_attribute(self, client: sy.Synnax) -> None:
        """Should read a calculation output as calculated, a plain virtual not."""
        calc = client.calculations.create(name=_name(), expression="return 1 + 1")
        plain = client.channels.create(
            name=_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        assert client.channels.retrieve(calc.output).calculated is True
        assert client.channels.retrieve(plain.key).calculated is False

    def test_channel_reads_new_expression(self, client: sy.Synnax) -> None:
        """Should show the new expression of a calculation on its output."""
        calc = client.calculations.create(name=_name(), expression="return 1 + 1")
        client.calculations.dispatch(
            calc.key,
            sy.calculation.set(
                sy.calculation.SetPayload(path=["expression"], value="return 1.5")
            ),
        )
        ch = client.channels.retrieve(calc.output)
        assert ch.expression == "return 1.5"
        assert ch.data_type == sy.DataType.FLOAT64

    def test_channel_reads_operations(self, client: sy.Synnax) -> None:
        """Should show the operations of a calculation on its output."""
        idx = client.channels.create(name=_name(), is_index=True)
        src = client.channels.create(
            name=_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=_name(), data_type=sy.DataType.BOOLEAN, index=idx.key
        )
        calc = client.calculations.create(
            name=_name(),
            expression=f"return {src.name}",
            operations=[sy.calculation.MinOperation(reset=reset.key)],
        )
        ch = client.channels.retrieve(calc.output)
        assert ch.virtual is True
        assert len(ch.operations) == 1
        assert ch.operations[0].type == "min"
        assert ch.operations[0].reset_channel == reset.key
        assert ch.operations[0].duration == 0

    def test_example_create_step_twice_at_once(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should let two runs of the stream_write example start at once."""
        host, port, username, password = login_info
        name = _name()
        barrier = threading.Barrier(2)
        errors: list[Exception] = []

        def run() -> None:
            c = sy.Synnax(
                host=host, port=port, username=username, password=password, secure=False
            )
            barrier.wait()
            try:
                if not c.channels.retrieve([name]):
                    c.calculations.create(name=name, expression="return 1")
            except Exception as e:
                errors.append(e)
            finally:
                c.close()

        threads = [threading.Thread(target=run) for _ in range(2)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        assert errors == []

    def test_create_retrieved_calculated_channel_again(self, client: sy.Synnax) -> None:
        """Should refuse to create a retrieved calculated channel a second time."""
        calc = client.calculations.create(name=_name(), expression="return 1.5")
        ch = client.channels.retrieve(calc.output)
        with pytest.raises(sy.PathError, match="already exists"):
            client.channels.create(ch)
        assert len(client.channels.retrieve([ch.name])) == 1

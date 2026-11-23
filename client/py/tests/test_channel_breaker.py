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

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from synnax.channel.client import Channel
from x.strings import random_name


@pytest.mark.channel
class TestChannelDescription:
    def test_create_and_retrieve_a_description(self, client: sy.Synnax) -> None:
        ch = client.channels.create(
            name=f"description_{random_name()}",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            description="Pressure at the inlet",
        )
        assert ch.description == "Pressure at the inlet"
        retrieved = client.channels.retrieve(ch.key)
        assert retrieved.description == "Pressure at the inlet"

    def test_default_to_no_description(self) -> None:
        ch = Channel(name="pt_02", data_type=sy.DataType.FLOAT32)
        assert ch.description == ""

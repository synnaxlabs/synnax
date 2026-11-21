#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from types import SimpleNamespace

import pytest

import synnax as sy
from synnax.channel.client import Channel, Client
from synnax.channel.payload import Payload


@pytest.mark.channel
class TestChannelDescription:
    def test_sugar_a_payload_with_a_description(self) -> None:
        """Every create and retrieve turns the payloads of the response into channels
        through Client._sugar."""
        payload = Payload(
            key=1,
            name="pt_01",
            data_type=sy.DataType.FLOAT32,
            description="Pressure at the inlet",
        )
        owner = SimpleNamespace(_frame_client=None)
        channels = Client._sugar(owner, [payload])  # type: ignore[arg-type]
        assert channels[0].description == "Pressure at the inlet"

    def test_default_to_no_description(self) -> None:
        ch = Channel(name="pt_02", data_type=sy.DataType.FLOAT32)
        assert ch.description == ""

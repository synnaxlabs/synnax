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
from synnax.channel.client import Client
from synnax.channel.payload import Payload


@pytest.mark.channel
class TestChannelPathBreaker:
    def test_sugar_a_payload_with_no_path(self) -> None:
        """Every create and retrieve turns the payloads of the response into channels
        through Client._sugar."""
        payload = Payload(key=1, name="pt_01", data_type=sy.DataType.FLOAT32)
        owner = SimpleNamespace(_frame_client=None)
        channels = Client._sugar(owner, [payload])  # type: ignore[arg-type]
        assert channels[0].name == "pt_01"

    def test_sugar_a_payload_with_a_path(self) -> None:
        payload = Payload(
            key=1,
            name="pt_01",
            data_type=sy.DataType.FLOAT32,
            path="stand_a.pt_01",
        )
        owner = SimpleNamespace(_frame_client=None)
        channels = Client._sugar(owner, [payload])  # type: ignore[arg-type]
        assert channels[0].path == "stand_a.pt_01"

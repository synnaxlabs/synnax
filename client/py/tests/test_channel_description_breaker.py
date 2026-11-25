#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import pytest

import synnax as sy
from synnax.channel.client import Channel, Client
from synnax.channel.payload import Payload


class _Creator:
    """Records the payloads that a create sends and returns them as the Core would."""

    def __init__(self) -> None:
        self.sent: list[Payload] = []

    def create(
        self, channels: list[Payload], retrieve_if_name_exists: bool
    ) -> list[Payload]:
        self.sent = channels
        return [c.model_copy(update={"key": i + 1}) for i, c in enumerate(channels)]


def _client(creator: _Creator) -> Client:
    none: Any = None
    return Client(client=none, frame_client=none, retriever=none, creator=creator)


@pytest.mark.channel
class TestChannelDescriptionBreaker:
    def test_to_payload_keeps_the_description(self) -> None:
        ch = Channel(
            name="pt_01", data_type=sy.DataType.FLOAT32, description="Inlet pressure"
        )
        assert ch.to_payload().description == "Inlet pressure"

    def test_create_of_a_channel_sends_its_description(self) -> None:
        creator = _Creator()
        created = _client(creator).create(
            Channel(
                name="pt_02",
                data_type=sy.DataType.FLOAT32,
                virtual=True,
                description="Inlet pressure",
            )
        )
        assert creator.sent[0].description == "Inlet pressure"
        assert created.description == "Inlet pressure"

    def test_create_by_keyword_takes_a_description(self) -> None:
        creator = _Creator()
        create: Any = _client(creator).create
        create(
            name="pt_03",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            description="Inlet pressure",
        )
        assert creator.sent[0].description == "Inlet pressure"

    def test_cap_counts_code_points_as_the_core_does(self) -> None:
        Channel(
            name="pt_04",
            data_type=sy.DataType.FLOAT32,
            description="\U0001f600" * 1024,
        )
        with pytest.raises(ValueError):
            Channel(
                name="pt_05",
                data_type=sy.DataType.FLOAT32,
                description="\U0001f600" * 1025,
            )

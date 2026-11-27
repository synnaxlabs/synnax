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
from x.strings import random_name


@pytest.mark.channel
class TestChannelPathBreaker:
    def test_retrieve_a_channel_with_no_path(self, client: sy.Synnax) -> None:
        """A retrieve that asks for no path returns each channel with none."""
        ch = client.channels.create(
            name=f"pt_{random_name()}",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
        )
        assert client.channels.retrieve(ch.key).path is None

    def test_retrieve_the_path_of_a_channel_in_a_folder(
        self, client: sy.Synnax
    ) -> None:
        folder = client.groups.create(f"stand_{random_name()}", scope="channel")
        ch = client.channels.create(
            name="pt_01",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            group=folder.key,
        )
        got = client.channels.retrieve(ch.key, include_path=True)
        assert got.path == f"{folder.name}.pt_01"

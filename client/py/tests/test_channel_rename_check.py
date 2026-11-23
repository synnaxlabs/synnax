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
class TestChannelRenameCheck:
    def test_rename_a_retrieved_channel(self, client: sy.Synnax) -> None:
        """Should rename a channel that the client retrieved"""
        created = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        retrieved = client.channels.retrieve(created.key)
        name = random_name()
        retrieved.rename(name)
        assert client.channels.retrieve(created.key).name == name

    def test_rename_a_created_channel(self, client: sy.Synnax) -> None:
        """Should rename a channel that the client created"""
        created = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        name = random_name()
        created.rename(name)
        assert client.channels.retrieve(created.key).name == name

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
class TestChannelRecreateCheck:
    def test_create_a_retrieved_channel_again(self, client: sy.Synnax) -> None:
        """Should refuse a second channel with the name of a retrieved channel"""
        ch = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        with pytest.raises(sy.PathError, match="already exists"):
            client.channels.create(client.channels.retrieve(ch.key))
        assert [c.key for c in client.channels.retrieve([ch.name])] == [ch.key]

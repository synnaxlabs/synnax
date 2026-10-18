#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import synnax as sy
from synnax.state import State


class TestState:
    """Tests reading channel values from State by key, name and string key."""

    @staticmethod
    def state() -> State:
        ch = sy.channel.Payload(key=12, name="pressure", data_type=sy.DataType.FLOAT32)
        state = State([ch])
        state.update(
            sy.Frame([ch.key], [sy.Series([1.5], data_type=sy.DataType.FLOAT32)])
        )
        return state

    def test_by_key(self) -> None:
        """Should return the series for a channel key."""
        assert self.state()[12].to_numpy().tolist() == [1.5]

    def test_by_name(self) -> None:
        """Should return the series for a channel name."""
        assert self.state()["pressure"].to_numpy().tolist() == [1.5]

    def test_by_numeric_string_key(self) -> None:
        """Should read a numeric string as a channel key, not a name."""
        assert self.state()["12"].to_numpy().tolist() == [1.5]

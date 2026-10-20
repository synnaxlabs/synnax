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
from synnax.state import LatestState, State


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


class TestStateAttributes:
    """Tests reading State and LatestState values as attributes."""

    @staticmethod
    def state() -> State:
        ch = sy.channel.Payload(key=12, name="pressure", data_type=sy.DataType.FLOAT32)
        state = State([ch])
        state.update(
            sy.Frame([ch.key], [sy.Series([1.5], data_type=sy.DataType.FLOAT32)])
        )
        return state

    def test_state_attribute(self) -> None:
        """Should return the series for a channel name read as an attribute."""
        assert self.state().pressure.to_numpy().tolist() == [1.5]

    def test_latest_state_attribute(self) -> None:
        """Should return the latest value for a channel name read as an attribute."""
        assert LatestState(self.state()).pressure == 1.5

    def test_state_missing_attribute(self) -> None:
        """Should raise AttributeError for a name that is not a channel."""
        state = self.state()
        assert not hasattr(state, "temperature")
        with pytest.raises(AttributeError):
            _ = state.temperature

    def test_latest_state_missing_attribute(self) -> None:
        """Should raise AttributeError for a name that is not a channel."""
        latest = LatestState(self.state())
        assert not hasattr(latest, "temperature")
        with pytest.raises(AttributeError):
            _ = latest.temperature

    def test_missing_item(self) -> None:
        """Should raise KeyError for an item that is not a channel."""
        with pytest.raises(KeyError):
            _ = self.state()["temperature"]


class TestStateUpdateByName:
    """Tests updating State from a frame keyed by channel name."""

    def test_update_by_name(self) -> None:
        """Should store a frame entry keyed by name under the channel's key."""
        ch = sy.channel.Payload(key=12, name="pressure", data_type=sy.DataType.FLOAT32)
        state = State([ch])
        state.update(
            sy.Frame([ch.name], [sy.Series([2.5], data_type=sy.DataType.FLOAT32)])
        )
        assert state[12].to_numpy().tolist() == [2.5]

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
from synnax.state import Latest, State

PRESSURE = sy.channel.Payload(key=12, name="pressure", data_type=sy.DataType.FLOAT32)
TEMPERATURE = sy.channel.Payload(
    key=13, name="temperature", data_type=sy.DataType.FLOAT32
)


def frame(ch: sy.channel.Payload, data: list[float]) -> sy.Frame:
    return sy.Frame([ch.key], [sy.Series(data, data_type=sy.DataType.FLOAT32)])


class TestState:
    """Tests reading channel values from State by key, name and string key."""

    @staticmethod
    def state() -> State:
        state = State([PRESSURE, TEMPERATURE])
        state.update(frame(PRESSURE, [1.5]))
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

    def test_by_attribute(self) -> None:
        """Should return the series for a channel name read as an attribute."""
        assert self.state().pressure.to_numpy().tolist() == [1.5]

    def test_update_replaces(self) -> None:
        """Should replace the series of a channel in a later frame."""
        state = self.state()
        state.update(frame(PRESSURE, [2.5, 3.5]))
        assert state["pressure"].to_numpy().tolist() == [2.5, 3.5]

    def test_update_keeps_absent(self) -> None:
        """Should keep the series of a channel absent from a later frame."""
        state = self.state()
        state.update(frame(TEMPERATURE, [20.0]))
        assert state["pressure"].to_numpy().tolist() == [1.5]
        assert state["temperature"].to_numpy().tolist() == [20.0]

    def test_no_data(self) -> None:
        """Should raise a KeyError for a channel with no data yet."""
        with pytest.raises(KeyError):
            self.state()["temperature"]

    def test_unknown_name(self) -> None:
        """Should raise a KeyError for a name outside the state's channels."""
        with pytest.raises(KeyError):
            self.state()["humidity"]

    def test_missing_attribute(self) -> None:
        """Should raise AttributeError for a name that is not a channel."""
        state = self.state()
        assert not hasattr(state, "humidity")
        with pytest.raises(AttributeError):
            _ = state.humidity

    def test_update_by_name(self) -> None:
        """Should store a frame entry keyed by name under the channel's key."""
        state = State([PRESSURE])
        state.update(
            sy.Frame([PRESSURE.name], [sy.Series([2.5], data_type=sy.DataType.FLOAT32)])
        )
        assert state[12].to_numpy().tolist() == [2.5]


class TestLatest:
    """Tests reading the latest sample of each channel from Latest."""

    @staticmethod
    def latest() -> Latest:
        state = State([PRESSURE])
        state.update(frame(PRESSURE, [1.5, 2.5]))
        return Latest(state)

    def test_by_key(self) -> None:
        """Should return the last sample for a channel key."""
        assert self.latest()[12] == 2.5

    def test_by_name(self) -> None:
        """Should return the last sample for a channel name."""
        assert self.latest()["pressure"] == 2.5

    def test_by_attribute(self) -> None:
        """Should return the last sample for a channel name read as an attribute."""
        assert self.latest().pressure == 2.5

    def test_missing_attribute(self) -> None:
        """Should raise AttributeError for a name that is not a channel."""
        latest = self.latest()
        assert not hasattr(latest, "humidity")
        with pytest.raises(AttributeError):
            _ = latest.humidity

    def test_follows_updates(self) -> None:
        """Should read from the state's latest frame."""
        state = State([PRESSURE])
        latest = Latest(state)
        state.update(frame(PRESSURE, [1.5]))
        state.update(frame(PRESSURE, [4.5]))
        assert latest["pressure"] == 4.5

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import pathlib
from collections.abc import Callable, Iterator
from contextlib import contextmanager
from uuid import uuid4

import numpy as np
import pytest

import synnax as sy
from x.telem import seconds_linspace
from x.testutil import untyped


@contextmanager
def deprecated() -> Iterator[None]:
    with pytest.warns(DeprecationWarning, match="by position is deprecated"):
        yield


class _StarterStopper(sy.task.StarterStopperMixin):
    def __init__(self, internal: sy.Task) -> None:
        self._internal = internal


@pytest.fixture
def written_pair(
    indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
) -> tuple[sy.Channel, sy.Channel]:
    idx, data = indexed_pair
    with client.open_writer(start=sy.TimeSpan.SECOND, channels=indexed_pair) as w:
        w.write({idx.key: seconds_linspace(1, count=3), data.key: np.arange(3.0)})
        w.commit()
    return idx, data


@pytest.mark.deprecation
class TestPositionalArguments:
    """Positional forms that 0.60 accepted should still work and warn."""

    def test_synnax(
        self, login_info: tuple[str, int, str, str], virtual_channel: sy.Channel
    ) -> None:
        host, port, username, password = login_info
        with deprecated():
            c = untyped(sy.Synnax)(host, port, username, password)
        try:
            assert c.channels.retrieve(virtual_channel.key).key == virtual_channel.key
        finally:
            c.close()

    def test_range(self) -> None:
        tr = sy.TimeRange(start=1, end=2)
        with deprecated():
            rng = untyped(sy.Range)("range", tr)
        assert rng.name == "range"
        assert rng.time_range == tr

    def test_framer(
        self,
        client: sy.Synnax,
        written_pair: tuple[sy.Channel, sy.Channel],
        tmp_path: pathlib.Path,
    ) -> None:
        idx, data = written_pair
        tr = sy.TimeRange.MAX
        with deprecated():
            untyped(client.read)(tr, data.key, None)
        with deprecated():
            untyped(client.read_latest)(data.key, 1)
        with deprecated():
            untyped(client.read_csv)(tr, data.key, tmp_path / "a.csv")
        with deprecated():
            untyped(client.read_tdms)(tr, data.key, tmp_path / "a.tdms")
        with deprecated():
            untyped(client.open_iterator)(tr, data.key, 100).close()
        with deprecated():
            s = untyped(client.open_streamer)(data.key, 1)
        with deprecated():
            untyped(s.read)(0)
        with deprecated():
            untyped(s.close)(1)
        start = sy.TimeStamp.now()
        with deprecated():
            w = untyped(client.open_writer)(start, [idx.key, data.key], 200)
        with w, deprecated():
            untyped(w.set_authority)(data.key, 255)

    def test_async_streamer(
        self, client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        async def run() -> None:
            with deprecated():
                s = await untyped(client.open_async_streamer)(virtual_channel.key, 1)
            await s.close()

        asyncio.run(run())

    def test_controller(self, client: sy.Synnax, virtual_channel: sy.Channel) -> None:
        cmd = client.channels.create(
            name=f"{virtual_channel.name}_cmd",
            virtual=True,
            data_type=sy.DataType.FLOAT64,
        )
        key = virtual_channel.key
        with deprecated():
            ctrl = untyped(client.control.acquire)("deprecation", [key], [cmd.key, key])
        with ctrl, client.open_streamer(channels=cmd.key) as s:
            assert [ch.key for ch in ctrl._receiver.channels] == [key]
            ctrl[cmd.key] = 2
            frame = s.read(timeout=5)
            assert frame is not None
            assert frame[cmd.key].to_numpy().tolist() == [2]
            with deprecated():
                untyped(ctrl.set_authority)(key, 200)
            ctrl[key] = 1
            with deprecated():
                assert untyped(ctrl.wait_until_defined)(key, 5)
            with deprecated():
                assert untyped(ctrl.wait_until)(lambda _: True, 1)
            with deprecated():
                assert untyped(ctrl.wait_while)(lambda _: False, 1)
            with deprecated():
                untyped(ctrl.remains_true_for)(lambda _: True, 0.01, 1)
            with deprecated():
                untyped(ctrl.sleep)(0, False)

    def test_signals(self, client: sy.Synnax, virtual_channel: sy.Channel) -> None:
        with deprecated():
            untyped(client.signals.on)([virtual_channel.key], lambda _: True)

    def test_role_retrieve(self, client: sy.Synnax, empty_role: sy.Role) -> None:
        with deprecated():
            untyped(client.access.roles.retrieve)(None, [empty_role.key])

    def test_group_create(self, client: sy.Synnax) -> None:
        parent = sy.group.resource_id(
            client.groups.create(str(uuid4()), scope="channel").key
        )
        with deprecated():
            g = untyped(client.groups.create)(parent, "child")
        assert (g.name, g.parent) == ("child", parent)
        with deprecated():
            g = untyped(client.groups.create)(parent, name="named")
        assert (g.name, g.parent) == ("named", parent)
        key = uuid4()
        with deprecated():
            g = untyped(client.groups.create)(parent, "keyed", key)
        assert (g.key, g.name, g.parent) == (key, "keyed", parent)

    def test_policy_retrieve(self, client: sy.Synnax) -> None:
        role = client.access.roles.create(sy.Role(name="deprecation"))
        policy = client.access.policies.create(
            sy.Policy(name="deprecation", actions=["retrieve"], role=role.key)
        )
        user = client.users.create(
            username=str(uuid4()), password="test", role=role.key
        )
        retrieve: Callable[..., list[sy.Policy]] = client.access.policies.retrieve
        with deprecated():
            matched = retrieve([policy.key])
        assert [p.key for p in matched] == [policy.key]
        with deprecated():
            matched = retrieve(None, [user.resource_id], False)
        assert [p.key for p in matched] == [policy.key]
        with deprecated():
            matched = retrieve(None, None, True)
        assert len(matched) > 0
        assert all(p.internal for p in matched)

    def test_task(self, client: sy.Synnax) -> None:
        tsk = client.tasks.create(name=str(uuid4()), type="pagerduty_alert")
        try:
            with deprecated():
                untyped(tsk.execute_command)("deprecation", None)
            with deprecated():
                untyped(client.tasks.maybe_assign_def_rack)(tsk.to_payload(), 0)
            with deprecated(), pytest.raises(TimeoutError):
                untyped(tsk.execute_command_sync)("deprecation", None, 0)
            starter = _StarterStopper(tsk)
            with deprecated(), pytest.raises(TimeoutError):
                untyped(starter.start)(0)
            with deprecated(), pytest.raises(TimeoutError):
                untyped(starter.stop)(0)
            with deprecated(), pytest.raises(TimeoutError), untyped(starter.run)(0):
                pass
        finally:
            client.tasks.delete(tsk.key)

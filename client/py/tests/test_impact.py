#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
import sys
from collections.abc import Callable
from uuid import uuid4

import pytest

import synnax as sy
from synnax.task.types_gen import resource_id as task_resource_id
from tests.conftest import HOST, PORT
from x.strings import random_name
from x.testutil import assert_eventually


def create_read_task(client: sy.Synnax, ch: sy.Channel) -> sy.Task:
    rack = client.racks.create(name=f"test_{random_name()}")
    device = client.devices.create(
        sy.Device(
            key=f"test_{random_name()}",
            name="plc",
            rack=rack.key,
            location="127.0.0.1:502",
            make="modbus",
            model="modbus",
        )
    )
    return client.tasks.create(
        name="read",
        type="modbus_read",
        rack=rack.key,
        config={
            "device": device.key,
            "channels": [{"type": "coil", "key": "coil", "channel": ch.key}],
        },
    )


def delete_of(ch: sy.Channel) -> sy.impact.DeleteChannelsChange:
    return sy.impact.DeleteChannelsChange(keys=[ch.key])


def assert_gone(retrieve: Callable[[], object]) -> None:
    """Raises AssertionError while retrieve still finds its record."""
    try:
        retrieve()
    except sy.NotFoundError:
        return
    raise AssertionError("the record still exists")


@pytest.mark.impact
class TestImpact:
    def test_plan_and_apply_with_a_pick(
        self, client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should plan a delete with a pick and apply it with the plan's token"""
        task = create_read_task(client, virtual_channel)
        record = task_resource_id(task.key)
        plan = client.impact.plan(delete_of(virtual_channel))
        assert [(str(e.record), e.effect) for e in plan.entries] == [
            (str(record), "restrict")
        ]
        assert plan.entries[0].default == "cancel"
        picks: dict[str, sy.impact.Choice] = {str(record): "delete"}
        picked = client.impact.plan(delete_of(virtual_channel), picks=picks)
        client.impact.apply(delete_of(virtual_channel), token=picked.token, picks=picks)
        assert_eventually(
            lambda: assert_gone(lambda: client.channels.retrieve(virtual_channel.key))
        )
        assert_eventually(
            lambda: assert_gone(lambda: client.tasks.retrieve(key=task.key))
        )

    def test_stale_token(self, client: sy.Synnax, virtual_channel: sy.Channel) -> None:
        """Should refuse a stale token with the plan as it is now"""
        stale = client.impact.plan(delete_of(virtual_channel))
        task = create_read_task(client, virtual_channel)
        with pytest.raises(sy.PlanChangedError) as raised:
            client.impact.apply(delete_of(virtual_channel), token=stale.token)
        now = sy.impact.plan_of(raised.value)
        assert [str(e.record) for e in now.entries] == [str(task_resource_id(task.key))]
        assert now.token != stale.token
        assert client.channels.retrieve(virtual_channel.key).key == virtual_channel.key

    def test_hidden(self, client: sy.Synnax, virtual_channel: sy.Channel) -> None:
        """Should count a record that the caller cannot see as hidden"""
        create_read_task(client, virtual_channel)
        role = client.access.roles.create(sy.Role(name=f"impact_{random_name()}"))
        client.access.policies.create(
            sy.Policy(
                name="impact",
                objects=[virtual_channel.resource_id],
                actions=["retrieve", "delete"],
                role=role.key,
            )
        )
        username, password = str(uuid4()), "impact"
        client.users.create(username=username, password=password, role=role.key)
        user_client = sy.Synnax(
            host=HOST, port=PORT, username=username, password=password, secure=False
        )
        plan = user_client.impact.plan(delete_of(virtual_channel))
        assert plan.entries == []
        assert plan.hidden == 1


@pytest.mark.impact
def test_plan_of_after_the_channel_module_loads_first() -> None:
    """Should decode a plan-changed error and parse its plan when the channel module
    loads before the exceptions and impact modules
    """
    script = "\n".join(
        [
            "import synnax.channel",
            "from synnax.exceptions import PlanChangedError, create_registry",
            "from synnax.impact import plan_of",
            "from x.exceptions import ExceptionPayload",
            'plan = \'{"entries": [], "kept": 0, "hidden": 1, "token": "t",\'',
            'plan += \'"change": {"type": "delete_channels", "keys": [1]}}\'',
            "err = create_registry().decode(",
            "    ExceptionPayload(type='sy.impact.plan_changed', data=plan)",
            ")",
            "assert isinstance(err, PlanChangedError), err",
            "assert plan_of(err).hidden == 1",
        ]
    )
    subprocess.run([sys.executable, "-c", script], check=True)

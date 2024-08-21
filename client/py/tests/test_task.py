#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy
from uuid import uuid4


@pytest.mark.task
class TestTaskClient:
    def test_create_single(self, client: sy.Synnax):
        task = client.hardware.tasks.create(name="test", type="test")
        assert task.key != 0

    def test_create_multiple(self, client: sy.Synnax):
        t1 = sy.Task(name="test1", type="test")
        t2 = sy.Task(name="test2", type="test")
        tasks = client.hardware.tasks.create(tasks=[t1, t2])
        assert len(tasks) == 2
        assert tasks[0].name == "test1"
        assert tasks[1].name == "test2"

    def test_retrieve_by_name(self, client: sy.Synnax):
        name = str(uuid4())
        task = client.hardware.tasks.create(name=name, type="test")
        res = client.hardware.tasks.retrieve(name=name)
        assert res.name == name
        assert res.key == task.key

    def test_retrieve_by_type(self, client: sy.Synnax):
        type = str(uuid4())
        task = client.hardware.tasks.create(type=type)
        res = client.hardware.tasks.retrieve(type=type)
        assert res.type == type
        assert res.key == task.key

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


@pytest.mark.task
class TestTaskClient:
    def test_create_single(self, client: sy.Synnax):
        r = client.hardware.racks.create(name="dog")
        task = client.hardware.tasks.create(
            name="test",
            type="test",
            rack=r.key,
            config="{}"
        )
        assert task.key != 0

    def test_create_multiple(self, client: sy.Synnax):
        r = client.hardware.racks.create(name="dog")
        t1 = sy.Task(rack=r.key, name="test1", type="test", config="{}")
        t2 = sy.Task(rack=r.key, name="test2", type="test", config="{}")
        tasks = client.hardware.tasks.create(tasks=[t1, t2])
        assert len(tasks) == 2
        assert tasks[0].name == "test1"
        assert tasks[1].name == "test2"

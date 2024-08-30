#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import json
import time

import pytest
import synnax as sy
import threading
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

    def test_execute_command_sync(self, client: sy.Synnax):
        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_cmd") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_state") as w:
                    ev.set()
                    f = s.read(timeout=1)
                    cmd = f["sy_task_cmd"][0]
                    w.write(
                        "sy_task_state",
                        [
                            {
                                "key": cmd["key"],
                                "task": cmd["task"],
                                "variant": "success",
                                "details": {"message": "Command executed."},
                            }
                        ],
                    )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        tsk = client.hardware.tasks.create(name="test", type="test")
        ev.wait()
        tsk.execute_command_sync("test", {"key": "value"})
        t.join()

    def test_task_configure_success(self, client: sy.Synnax):
        """Should not throw an error when the task is configured successfully."""

        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_state") as w:
                    ev.set()
                    f = s.read(timeout=2)
                    key = f["sy_task_set"][0]
                    w.write(
                        "sy_task_state",
                        [
                            {
                                "task": int(key),
                                "variant": "success",
                                "details": {"message": "Task configured."},
                            }
                        ],
                    )

        tsk = sy.Task()
        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        client.hardware.tasks.configure(tsk)
        t.join()

    def test_task_configure_invalid_config(self, client: sy.Synnax):
        """Should throw an error when the driver responds with an error"""

        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                with client.open_writer(sy.TimeStamp.now(), "sy_task_state") as w:
                    ev.set()
                    f = s.read(timeout=1)
                    key = f["sy_task_set"][0]
                    w.write(
                        "sy_task_state",
                        [
                            {
                                "task": int(key),
                                "variant": "error",
                                "details": {"message": "Invalid Configuration."},
                            }
                        ],
                    )

        tsk = sy.Task()
        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        with pytest.raises(sy.ConfigurationError, match="Invalid Configuration."):
            client.hardware.tasks.configure(tsk)
        t.join()

    def test_task_configure_timeout(self, client: sy.Synnax):
        """Should throw an error when the task is not configured within the timeout."""
        tsk = sy.Task()
        with pytest.raises(TimeoutError):
            client.hardware.tasks.configure(tsk, timeout=0.1)

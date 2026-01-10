#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from uuid import uuid4

import pytest

import synnax as sy


@pytest.mark.task
class TestTaskClient:
    def test_create_single(self, client: sy.Synnax):
        task = client.tasks.create(name="test", type="test")
        print(task.key)
        assert task.key != 0

    def test_create_multiple(self, client: sy.Synnax):
        t1 = sy.Task(name="test1", type="test")
        t2 = sy.Task(name="test2", type="test")
        tasks = client.tasks.create(tasks=[t1, t2])
        assert len(tasks) == 2
        assert tasks[0].name == "test1"
        assert tasks[1].name == "test2"

    def test_retrieve_by_name(self, client: sy.Synnax):
        name = str(uuid4())
        task = client.tasks.create(name=name, type="test")
        res = client.tasks.retrieve(name=name)
        assert res.name == name
        assert res.key == task.key

    def test_retrieve_by_type(self, client: sy.Synnax):
        type = str(uuid4())
        task = client.tasks.create(type=type)
        res = client.tasks.retrieve(type=type)
        assert res.type == type
        assert res.key == task.key

    def test_execute_command_sync(self, client: sy.Synnax):
        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_cmd") as s:
                ev.set()
                f = s.read(timeout=1)
                cmd = f["sy_task_cmd"][0]
                client.statuses.set(
                    sy.TaskStatus(
                        key=str(sy.task.ontology_id(cmd["task"])),
                        variant=sy.status.SUCCESS_VARIANT,
                        message="Command executed.",
                        details=sy.TaskStatusDetails(
                            task=int(cmd["task"]), cmd=cmd["key"]
                        ),
                    )
                )

        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        tsk = client.tasks.create(name="test", type="test")
        ev.wait()
        tsk.execute_command_sync("test", {"key": "value"})
        t.join()

    def test_task_configure_success(self, client: sy.Synnax):
        """Should not throw an error when the task is configured successfully."""

        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                ev.set()
                f = s.read(timeout=2)
                key = f["sy_task_set"][0]
                client.statuses.set(
                    sy.TaskStatus(
                        key=str(sy.task.ontology_id(int(key))),
                        variant=sy.status.SUCCESS_VARIANT,
                        message="Task configured.",
                        details=sy.TaskStatusDetails(task=int(key)),
                    )
                )

        tsk = sy.Task()
        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        client.tasks.configure(tsk)
        t.join()

    def test_task_configure_invalid_config(self, client: sy.Synnax):
        """Should throw an error when the driver responds with an error"""

        def driver(ev: threading.Event):
            with client.open_streamer("sy_task_set") as s:
                ev.set()
                f = s.read(timeout=1)
                key = f["sy_task_set"][0]
                client.statuses.set(
                    sy.TaskStatus(
                        key=str(sy.task.ontology_id(int(key))),
                        variant=sy.status.ERROR_VARIANT,
                        message="Invalid Configuration.",
                        details=sy.TaskStatusDetails(task=int(key)),
                    )
                )

        tsk = sy.Task()
        ev = threading.Event()
        t = threading.Thread(target=driver, args=(ev,))
        t.start()
        ev.wait()
        with pytest.raises(sy.ConfigurationError, match="Invalid Configuration."):
            client.tasks.configure(tsk)
        t.join()

    def test_task_configure_timeout(self, client: sy.Synnax):
        """Should throw an error when the task is not configured within the timeout."""
        tsk = sy.Task()
        with pytest.raises(TimeoutError):
            client.tasks.configure(tsk, timeout=0.1)

    def test_list_tasks(self, client: sy.Synnax):
        """Should list all tasks on the default rack."""
        # Create some tasks
        task1 = client.tasks.create(name=str(uuid4()), type="test1")
        task2 = client.tasks.create(name=str(uuid4()), type="test2")

        # List all tasks
        tasks = client.tasks.list()

        # Should contain at least the tasks we just created
        task_keys = [t.key for t in tasks]
        assert task1.key in task_keys
        assert task2.key in task_keys

    def test_copy_task(self, client: sy.Synnax):
        """Should copy a task with a new name."""
        # Create an original task
        original_name = str(uuid4())
        original = client.tasks.create(
            name=original_name, type="test", config='{"foo": "bar"}'
        )

        # Copy the task
        copy_name = str(uuid4())
        copied = client.tasks.copy(
            key=original.key,
            name=copy_name,
        )

        # Verify the copy
        assert copied.key != original.key
        assert copied.name == copy_name
        assert copied.type == original.type
        assert copied.config == original.config
